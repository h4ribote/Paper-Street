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
	buyIDs  []int64
	sellIDs []int64
	lastMid int64
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
	var snapshot engine.OrderBookSnapshot
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	orderbook, err := client.OrderBook(ctx, cfg.AssetID, cfg.Depth)
	cancel()
	if err != nil {
		log.Printf("orderbook fetch failed: %v (using fallback price)", err)
	} else {
		snapshot = orderbook
	}

	midPrice := bots.MidPrice(snapshot, cfg.FallbackPrice)
	if midPrice <= 0 {
		return nil
	}

	priceDiff := midPrice - state.lastMid
	if priceDiff < 0 {
		priceDiff = -priceDiff
	}

	needsUpdate := state.lastMid == 0 || len(state.buyIDs) == 0 || len(state.sellIDs) == 0 || float64(priceDiff)/float64(state.lastMid) > 0.005
	if !needsUpdate {
		return nil
	}

	ctxBalances, cancelBal := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	balances, _ := client.Balances(ctxBalances, cfg.UserID)
	cancelBal()
	var cash int64
	for _, b := range balances {
		if b.Currency == "ARC" {
			cash = b.Amount
			break
		}
	}

	ctxAssets, cancelAst := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	assets, _ := client.PortfolioAssets(ctxAssets, cfg.UserID)
	cancelAst()
	var inventory int64
	for _, a := range assets {
		if a.Asset.ID == cfg.AssetID {
			inventory = a.Quantity
			break
		}
	}

	// キャンセル
	for _, id := range state.buyIDs {
		cancelOrder(client, cfg.RequestTimeout, cfg.AssetID, id)
	}
	for _, id := range state.sellIDs {
		cancelOrder(client, cfg.RequestTimeout, cfg.AssetID, id)
	}
	state.buyIDs = nil
	state.sellIDs = nil

	submitOrder := func(req bots.OrderRequest) (*engine.Order, error) {
		if req.Quantity <= 0 {
			return nil, errors.New("insufficient balance/inventory")
		}
		ctxSubmit, cancelSubmit := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer cancelSubmit()
		return client.SubmitOrder(ctxSubmit, req)
	}

	levels := 5
	// 保有株式を全量売り注文にする（板の厚さに応じて分散）
	if inventory > 0 {
		sellQty := inventory / int64(levels)
		rem := inventory % int64(levels)
		for i := 1; i <= levels; i++ {
			qty := sellQty
			if i == levels {
				qty += rem
			}
			if qty > 0 {
				spreadMultiplier := float64(i) * float64(cfg.SpreadBps) / 10000.0
				askPrice := int64(math.Round(float64(midPrice) * (1.0 + spreadMultiplier/2.0)))
				if askPrice <= midPrice {
					askPrice = midPrice + int64(i)
				}
				sellReq := bots.OrderRequest{
					AssetID:  cfg.AssetID,
					UserID:   cfg.UserID,
					Side:     "SELL",
					Type:     "LIMIT",
					Quantity: qty,
					Price:    askPrice,
				}
				if order, err := submitOrder(sellReq); err == nil {
					state.sellIDs = append(state.sellIDs, order.ID)
					log.Printf("sell order placed id=%d price=%d qty=%d", order.ID, order.Price, order.Quantity)
				} else {
					log.Printf("submit sell order failed: %v", err)
				}
			}
		}
	}

	if cash > midPrice*int64(levels) {
		cashPerLevel := cash / int64(levels)
		for i := 1; i <= levels; i++ {
			spreadMultiplier := float64(i) * float64(cfg.SpreadBps) / 10000.0
			bidPrice := int64(math.Round(float64(midPrice) * (1.0 - spreadMultiplier/2.0)))
			if bidPrice >= midPrice {
				bidPrice = midPrice - int64(i)
			}
			if bidPrice < 1 {
				bidPrice = 1
			}
			qty := cashPerLevel / bidPrice
			if qty > 0 {
				buyReq := bots.OrderRequest{
					AssetID:  cfg.AssetID,
					UserID:   cfg.UserID,
					Side:     "BUY",
					Type:     "LIMIT",
					Quantity: qty,
					Price:    bidPrice,
				}
				if order, err := submitOrder(buyReq); err == nil {
					state.buyIDs = append(state.buyIDs, order.ID)
					log.Printf("buy order placed id=%d price=%d qty=%d", order.ID, order.Price, order.Quantity)
				} else {
					log.Printf("submit buy order failed: %v", err)
				}
			}
		}
	}

	state.lastMid = midPrice
	return nil
}

func cancelOrder(client *bots.APIClient, timeout time.Duration, assetID int64, orderID int64) {
	if orderID == 0 || client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.CancelOrder(ctx, assetID, orderID); err != nil {
		log.Printf("cancel order %d failed: %v", orderID, err)
	}
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
	quantity, err := bots.EnvInt64("ORDER_QUANTITY", 10)
	if err != nil {
		return config{}, err
	}
	if quantity <= 0 {
		return config{}, errors.New("ORDER_QUANTITY must be positive")
	}
	spreadBps, err := bots.EnvInt64("SPREAD_BPS", 50)
	if err != nil {
		return config{}, err
	}
	if spreadBps <= 0 {
		return config{}, errors.New("SPREAD_BPS must be positive")
	}
	depth, err := bots.EnvInt("ORDERBOOK_DEPTH", 1)
	if err != nil {
		return config{}, err
	}
	if depth <= 0 {
		return config{}, errors.New("ORDERBOOK_DEPTH must be positive")
	}
	fallbackPrice, err := bots.EnvInt64("FALLBACK_PRICE", 10000)
	if err != nil {
		return config{}, err
	}
	if fallbackPrice <= 0 {
		return config{}, errors.New("FALLBACK_PRICE must be positive")
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 2*time.Second)
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
		Quantity:        quantity,
		SpreadBps:       spreadBps,
		Depth:           depth,
		FallbackPrice:   fallbackPrice,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}
