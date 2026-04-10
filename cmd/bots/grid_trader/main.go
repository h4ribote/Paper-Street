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
	AssetIDs        []int64
	UserID          int64
	Quantity        int64
	StepBps         int64
	Levels          int
	Depth           int
	FallbackPrice   int64
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
}

type gridState struct {
	orderIDs []int64
	lastMid  int64
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	client := bots.NewAPIClient(cfg.BaseURL, cfg.APIKey, cfg.RequestTimeout)
	state := make(map[int64]*gridState)
	ticker := time.NewTicker(cfg.RefreshInterval)
	defer ticker.Stop()
	for {
		runOnce(client, cfg, state)
		<-ticker.C
	}
}

func runOnce(client *bots.APIClient, cfg config, state map[int64]*gridState) {
	for _, assetID := range cfg.AssetIDs {
		entry := state[assetID]
		if entry == nil {
			entry = &gridState{}
			state[assetID] = entry
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		snapshot, err := client.OrderBook(ctx, assetID, cfg.Depth)
		cancel()
		if err != nil {
			log.Printf("grid trader: orderbook fetch failed asset=%d: %v", assetID, err)
			continue
		}
		mid := bots.MidPrice(snapshot, cfg.FallbackPrice)
		step := int64(math.Round(float64(mid) * float64(cfg.StepBps) / 10000.0))
		if step < 1 {
			step = 1
		}
		if entry.lastMid != 0 && math.Abs(float64(entry.lastMid-mid)) < float64(step)/2 {
			continue
		}

		ctxBal, cancelBal := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		balances, _ := client.Balances(ctxBal, cfg.UserID)
		cancelBal()
		var cash int64
		for _, b := range balances {
			if b.Currency == "ARC" {
				cash = b.Amount
				break
			}
		}

		ctxAst, cancelAst := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		assets, _ := client.PortfolioAssets(ctxAst, cfg.UserID)
		cancelAst()
		var inventory int64
		for _, a := range assets {
			if a.AssetID == assetID {
				inventory = a.Quantity
				break
			}
		}

		cancelOrders(client, cfg.RequestTimeout, assetID, entry.orderIDs)
		entry.orderIDs = nil
		entry.lastMid = mid
		levels := bots.GridLevels(mid, cfg.StepBps, cfg.Levels)
		
		usedCash := int64(0)
		usedInventory := int64(0)
		placedCount := 0

		for _, level := range levels {
			qty := cfg.Quantity
			if level.Side == "BUY" {
				cost := level.Price * qty
				if cash < usedCash+cost {
					qty = (cash - usedCash) / level.Price
				}
				if qty <= 0 {
					continue
				}
				usedCash += level.Price * qty
			} else {
				if inventory < usedInventory+qty {
					qty = inventory - usedInventory
				}
				if qty <= 0 {
					continue
				}
				usedInventory += qty
			}

			req := bots.OrderRequest{
				AssetID:  assetID,
				UserID:   cfg.UserID,
				Side:     level.Side,
				Type:     "LIMIT",
				Quantity: qty,
				Price:    level.Price,
			}
			ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
			order, err := client.SubmitOrder(ctx, req)
			cancel()
			if err != nil {
				log.Printf("grid trader: submit order failed asset=%d: %v", assetID, err)
				continue
			}
			entry.orderIDs = append(entry.orderIDs, order.ID)
			placedCount++
		}
		log.Printf("grid trader: placed %d grid orders asset=%d mid=%d", placedCount, assetID, mid)
	}
}

func cancelOrders(client *bots.APIClient, timeout time.Duration, assetID int64, orderIDs []int64) {
	for _, orderID := range orderIDs {
		if orderID == 0 {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err := client.CancelOrder(ctx, assetID, orderID)
		cancel()
		if err != nil {
			log.Printf("grid trader: cancel order %d failed: %v", orderID, err)
		}
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
	assetIDs, err := bots.EnvInt64List("ASSET_IDS", nil)
	if err != nil {
		return config{}, err
	}
	if len(assetIDs) == 0 {
		assetID, err := bots.EnvInt64("ASSET_ID", 1)
		if err != nil {
			return config{}, err
		}
		if assetID <= 0 {
			return config{}, errors.New("ASSET_ID must be positive")
		}
		assetIDs = []int64{assetID}
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
	stepBps, err := bots.EnvInt64("STEP_BPS", 30)
	if err != nil {
		return config{}, err
	}
	if stepBps <= 0 {
		return config{}, errors.New("STEP_BPS must be positive")
	}
	levels, err := bots.EnvInt("GRID_LEVELS", 3)
	if err != nil {
		return config{}, err
	}
	if levels <= 0 {
		return config{}, errors.New("GRID_LEVELS must be positive")
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
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 4*time.Second)
	if err != nil {
		return config{}, err
	}
	if refreshInterval <= 0 {
		return config{}, errors.New("REFRESH_INTERVAL must be positive")
	}
	return config{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		AssetIDs:        assetIDs,
		UserID:          userID,
		Quantity:        quantity,
		StepBps:         stepBps,
		Levels:          levels,
		Depth:           depth,
		FallbackPrice:   fallbackPrice,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}
