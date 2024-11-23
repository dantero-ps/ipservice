package main

import (
	"context"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"ipservice/internal/config"
	"ipservice/internal/handler"
	"ipservice/internal/repository"
	"ipservice/internal/service"
)

var (
	lastLogTime atomic.Value
	logMutex    sync.Mutex
)

func init() {
	lastLogTime.Store(time.Now())
}

func main() {
	// Initialize logger
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := logConfig.Build()
	defer logger.Sync()

	logger.Info("Starting up server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize PostgreSQL connection
	db, err := sqlx.Connect("postgres", cfg.PostgresURL)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Initialize Redis connection
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}

	redisClient := redis.NewClient(opt)
	defer redisClient.Close()

	// Initialize repositories
	postgresRepo := repository.NewPostgresRepository(db, logger)
	redisRepo := repository.NewRedisRepository(redisClient, logger)

	// Initialize services
	rirService := service.NewRIRService(logger)
	ipService := service.NewIPService(
		postgresRepo,
		redisRepo,
		rirService,
		cfg,
		logger,
	)

	// Start IP service background tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := ipService.Start(ctx); err != nil {
		logger.Fatal("Failed to start IP service", zap.Error(err))
	}

	// Initialize HTTP server
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		// Always log errors and slow requests
		if err != nil || latency > 100*time.Millisecond || c.Response().StatusCode() != 200 {
			logger.Info("request",
				zap.Int("status", c.Response().StatusCode()),
				zap.Duration("latency", latency),
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Error(err),
			)
			return err
		}

		// Check if 10 seconds have passed since last log
		last := lastLogTime.Load().(time.Time)
		if time.Since(last) >= 10*time.Second {
			logMutex.Lock()
			// Double-check after acquiring lock
			if time.Since(last) >= 10*time.Second {
				logger.Info("sampled_request",
					zap.Int("status", c.Response().StatusCode()),
					zap.Duration("latency", latency),
					zap.String("method", c.Method()),
					zap.String("path", c.Path()),
				)
				lastLogTime.Store(time.Now())
			}
			logMutex.Unlock()
		}

		return err
	})

	// Initialize and register handlers
	h := handler.NewHandler(ipService, logger)
	h.RegisterRoutes(app)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := app.Listen(cfg.ServerPort); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	<-sigChan
	logger.Info("Shutting down server...")

	if err := app.Shutdown(); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
	}
}
