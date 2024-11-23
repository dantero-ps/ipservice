package repository

import (
	"context"
	"fmt"
	"ipservice/internal/model"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisRepository struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisRepository(client *redis.Client, logger *zap.Logger) *RedisRepository {
	return &RedisRepository{
		client: client,
		logger: logger,
	}
}

func (r *RedisRepository) SetCountry(ctx context.Context, ip, countryCode string) error {
	err := r.client.Set(ctx, "ip:"+ip, countryCode, 24*time.Hour).Err()
	if err != nil {
		r.logger.Error("failed to set country in cache",
			zap.String("ip", ip),
			zap.Error(err))
	}
	return err
}

func (r *RedisRepository) GetCountry(ctx context.Context, ip string) (string, error) {
	countryCode, err := r.client.Get(ctx, "ip:"+ip).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		r.logger.Error("failed to get country from cache",
			zap.String("ip", ip),
			zap.Error(err))
		return "", err
	}
	return countryCode, nil
}

func ipToInt(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func (r *RedisRepository) CacheIPRanges(ctx context.Context, ranges []model.IPRange) error {
	pipe := r.client.Pipeline()

	// Clear existing data
	pipe.Del(ctx, "ipranges")

	// Store ranges sorted by start IP
	for _, ipRange := range ranges {
		startIP := ipToInt(ipRange.Network.IP)
		// Store as: startIP countryCode|mask
		pipe.ZAdd(ctx, "ipranges", redis.Z{
			Score:  float64(startIP),
			Member: fmt.Sprintf("%s|%s", ipRange.CountryCode, ipRange.Network.Mask.String()),
		})
	}

	pipe.Expire(ctx, "ipranges", 24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisRepository) GetCachedRange(ctx context.Context, ip net.IP) (string, error) {
	ipInt := ipToInt(ip)

	// Find the largest range start that's less than or equal to our IP
	ranges, err := r.client.ZRevRangeByScore(ctx, "ipranges", &redis.ZRangeBy{
		Min:    "0",
		Max:    fmt.Sprintf("%d", ipInt),
		Offset: 0,
		Count:  1,
	}).Result()

	if err != nil {
		return "", err
	}

	if len(ranges) == 0 {
		return "", nil
	}

	// Parse the stored range
	parts := strings.Split(ranges[0], "|")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid range format")
	}

	countryCode := parts[0]
	mask := parts[1]

	// Verify IP is in range
	_, network, err := net.ParseCIDR(fmt.Sprintf("%s/%s", ip.String(), mask))
	if err != nil {
		return "", err
	}

	if network.Contains(ip) {
		return countryCode, nil
	}

	return "", nil
}
