package main

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/bots"
)

type config struct {
	BaseURL         string
	APIKey          string
	AssetID         int64
	UserID          int64
	TotalQuantity   int64
	Slices          int
	Side            string
	Timeframe       string
	CandleLimit     int
	Gamma           float64
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
}

type executionState struct {
	remaining int64
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	client := bots.NewAPIClient(cfg.BaseURL, cfg.APIKey, cfg.RequestTimeout)
	state := &executionState{remaining: cfg.TotalQuantity}
	ticker := time.NewTicker(cfg.RefreshInterval)
	defer ticker.Stop()
	for {
		runOnce(client, cfg, state)
		<-ticker.C
	}
}

func runOnce(client *bots.APIClient, cfg config, state *executionState) {
	// Asset check
	ctxA, cancelA := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	_, errA := client.Asset(ctxA, cfg.AssetID)
	cancelA()
	if errA != nil {
		return // skip non-existent
	}

	if state.remaining <= 0 {
		state.remaining = cfg.TotalQuantity
	}
	sliceQty := int64(math.Round(float64(cfg.TotalQuantity) / float64(cfg.Slices)))
	if sliceQty <= 0 {
		sliceQty = cfg.TotalQuantity
	}
	if sliceQty > state.remaining {
		sliceQty = state.remaining
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	candles, err := client.Candles(ctx, cfg.AssetID, cfg.Timeframe, cfg.CandleLimit)
	cancel()
	if err != nil {
		log.Printf("whale: candles fetch failed asset=%d: %v", cfg.AssetID, err)
		return
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	snapshot, err := client.OrderBook(ctx, cfg.AssetID, 1)
	cancel()
	if err != nil {
		log.Printf("whale: orderbook fetch failed asset=%d: %v", cfg.AssetID, err)
		return
	}
	closes := bots.ClosingPrices(candles)
	sigma := bots.RelativeVolatility(closes)
	var volume int64
	for _, candle := range candles {
		volume += candle.Volume
	}
	mid := bots.MidPrice(snapshot, 0)
	if mid <= 1 {
		mid = bots.VWAP(candles)
	}
	if mid <= 1 {
		log.Printf("whale: skipping order asset=%d due to lack of pricing data", cfg.AssetID)
		return
	}
	impactPrice := bots.ImpactAdjustedPrice(mid, sigma, sliceQty, volume, cfg.Gamma, cfg.Side)
	price := impactPrice
	vwap := bots.VWAP(candles)
	if vwap > 0 {
		price = int64(math.Round((float64(price) + float64(vwap)) / 2))
	}

	ctxB, cancelB := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	balances, _ := client.Balances(ctxB, cfg.UserID)
	cancelB()
	var cash int64
	for _, b := range balances {
		if b.Currency == "ARC" {
			cash = b.Amount
			break
		}
	}

	ctxI, cancelI := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	assets, _ := client.PortfolioAssets(ctxI, cfg.UserID)
	cancelI()
	var inventory int64
	for _, a := range assets {
		if a.Asset.ID == cfg.AssetID {
			inventory = a.Quantity
			break
		}
	}

	if cfg.Side == "BUY" {
		if cash < price*sliceQty {
			sliceQty = cash / price
		}
	} else {
		if inventory < sliceQty {
			sliceQty = inventory
		}
	}

	if sliceQty <= 0 {
		return
	}

	req := bots.OrderRequest{
		AssetID:  cfg.AssetID,
		UserID:   cfg.UserID,
		Side:     cfg.Side,
		Type:     "LIMIT",
		Quantity: sliceQty,
		Price:    price,
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	order, err := client.SubmitOrder(ctx, req)
	cancel()
	if err != nil {
		log.Printf("whale: submit order failed asset=%d: %v", cfg.AssetID, err)
		return
	}
	state.remaining -= sliceQty
	log.Printf("whale: slice order placed id=%d asset=%d side=%s qty=%d price=%d", order.ID, cfg.AssetID, order.Side, order.Quantity, order.Price)
}

func loadConfig() (config, error) {
	baseURL := strings.TrimSpace(os.Getenv("API_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	requestTimeout, err := bots.EnvDuration("REQUEST_TIMEOUT", 2*time.Second)
	if err != nil {
		return config{}, err
	}
	if requestTimeout <= 0 {
		return config{}, errors.New("REQUEST_TIMEOUT must be positive")
	}
	authResult, err := bots.ResolveAuth(
		baseURL,
		strings.TrimSpace(os.Getenv("API_KEY")),
		strings.TrimSpace(os.Getenv("BOT_ROLE")),
		strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
		strings.TrimSpace(os.Getenv("API_KEY_FILE")),
		requestTimeout,
	)
	if err != nil {
		return config{}, err
	}
	apiKey := authResult.APIKey
	assetID, err := bots.EnvInt64("ASSET_ID", 1)
	if err != nil {
		return config{}, err
	}
	if assetID <= 0 {
		return config{}, errors.New("ASSET_ID must be positive")
	}
	userID, err := bots.EnvInt64("USER_ID", authResult.UserID)
	if err != nil {
		return config{}, err
	}
	if userID <= 0 {
		return config{}, errors.New("USER_ID or BOT_ROLE is required")
	}
	if authResult.UserID != 0 && userID != authResult.UserID {
		return config{}, errors.New("USER_ID does not match role assignment")
	}
	totalQuantity, err := bots.EnvInt64("TOTAL_QUANTITY", 1000)
	if err != nil {
		return config{}, err
	}
	if totalQuantity <= 0 {
		return config{}, errors.New("TOTAL_QUANTITY must be positive")
	}
	slices, err := bots.EnvInt("SLICES", 5)
	if err != nil {
		return config{}, err
	}
	if slices <= 0 {
		return config{}, errors.New("SLICES must be positive")
	}
	side := strings.ToUpper(strings.TrimSpace(os.Getenv("SIDE")))
	if side == "" {
		side = "BUY"
	}
	if side != "BUY" && side != "SELL" {
		return config{}, errors.New("SIDE must be BUY or SELL")
	}
	timeframe := strings.TrimSpace(os.Getenv("TIMEFRAME"))
	if timeframe == "" {
		timeframe = "5m"
	}
	candleLimit, err := bots.EnvInt("CANDLE_LIMIT", 48)
	if err != nil {
		return config{}, err
	}
	if candleLimit <= 0 {
		return config{}, errors.New("CANDLE_LIMIT must be positive")
	}
	gamma, err := bots.EnvFloat64("IMPACT_GAMMA", 0.5)
	if err != nil {
		return config{}, err
	}
	if gamma <= 0 {
		return config{}, errors.New("IMPACT_GAMMA must be positive")
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 6*time.Second)
	if err != nil {
		return config{}, err
	}
	if refreshInterval <= 0 {
		return config{}, errors.New("REFRESH_INTERVAL must be positive")
	}
	return config{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		AssetID:         assetID,
		UserID:          userID,
		TotalQuantity:   totalQuantity,
		Slices:          slices,
		Side:            side,
		Timeframe:       timeframe,
		CandleLimit:     candleLimit,
		Gamma:           gamma,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}
