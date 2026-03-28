package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/bots"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

type config struct {
	BaseURL         string
	APIKey          string
	AssetID         int64
	UserID          int64
	Quantity        int64
	SpreadBps       int64
	Depth           int
	FallbackPrice   int64
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
}

type orderState struct {
	buyID  int64
	sellID int64
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	client := bots.NewAPIClient(cfg.BaseURL, cfg.APIKey, cfg.RequestTimeout)
	state := &orderState{}
	if err := runOnce(client, cfg, state); err != nil {
		log.Printf("initial cycle error: %v", err)
	}
	ticker := time.NewTicker(cfg.RefreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := runOnce(client, cfg, state); err != nil {
			log.Printf("cycle error: %v", err)
		}
	}
}

func runOnce(client *bots.APIClient, cfg config, state *orderState) error {
	if client == nil || state == nil {
		return errors.New("missing client or state")
	}
	cancelOrder := func(orderID int64) {
		if orderID == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer cancel()
		if err := client.CancelOrder(ctx, orderID); err != nil {
			log.Printf("cancel order %d failed: %v", orderID, err)
		}
	}
	cancelOrder(state.buyID)
	cancelOrder(state.sellID)

	var snapshot engine.OrderBookSnapshot
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	orderbook, err := client.OrderBook(ctx, cfg.AssetID, cfg.Depth)
	cancel()
	if err != nil {
		log.Printf("orderbook fetch failed: %v (using fallback price)", err)
	} else {
		snapshot = orderbook
	}

	quote := bots.QuoteFromSnapshot(snapshot, cfg.SpreadBps, cfg.FallbackPrice)
	buyReq := bots.OrderRequest{
		AssetID:  cfg.AssetID,
		UserID:   cfg.UserID,
		Side:     "BUY",
		Type:     "LIMIT",
		Quantity: cfg.Quantity,
		Price:    quote.BidPrice,
	}
	sellReq := bots.OrderRequest{
		AssetID:  cfg.AssetID,
		UserID:   cfg.UserID,
		Side:     "SELL",
		Type:     "LIMIT",
		Quantity: cfg.Quantity,
		Price:    quote.AskPrice,
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	if order, err := client.SubmitOrder(ctx, buyReq); err != nil {
		log.Printf("submit buy order failed: %v", err)
		state.buyID = 0
	} else {
		state.buyID = order.ID
		log.Printf("buy order placed id=%d price=%d qty=%d", order.ID, order.Price, order.Quantity)
	}
	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	if order, err := client.SubmitOrder(ctx, sellReq); err != nil {
		log.Printf("submit sell order failed: %v", err)
		state.sellID = 0
	} else {
		state.sellID = order.ID
		log.Printf("sell order placed id=%d price=%d qty=%d", order.ID, order.Price, order.Quantity)
	}
	cancel()
	return nil
}

func loadConfig() (config, error) {
	baseURL := strings.TrimSpace(os.Getenv("API_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	if apiKey == "" {
		return config{}, errors.New("API_KEY is required")
	}
	assetID, err := envInt64("ASSET_ID", 1)
	if err != nil {
		return config{}, err
	}
	if assetID <= 0 {
		return config{}, errors.New("ASSET_ID must be positive")
	}
	userID, err := envInt64("USER_ID", 1)
	if err != nil {
		return config{}, err
	}
	if userID <= 0 {
		return config{}, errors.New("USER_ID must be positive")
	}
	quantity, err := envInt64("ORDER_QUANTITY", 10)
	if err != nil {
		return config{}, err
	}
	if quantity <= 0 {
		return config{}, errors.New("ORDER_QUANTITY must be positive")
	}
	spreadBps, err := envInt64("SPREAD_BPS", 50)
	if err != nil {
		return config{}, err
	}
	if spreadBps <= 0 {
		return config{}, errors.New("SPREAD_BPS must be positive")
	}
	depth, err := envInt("ORDERBOOK_DEPTH", 1)
	if err != nil {
		return config{}, err
	}
	if depth <= 0 {
		return config{}, errors.New("ORDERBOOK_DEPTH must be positive")
	}
	fallbackPrice, err := envInt64("FALLBACK_PRICE", 10000)
	if err != nil {
		return config{}, err
	}
	if fallbackPrice <= 0 {
		return config{}, errors.New("FALLBACK_PRICE must be positive")
	}
	refreshInterval, err := envDuration("REFRESH_INTERVAL", 2*time.Second)
	if err != nil {
		return config{}, err
	}
	if refreshInterval <= 0 {
		return config{}, errors.New("REFRESH_INTERVAL must be positive")
	}
	requestTimeout, err := envDuration("REQUEST_TIMEOUT", 2*time.Second)
	if err != nil {
		return config{}, err
	}
	if requestTimeout <= 0 {
		return config{}, errors.New("REQUEST_TIMEOUT must be positive")
	}
	return config{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		AssetID:         assetID,
		UserID:          userID,
		Quantity:        quantity,
		SpreadBps:       spreadBps,
		Depth:           depth,
		FallbackPrice:   fallbackPrice,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}

func envInt64(key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func envInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
