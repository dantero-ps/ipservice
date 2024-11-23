package service

import (
	"context"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRIRService_FetchIPRanges(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		responseCode  int
		expectedError bool
	}{
		{
			name: "valid response",
			response: `2|US|ipv4|192.168.0.0|65536|20100101|allocated
2|CA|ipv6|2001:db8::|32|20100101|allocated`,
			responseCode:  http.StatusOK,
			expectedError: false,
		},
		{
			name:          "server error",
			responseCode:  http.StatusInternalServerError,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			logger, _ := zap.NewDevelopment()
			service := NewRIRService(logger)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			ranges, _, err := service.FetchIPRanges(ctx, server.URL)

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

			if len(ranges) == 0 {
				t.Error("expected non-empty ranges")
			}
		})
	}
}
