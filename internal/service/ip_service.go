package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"ipservice/internal/config"
	"ipservice/internal/model"
)

type Repository interface {
	SaveIPRanges(ctx context.Context, ranges []model.IPRange) error
	FindCountryForIP(ctx context.Context, ip net.IP) (string, error)
	ClearIPRanges(ctx context.Context) error
	GetRangesCount(ctx context.Context) (int64, error)
}

type Cache interface {
	SetCountry(ctx context.Context, ip, countryCode string) error
	GetCountry(ctx context.Context, ip string) (string, error)
	CacheIPRanges(ctx context.Context, ranges []model.IPRange) error
	GetCachedRange(ctx context.Context, ip net.IP) (string, error)
}

type IPService struct {
	repo      Repository
	cache     Cache
	rirSvc    *RIRService
	config    *config.Config
	logger    *zap.Logger
	updateMux sync.Mutex
}

func NewIPService(
	repo Repository,
	cache Cache,
	rirSvc *RIRService,
	config *config.Config,
	logger *zap.Logger,
) *IPService {
	return &IPService{
		repo:   repo,
		cache:  cache,
		rirSvc: rirSvc,
		config: config,
		logger: logger,
	}
}

func (s *IPService) Start(ctx context.Context) error {
	// Quick check if data exists
	exists, err := s.checkDataExists(ctx)
	if err != nil {
		return fmt.Errorf("checking data existence: %w", err)
	}

	if !exists {
		s.logger.Info("No IP ranges found in database, performing initial load")
		if err := s.UpdateIPRanges(ctx); err != nil {
			return fmt.Errorf("initial IP ranges update failed: %w", err)
		}
	} else {
		s.logger.Info("Existing IP ranges found in database, skipping initial load")
	}

	// Schedule periodic updates
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := s.UpdateIPRanges(ctx); err != nil {
					s.logger.Error("scheduled IP ranges update failed", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

func (s *IPService) UpdateIPRanges(ctx context.Context) error {
	s.updateMux.Lock()
	defer s.updateMux.Unlock()

	var allRanges []model.IPRange
	var errors []error
	totalStats := struct {
		TotalRanges   int
		IPv4Ranges    int
		IPv6Ranges    int
		SkippedRanges int
		ParseErrors   int
	}{}

	s.logger.Info("Starting IP ranges update")

	for _, rir := range s.config.RIRs {
		ranges, stats, err := s.rirSvc.FetchIPRanges(ctx, rir.URL)
		if err != nil {
			s.logger.Error("failed to fetch IP ranges",
				zap.String("rir", rir.Name),
				zap.Error(err))
			errors = append(errors, fmt.Errorf("%s: %w", rir.Name, err))
			continue
		}
		allRanges = append(allRanges, ranges...)

		// Update total statistics
		totalStats.IPv4Ranges += stats.IPv4Count
		totalStats.IPv6Ranges += stats.IPv6Count
		totalStats.SkippedRanges += stats.SkippedCount
		totalStats.ParseErrors += stats.ParseErrors
		totalStats.TotalRanges += len(ranges)

		s.logger.Info("Fetched IP ranges",
			zap.String("rir", rir.Name),
			zap.Int("total_ranges", len(ranges)),
			zap.Int("ipv4_ranges", stats.IPv4Count),
			zap.Int("ipv6_ranges", stats.IPv6Count),
			zap.Int("skipped_ranges", stats.SkippedCount),
			zap.Int("parse_errors", stats.ParseErrors))
	}

	if len(allRanges) == 0 {
		return fmt.Errorf("no IP ranges fetched: %v", errors)
	}

	s.logger.Info("Total statistics",
		zap.Int("total_ranges", totalStats.TotalRanges),
		zap.Int("ipv4_ranges", totalStats.IPv4Ranges),
		zap.Int("ipv6_ranges", totalStats.IPv6Ranges),
		zap.Int("skipped_ranges", totalStats.SkippedRanges),
		zap.Int("parse_errors", totalStats.ParseErrors))

	if err := s.repo.ClearIPRanges(ctx); err != nil {
		return fmt.Errorf("clearing existing IP ranges: %w", err)
	}

	startTime := time.Now()
	err := s.repo.SaveIPRanges(ctx, allRanges)
	if err != nil {
		s.logger.Error("Failed to save IP ranges",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)))
		return err
	}

	// Cache the ranges in Redis
	if err := s.cache.CacheIPRanges(ctx, allRanges); err != nil {
		s.logger.Error("Failed to cache IP ranges",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)))
		// Don't return error as database update was successful
	}

	s.logger.Info("Successfully saved IP ranges",
		zap.Int("total_ranges", len(allRanges)),
		zap.Duration("duration", time.Since(startTime)))

	return nil
}

func (s *IPService) LookupIP(ctx context.Context, ipStr string) (*model.IPResponse, error) {
	// Try direct IP cache first
	if countryCode, err := s.cache.GetCountry(ctx, ipStr); err == nil && countryCode != "" {
		return &model.IPResponse{
			IP:          ipStr,
			CountryCode: countryCode,
		}, nil
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Try cached ranges
	if countryCode, err := s.cache.GetCachedRange(ctx, ip); err == nil && countryCode != "" {
		// Cache the specific IP for faster future lookups
		if err := s.cache.SetCountry(ctx, ipStr, countryCode); err != nil {
			s.logger.Warn("failed to cache IP lookup result",
				zap.String("ip", ipStr),
				zap.Error(err))
		}
		return &model.IPResponse{
			IP:          ipStr,
			CountryCode: countryCode,
		}, nil
	}

	// Fall back to database
	countryCode, err := s.repo.FindCountryForIP(ctx, ip)
	if err != nil {
		return nil, err
	}

	// Don't cache unknown results
	if countryCode != "ZZ" {
		if err := s.cache.SetCountry(ctx, ipStr, countryCode); err != nil {
			s.logger.Warn("failed to cache IP lookup result",
				zap.String("ip", ipStr),
				zap.Error(err))
		}
	}

	return &model.IPResponse{
		IP:          ipStr,
		CountryCode: countryCode,
	}, nil
}

func (s *IPService) checkDataExists(ctx context.Context) (bool, error) {
	count, err := s.repo.GetRangesCount(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
