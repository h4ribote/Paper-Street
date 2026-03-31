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
	UserID          int64
	BaseQuantity    int64
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
	assets, err := client.Assets(ctx, "", "")
	cancel()
	if err != nil {
		log.Printf("public consumer: assets fetch failed: %v", err)
		return
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	indicators, err := client.MacroIndicators(ctx)
	cancel()
	if err != nil {
		log.Printf("public consumer: macro indicators fetch failed: %v", err)
		return
	}
	cci := averageCCI(indicators)
	luxuryShare := bots.ConsumerLuxuryShare(cci)
	essential, luxury := splitAssets(assets)
	if len(essential) == 0 {
		essential = assets
	}
	essentialQty := bots.ClampQuantity(cfg.BaseQuantity, 1-luxuryShare)
	luxuryQty := bots.ClampQuantity(cfg.BaseQuantity, luxuryShare)
	placeOrders(client, cfg, essential, essentialQty)
	if len(luxury) > 0 && luxuryQty > 0 {
		placeOrders(client, cfg, luxury, luxuryQty)
	}
	log.Printf("public consumer: placed orders essential=%d luxury=%d cci=%d", essentialQty, luxuryQty, cci)
}

func placeOrders(client *bots.APIClient, cfg config, assets []bots.Asset, quantity int64) {
	if quantity <= 0 {
		return
	}
	perAsset := quantity
	if len(assets) > 0 {
		perAsset = quantity / int64(len(assets))
		if perAsset == 0 {
			perAsset = 1
		}
	}
	for _, asset := range assets {
		req := bots.OrderRequest{
			AssetID:  asset.ID,
			UserID:   cfg.UserID,
			Side:     "BUY",
			Type:     "MARKET",
			Quantity: perAsset,
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		order, err := client.SubmitOrder(ctx, req)
		cancel()
		if err != nil {
			log.Printf("public consumer: submit order failed asset=%d: %v", asset.ID, err)
			continue
		}
		log.Printf("public consumer: bought asset=%d qty=%d id=%d", asset.ID, order.Quantity, order.ID)
	}
}

func splitAssets(assets []bots.Asset) (essential []bots.Asset, luxury []bots.Asset) {
	for _, asset := range assets {
		assetType := strings.ToUpper(asset.Type)
		sector := strings.ToUpper(asset.Sector)
		if assetType == "COMMODITY" || sector == "FOOD" || sector == "ENERGY" {
			essential = append(essential, asset)
			continue
		}
		luxury = append(luxury, asset)
	}
	return essential, luxury
}

func averageCCI(indicators []bots.MacroIndicator) int64 {
	var sum int64
	var count int64
	for _, indicator := range indicators {
		if strings.EqualFold(indicator.Type, "CONSUMER_CONFIDENCE") {
			sum += indicator.Value
			count++
		}
	}
	if count == 0 {
		return 100
	}
	return sum / count
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
	userID, err := bots.EnvInt64("USER_ID", 1)
	if err != nil {
		return config{}, err
	}
	if userID <= 0 {
		return config{}, errors.New("USER_ID must be positive")
	}
	baseQuantity, err := bots.EnvInt64("ORDER_QUANTITY", 10)
	if err != nil {
		return config{}, err
	}
	if baseQuantity <= 0 {
		return config{}, errors.New("ORDER_QUANTITY must be positive")
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
		BaseURL:         baseURL,
		APIKey:          apiKey,
		UserID:          userID,
		BaseQuantity:    baseQuantity,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}
