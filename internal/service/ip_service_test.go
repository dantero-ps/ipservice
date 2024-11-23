package service

import (
	"context"
	"go.uber.org/zap"
	"ipservice/internal/config"
	"ipservice/internal/model"
	"ipservice/tests/mocks"
	"net"
	"testing"
)

func TestIPService_LookupIP(t *testing.T) {
	tests := []struct {
		name          string
		ip            string
		cacheResponse string
		cacheError    error
		cachedRange   string
		rangeError    error
		repoResponse  string
		repoError     error
		expected      *model.IPResponse
		expectedError bool
	}{
		{
			name:          "cache hit",
			ip:            "8.8.8.8",
			cacheResponse: "US",
			expected: &model.IPResponse{
				IP:          "8.8.8.8",
				CountryCode: "US",
			},
		},
		{
			name:          "cache miss, cached range hit",
			ip:            "8.8.8.8",
			cacheResponse: "",
			cachedRange:   "US",
			expected: &model.IPResponse{
				IP:          "8.8.8.8",
				CountryCode: "US",
			},
		},
		{
			name:          "cache miss, repo hit",
			ip:            "8.8.8.8",
			cacheResponse: "",
			cachedRange:   "",
			repoResponse:  "US",
			expected: &model.IPResponse{
				IP:          "8.8.8.8",
				CountryCode: "US",
			},
		},
		{
			name:          "invalid ip",
			ip:            "invalid",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := &mocks.MockCache{
				GetCountryFunc: func(ctx context.Context, ip string) (string, error) {
					return tt.cacheResponse, tt.cacheError
				},
				SetCountryFunc: func(ctx context.Context, ip, countryCode string) error {
					return nil
				},
				GetCachedRangeFunc: func(ctx context.Context, ip net.IP) (string, error) {
					return tt.cachedRange, tt.rangeError
				},
			}

			mockRepo := &mocks.MockRepository{
				FindCountryForIPFunc: func(ctx context.Context, ip net.IP) (string, error) {
					return tt.repoResponse, tt.repoError
				},
			}

			logger, _ := zap.NewDevelopment()
			cfg := &config.Config{}
			rirSvc := NewRIRService(logger)
			svc := NewIPService(mockRepo, mockCache, rirSvc, cfg, logger)

			result, err := svc.LookupIP(context.Background(), tt.ip)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.IP != tt.expected.IP || result.CountryCode != tt.expected.CountryCode {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
