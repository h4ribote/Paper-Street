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
	BaseURL         string
	APIKey          string
	BondAssetID     int64
	EquityAssetID   int64
	UserID          int64
	Quantity        int64
	DividendPerUnit int64
	RiskPremium     float64
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
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
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	bonds, err := client.Bonds(ctx)
	cancel()
	if err != nil {
		log.Printf("yield hunter: bonds fetch failed: %v", err)
		return
	}
	var bond bots.PerpetualBondInfo
	found := false
	for _, candidate := range bonds {
		if candidate.Asset.ID == cfg.BondAssetID {
			bond = candidate
			found = true
			break
		}
	}
	if !found {
		log.Printf("yield hunter: bond asset %d not found", cfg.BondAssetID)
		return
	}
	bondPrice := fetchMidPrice(client, cfg.BondAssetID, cfg.RequestTimeout)
	if bondPrice <= 0 {
		bondPrice = bond.TheoreticalPrice
	}
	equityPrice := fetchMidPrice(client, cfg.EquityAssetID, cfg.RequestTimeout)
	if bondPrice <= 0 || equityPrice <= 0 {
		return
	}
	bondYield := bots.BondYield(bond.BaseCoupon, bondPrice)
	equityYield := bots.EquityYield(cfg.DividendPerUnit, equityPrice)
	preference := bots.YieldPreference(bondYield, equityYield, cfg.RiskPremium)
	if preference == "" {
		return
	}
	assetID := cfg.EquityAssetID
	price := equityPrice
	if preference == "BOND" {
		assetID = cfg.BondAssetID
		price = bondPrice
	}

	ctxB, cancelB := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	balances, _ := client.Balances(ctxB, cfg.UserID)
	cancelB()
	var cash int64
	for _, b := range balances {
		if b.Currency == "USD" {
			cash = b.Amount
			break
		}
	}

	qty := cfg.Quantity
	if cash < price*qty {
		qty = cash / price
	}
	if qty <= 0 {
		return
	}

	req := bots.OrderRequest{
		AssetID:  assetID,
		UserID:   cfg.UserID,
		Side:     "BUY",
		Type:     "LIMIT",
		Quantity: qty,
		Price:    price,
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	order, err := client.SubmitOrder(ctx, req)
	cancel()
	if err != nil {
		log.Printf("yield hunter: submit order failed: %v", err)
		return
	}
	log.Printf("yield hunter: placed %s order id=%d asset=%d price=%d", preference, order.ID, order.AssetID, order.Price)
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
	requestTimeout, err := bots.EnvDuration("REQUEST_TIMEOUT", 3*time.Second)
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
	bondAssetID, err := bots.EnvInt64("BOND_ASSET_ID", 301)
	if err != nil {
		return config{}, err
	}
	if bondAssetID <= 0 {
		return config{}, errors.New("BOND_ASSET_ID must be positive")
	}
	equityAssetID, err := bots.EnvInt64("EQUITY_ASSET_ID", 101)
	if err != nil {
		return config{}, err
	}
	if equityAssetID <= 0 {
		return config{}, errors.New("EQUITY_ASSET_ID must be positive")
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
	dividendPerUnit, err := bots.EnvInt64("DIVIDEND_PER_UNIT", 100)
	if err != nil {
		return config{}, err
	}
	if dividendPerUnit <= 0 {
		return config{}, errors.New("DIVIDEND_PER_UNIT must be positive")
	}
	riskPremium, err := bots.EnvFloat64("RISK_PREMIUM", 0.005)
	if err != nil {
		return config{}, err
	}
	if riskPremium < 0 {
		return config{}, errors.New("RISK_PREMIUM must be non-negative")
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 15*time.Second)
	if err != nil {
		return config{}, err
	}
	if refreshInterval <= 0 {
		return config{}, errors.New("REFRESH_INTERVAL must be positive")
	}
	return config{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		BondAssetID:     bondAssetID,
		EquityAssetID:   equityAssetID,
		UserID:          userID,
		Quantity:        quantity,
		DividendPerUnit: dividendPerUnit,
		RiskPremium:     riskPremium,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}
