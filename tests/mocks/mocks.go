package mocks

import (
	"context"
	"ipservice/internal/model"
	"net"
)

type MockRepository struct {
	SaveIPRangesFunc     func(ctx context.Context, ranges []model.IPRange) error
	FindCountryForIPFunc func(ctx context.Context, ip net.IP) (string, error)
	ClearIPRangesFunc    func(ctx context.Context) error
	GetRangesCountFunc   func(ctx context.Context) (int64, error)
}

func (m *MockRepository) SaveIPRanges(ctx context.Context, ranges []model.IPRange) error {
	return m.SaveIPRangesFunc(ctx, ranges)
}

func (m *MockRepository) FindCountryForIP(ctx context.Context, ip net.IP) (string, error) {
	return m.FindCountryForIPFunc(ctx, ip)
}

func (m *MockRepository) ClearIPRanges(ctx context.Context) error {
	return m.ClearIPRangesFunc(ctx)
}

func (m *MockRepository) GetRangesCount(ctx context.Context) (int64, error) {
	return m.GetRangesCountFunc(ctx)
}

type MockCache struct {
	SetCountryFunc     func(ctx context.Context, ip, countryCode string) error
	GetCountryFunc     func(ctx context.Context, ip string) (string, error)
	CacheIPRangesFunc  func(ctx context.Context, ranges []model.IPRange) error
	GetCachedRangeFunc func(ctx context.Context, ip net.IP) (string, error)
}

func (m *MockCache) SetCountry(ctx context.Context, ip, countryCode string) error {
	return m.SetCountryFunc(ctx, ip, countryCode)
}

func (m *MockCache) GetCountry(ctx context.Context, ip string) (string, error) {
	return m.GetCountryFunc(ctx, ip)
}

func (m *MockCache) CacheIPRanges(ctx context.Context, ranges []model.IPRange) error {
	return m.CacheIPRangesFunc(ctx, ranges)
}

func (m *MockCache) GetCachedRange(ctx context.Context, ip net.IP) (string, error) {
	return m.GetCachedRangeFunc(ctx, ip)
}
