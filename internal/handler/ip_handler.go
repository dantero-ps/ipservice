package handler

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"ipservice/internal/model"
	"strings"
)

type IPService interface {
	LookupIP(ctx context.Context, ip string) (*model.IPResponse, error)
}

type Handler struct {
	service IPService
	logger  *zap.Logger
}

func NewHandler(service IPService, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	app.Get("/api/v1/lookup/:ip", h.LookupIP)
	app.Get("/api/v1/health", h.HealthCheck)
}

func (h *Handler) LookupIP(c *fiber.Ctx) error {
	ip := c.Params("ip")
	if ip == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.Error{
			Message: "IP address is required",
		})
	}

	result, err := h.service.LookupIP(c.Context(), ip)
	if err != nil {
		if strings.Contains(err.Error(), "invalid IP address") {
			return c.Status(fiber.StatusBadRequest).JSON(model.Error{
				Message: fmt.Sprintf("Invalid IP address format: %s", ip),
			})
		}

		h.logger.Error("IP lookup failed",
			zap.String("ip", ip),
			zap.Error(err))

		return c.Status(fiber.StatusInternalServerError).JSON(model.Error{
			Message: "Failed to lookup IP address",
		})
	}

	if result.CountryCode == "ZZ" {
		return c.Status(fiber.StatusNotFound).JSON(model.Error{
			Message: "No country information found for this IP",
		})
	}

	return c.JSON(result)
}

func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
	})
}
