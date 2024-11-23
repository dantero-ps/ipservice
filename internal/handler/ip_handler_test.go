package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"ipservice/internal/model"
)

type mockIPService struct {
	lookupIPFunc func(ctx context.Context, ip string) (*model.IPResponse, error)
}

func (m *mockIPService) LookupIP(ctx context.Context, ip string) (*model.IPResponse, error) {
	return m.lookupIPFunc(ctx, ip)
}

func TestHandler_LookupIP(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		mockResponse *model.IPResponse
		mockError    error
		expectedCode int
		expectedBody string
	}{
		{
			name: "success",
			path: "/api/v1/lookup/8.8.8.8",
			mockResponse: &model.IPResponse{
				IP:          "8.8.8.8",
				CountryCode: "US",
			},
			mockError:    nil,
			expectedCode: 200,
			expectedBody: `{"ip":"8.8.8.8","country_code":"US"}`,
		},
		{
			name:         "invalid ip",
			path:         "/api/v1/lookup/invalid",
			mockResponse: nil,
			mockError:    fmt.Errorf("invalid IP address: invalid"),
			expectedCode: 400,
			expectedBody: `{"message":"Invalid IP address format: invalid"}`,
		},
	}

	logger, _ := zap.NewDevelopment()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockIPService{
				lookupIPFunc: func(ctx context.Context, ip string) (*model.IPResponse, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			h := NewHandler(mockService, logger)
			app := fiber.New()
			h.RegisterRoutes(app)

			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.expectedCode {
				t.Errorf("expected status code %d, got %d", tt.expectedCode, resp.StatusCode)
			}

			var body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}

			expectedBody := make(map[string]interface{})
			if err := json.Unmarshal([]byte(tt.expectedBody), &expectedBody); err != nil {
				t.Fatal(err)
			}

			if !jsonEqual(body, expectedBody) {
				t.Errorf("expected body %v, got %v", expectedBody, body)
			}
		})
	}
}

func jsonEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

func TestHandler_HealthCheck(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewHandler(nil, logger)

	app := fiber.New()
	h.RegisterRoutes(app)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", body["status"])
	}
}
