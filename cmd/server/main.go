package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amansharma/config-service/internal/database"
	"github.com/amansharma/config-service/internal/handler"
	"github.com/amansharma/config-service/internal/repository"
	"github.com/amansharma/config-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	_ "github.com/lib/pq"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg := loadConfig(logger)

	db, err := connectDB(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	logger.Info().Msg("database connection established")

	if err := database.RunMigrations(cfg.dsn(), logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}

	repo := repository.NewConfigRepository(db)
	svc := service.NewConfigService(repo)
	h := handler.NewConfigHandler(svc, logger)

	r := chi.NewRouter()
	r.Use(handler.RequestLogger(logger))

	r.Get("/ping", h.Ping)
	r.Get("/healthz", h.Healthz)
	r.Get("/configs/{id}", h.GetConfig)
	r.Post("/configs", h.UpsertConfig)
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         ":" + cfg.serverPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.serverPort).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("server forced to shutdown")
	}
	logger.Info().Msg("server stopped")
}

type appConfig struct {
	serverPort string
	dbHost     string
	dbPort     string
	dbUser     string
	dbPassword string
	dbName     string
	dbSSLMode  string
}

func (c appConfig) dsn() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.dbUser, c.dbPassword, c.dbHost, c.dbPort, c.dbName, c.dbSSLMode)
}

func loadConfig(logger zerolog.Logger) appConfig {
	cfg := appConfig{
		serverPort: envOrDefault("SERVER_PORT", "8080"),
		dbHost:     envOrDefault("DB_HOST", "localhost"),
		dbPort:     envOrDefault("DB_PORT", "5432"),
		dbUser:     envOrDefault("DB_USER", "configservice"),
		dbPassword: os.Getenv("DB_PASSWORD"),
		dbName:     envOrDefault("DB_NAME", "configservice"),
		dbSSLMode:  envOrDefault("DB_SSLMODE", "disable"),
	}

	if cfg.dbPassword == "" {
		logger.Fatal().Msg("DB_PASSWORD environment variable is required")
	}

	logger.Info().
		Str("server_port", cfg.serverPort).
		Str("db_host", cfg.dbHost).
		Str("db_port", cfg.dbPort).
		Str("db_user", cfg.dbUser).
		Str("db_name", cfg.dbName).
		Str("db_sslmode", cfg.dbSSLMode).
		Msg("configuration loaded")

	return cfg
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func connectDB(cfg appConfig, logger zerolog.Logger) (*sqlx.DB, error) {
	maxRetries := 5
	retryInterval := 3 * time.Second

	for i := 1; i <= maxRetries; i++ {
		db, err := sqlx.Connect("postgres", cfg.dsn())
		if err == nil {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			return db, nil
		}

		logger.Warn().
			Err(err).
			Int("attempt", i).
			Int("max_retries", maxRetries).
			Dur("retry_in", retryInterval).
			Msg("failed to connect to database, retrying")

		if i < maxRetries {
			time.Sleep(retryInterval)
		}
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts", maxRetries)
}
