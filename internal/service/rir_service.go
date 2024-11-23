package service

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"ipservice/internal/model"
)

type RIRService struct {
	logger *zap.Logger
	client *http.Client
}

func NewRIRService(logger *zap.Logger) *RIRService {
	return &RIRService{
		logger: logger,
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: true,
				MaxConnsPerHost:    100,
				DisableKeepAlives:  false,
				ForceAttemptHTTP2:  true,
			},
		},
	}
}

type RIRStats struct {
	IPv4Count    int
	IPv6Count    int
	SkippedCount int
	ParseErrors  int
}

func (s *RIRService) FetchIPRanges(ctx context.Context, url string) ([]model.IPRange, RIRStats, error) {
	maxRetries := 3
	var lastErr error
	var stats RIRStats

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * 5 * time.Second
			time.Sleep(delay)
		}

		ranges, stats, err := s.fetchWithTimeout(ctx, url)
		if err == nil {
			return ranges, stats, nil
		}

		lastErr = err
		s.logger.Warn("Failed to fetch RIR data, retrying...",
			zap.String("url", url),
			zap.Int("attempt", attempt+1),
			zap.Error(err))
	}

	return nil, stats, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (s *RIRService) fetchWithTimeout(ctx context.Context, url string) ([]model.IPRange, RIRStats, error) {
	startTime := time.Now()
	var stats RIRStats

	s.logger.Info("Starting RIR data fetch", zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, stats, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "IPLocator/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, stats, fmt.Errorf("fetching RIR data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, stats, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	s.logger.Info("Successfully downloaded RIR data",
		zap.String("url", url),
		zap.Duration("download_time", time.Since(startTime)))

	var ranges []model.IPRange
	scanner := bufio.NewScanner(resp.Body)
	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineCount := 0
	parseStartTime := time.Now()

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		if strings.HasPrefix(line, "#") || len(line) == 0 {
			stats.SkippedCount++
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			stats.SkippedCount++
			continue
		}

		if parts[1] == "*" || (parts[6] != "allocated" && parts[6] != "assigned") {
			stats.SkippedCount++
			continue
		}

		if parts[2] != "ipv4" && parts[2] != "ipv6" {
			stats.SkippedCount++
			continue
		}

		ipRange, err := s.parseIPRange(parts)
		if err != nil {
			stats.ParseErrors++
			s.logger.Debug("failed to parse IP range",
				zap.String("line", line),
				zap.Error(err))
			continue
		}

		if ipRange.Version == 4 {
			stats.IPv4Count++
		} else {
			stats.IPv6Count++
		}

		ranges = append(ranges, ipRange)
	}

	if err := scanner.Err(); err != nil {
		return nil, stats, fmt.Errorf("reading RIR data: %w", err)
	}

	s.logger.Info("Finished parsing RIR data",
		zap.String("url", url),
		zap.Int("total_lines", lineCount),
		zap.Int("ipv4_ranges", stats.IPv4Count),
		zap.Int("ipv6_ranges", stats.IPv6Count),
		zap.Int("skipped_lines", stats.SkippedCount),
		zap.Int("parse_errors", stats.ParseErrors),
		zap.Duration("parse_time", time.Since(parseStartTime)),
		zap.Duration("total_time", time.Since(startTime)))

	return ranges, stats, nil
}

func (s *RIRService) parseIPRange(parts []string) (model.IPRange, error) {
	var ipRange model.IPRange

	countryCode := parts[1]
	startIP := parts[3]

	var network *net.IPNet

	switch parts[2] {
	case "ipv4":
		value, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			return ipRange, err
		}

		// Convert value to CIDR prefix length
		prefixLen := 32 - int(math.Log2(float64(value)))
		network, err = parseCIDR(startIP, prefixLen)
		if err != nil {
			return ipRange, err
		}
		ipRange.Version = 4

	case "ipv6":
		prefixLen, err := strconv.Atoi(parts[4])
		if err != nil {
			return ipRange, err
		}
		network, err = parseCIDR(startIP, prefixLen)
		if err != nil {
			return ipRange, err
		}
		ipRange.Version = 6
	}

	ipRange.Network = *network
	ipRange.CountryCode = countryCode

	return ipRange, nil
}

func parseCIDR(ip string, prefixLen int) (*net.IPNet, error) {
	cidr := fmt.Sprintf("%s/%d", ip, prefixLen)
	_, network, err := net.ParseCIDR(cidr)
	return network, err
}
