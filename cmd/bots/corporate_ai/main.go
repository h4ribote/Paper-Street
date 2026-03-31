package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/bots"
)

type config struct {
	BaseURL          string
	APIKey           string
	ProcurementIDs   []int64
	BuybackAssetID   int64
	UserID           int64
	Quantity         int64
	PremiumBps       int64
	BuybackThreshold int64
	CashCurrency     string
	RefreshInterval  time.Duration
	RequestTimeout   time.Duration
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	client := bots.NewAPIClient(cfg.BaseURL, cfg.APIKey, cfg.RequestTimeout)
	ticker := time.NewTicker(cfg.RefreshInterval)
	defer ticker.Stop()
	for {
		runOnce(client, cfg)
		<-ticker.C
	}
}

func runOnce(client *bots.APIClient, cfg config) {
	for _, assetID := range cfg.ProcurementIDs {
		price := fetchMidPrice(client, assetID, cfg.RequestTimeout)
		if price <= 0 {
			continue
		}
		premium := price * cfg.PremiumBps / 10000
		req := bots.OrderRequest{
			AssetID:  assetID,
			UserID:   cfg.UserID,
			Side:     "BUY",
			Type:     "LIMIT",
			Quantity: cfg.Quantity,
			Price:    price + premium,
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		order, err := client.SubmitOrder(ctx, req)
		cancel()
		if err != nil {
			log.Printf("corporate ai: procurement order failed asset=%d: %v", assetID, err)
			continue
		}
		log.Printf("corporate ai: procurement order placed id=%d asset=%d price=%d", order.ID, assetID, order.Price)
	}
	if cfg.BuybackAssetID == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	balances, err := client.Balances(ctx, cfg.UserID)
	cancel()
	if err != nil {
		log.Printf("corporate ai: balances fetch failed: %v", err)
		return
	}
	cash := balanceForCurrency(balances, cfg.CashCurrency)
	if cash < cfg.BuybackThreshold {
		return
	}
	req := bots.OrderRequest{
		AssetID:  cfg.BuybackAssetID,
		UserID:   cfg.UserID,
		Side:     "BUY",
		Type:     "MARKET",
		Quantity: cfg.Quantity,
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	order, err := client.SubmitOrder(ctx, req)
	cancel()
	if err != nil {
		log.Printf("corporate ai: buyback order failed asset=%d: %v", cfg.BuybackAssetID, err)
		return
	}
	log.Printf("corporate ai: buyback order placed id=%d asset=%d qty=%d", order.ID, order.AssetID, order.Quantity)
}

func balanceForCurrency(balances []bots.Balance, currency string) int64 {
	for _, balance := range balances {
		if strings.EqualFold(balance.Currency, currency) {
			return balance.Amount
		}
	}
	return 0
}

func fetchMidPrice(client *bots.APIClient, assetID int64, timeout time.Duration) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	snapshot, err := client.OrderBook(ctx, assetID, 1)
	cancel()
	if err != nil {
		return 0
	}
	return bots.MidPrice(snapshot, 0)
}

func loadConfig() (config, error) {
	baseURL := strings.TrimSpace(os.Getenv("API_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	if apiKey == "" {
		apiKey = bots.FirstAPIKey(os.Getenv("API_KEYS"))
	}
	if apiKey == "" {
		return config{}, errors.New("API_KEY or API_KEYS is required")
	}
	procurementIDs, err := bots.EnvInt64List("PROCUREMENT_ASSET_IDS", nil)
	if err != nil {
		return config{}, err
	}
	if len(procurementIDs) == 0 {
		return config{}, errors.New("PROCUREMENT_ASSET_IDS is required")
	}
	buybackAssetID, err := bots.EnvInt64("BUYBACK_ASSET_ID", 0)
	if err != nil {
		return config{}, err
	}
	userID, err := bots.EnvInt64("USER_ID", 1)
	if err != nil {
		return config{}, err
	}
	if userID <= 0 {
		return config{}, errors.New("USER_ID must be positive")
	}
	quantity, err := bots.EnvInt64("ORDER_QUANTITY", 10)
	if err != nil {
		return config{}, err
	}
	if quantity <= 0 {
		return config{}, errors.New("ORDER_QUANTITY must be positive")
	}
	premiumBps, err := bots.EnvInt64("PREMIUM_BPS", 800)
	if err != nil {
		return config{}, err
	}
	if premiumBps <= 0 {
		return config{}, errors.New("PREMIUM_BPS must be positive")
	}
	buybackThreshold, err := bots.EnvInt64("BUYBACK_CASH_THRESHOLD", 1000000)
	if err != nil {
		return config{}, err
	}
	if buybackThreshold < 0 {
		return config{}, errors.New("BUYBACK_CASH_THRESHOLD must be non-negative")
	}
	cashCurrency := strings.TrimSpace(os.Getenv("CASH_CURRENCY"))
	if cashCurrency == "" {
		cashCurrency = "ARC"
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 20*time.Second)
	if err != nil {
		return config{}, err
	}
	if refreshInterval <= 0 {
		return config{}, errors.New("REFRESH_INTERVAL must be positive")
	}
	requestTimeout, err := bots.EnvDuration("REQUEST_TIMEOUT", 3*time.Second)
	if err != nil {
		return config{}, err
	}
	if requestTimeout <= 0 {
		return config{}, errors.New("REQUEST_TIMEOUT must be positive")
	}
	return config{
		BaseURL:          baseURL,
		APIKey:           apiKey,
		ProcurementIDs:   procurementIDs,
		BuybackAssetID:   buybackAssetID,
		UserID:           userID,
		Quantity:         quantity,
		PremiumBps:       premiumBps,
		BuybackThreshold: buybackThreshold,
		CashCurrency:     cashCurrency,
		RefreshInterval:  refreshInterval,
		RequestTimeout:   requestTimeout,
	}, nil
}
