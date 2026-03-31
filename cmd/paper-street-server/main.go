package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/h4ribote/Paper-Street/internal/api"
	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/db"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	store := createStore()
	engine := engine.NewEngine(store)
	apiKeys := loadAPIKeys()
	handler := api.NewRouter(engine, apiKeys, store)
	newsCtx, newsCancel := context.WithCancel(context.Background())
	newsConfig := api.DefaultNewsEngineConfig()
	newsConfig.Interval = envDuration("NEWS_INTERVAL", newsConfig.Interval)
	newsConfig.BaseQuantity = envInt64("NEWS_BASE_QUANTITY", newsConfig.BaseQuantity)
	newsConfig.MinConfidence = envFloat64("NEWS_MIN_CONFIDENCE", newsConfig.MinConfidence)
	newsConfig.ImpactFactor = envFloat64("NEWS_IMPACT_FACTOR", newsConfig.ImpactFactor)
	newsConfig.ImpactJitter = envFloat64("NEWS_IMPACT_JITTER", newsConfig.ImpactJitter)
	api.StartNewsEngine(newsCtx, store, engine, newsConfig)
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	newsCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	if err := engine.Shutdown(ctx); err != nil {
		log.Printf("engine shutdown error: %v", err)
	}
}

func createStore() *api.MarketStore {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	if dsn == "" {
		return api.NewMarketStore()
	}
	conn, err := db.NewConnection(dsn)
	if err != nil {
		log.Fatalf("db connection error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("db ping error: %v", err)
	}
	queries := db.NewQueries(conn)
	store, err := api.NewMarketStoreWithDB(ctx, queries)
	if err != nil {
		log.Fatalf("db store error: %v", err)
	}
	return store
}

func loadAPIKeys() *auth.APIKeyCache {
	cache := auth.NewAPIKeyCache()
	raw := strings.TrimSpace(os.Getenv("API_KEYS"))
	if raw == "" {
		return cache
	}
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if err := cache.AddHex(value); err != nil {
			log.Printf("invalid API key %q: %v", value, err)
		}
	}
	return cache
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid duration for %s: %v", key, err)
		return fallback
	}
	return parsed
}

func envInt64(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		log.Printf("invalid int64 for %s: %v", key, err)
		return fallback
	}
	return parsed
}

func envFloat64(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		log.Printf("invalid float64 for %s: %v", key, err)
		return fallback
	}
	return parsed
}
