package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"erp-backend/internal/app"
	httpapi "erp-backend/internal/http"
)

func main() {
	// Set default timezone to Europe/Kyiv (GMT+3 / GMT+2)
	if os.Getenv("TZ") == "" {
		if loc, err := time.LoadLocation("Europe/Kyiv"); err == nil {
			time.Local = loc
		}
	}

	addr := envOrDefault("APP_ADDR", ":8080")
	jwtSecret := envOrDefault("JWT_SECRET", "local-dev-secret")
	databaseURL := os.Getenv("DATABASE_URL")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	store := app.NewStore()
	if databaseURL != "" {
		db, err := app.InitPostgres(databaseURL)
		if err != nil {
			log.Fatalf("cannot init postgres: %v", err)
		}
		defer db.Close()
		store = app.NewStoreWithDB(db)
	}
	store.SetNotificationSender(app.NewProviderNotifierFromEnv())
	store.SetReceiptSender(app.NewCheckboxReceiptSenderFromEnv())
	// Override with DB-stored config if available (takes priority over env vars)
	store.RebuildNotificationSenderFromDB()

	backgroundJobsAutoRunEnabled := envBoolOrDefault("BACKGROUND_JOBS_AUTORUN_ENABLED", true)
	backgroundJobsAutoRunInterval := envDurationOrDefault("BACKGROUND_JOBS_AUTORUN_INTERVAL", 30*time.Second)
	var scheduler *app.BackgroundJobScheduler
	if backgroundJobsAutoRunEnabled {
		scheduler = app.NewBackgroundJobScheduler(store, logger, backgroundJobsAutoRunInterval)
		scheduler.Start()
		defer scheduler.Stop()
		logger.Info(
			"background jobs scheduler enabled",
			"interval",
			backgroundJobsAutoRunInterval.String(),
		)
	}

	tokens := app.NewTokenManager(jwtSecret)
	server := httpapi.NewServer(store, tokens, logger)

	log.Printf("backend listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Router()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
