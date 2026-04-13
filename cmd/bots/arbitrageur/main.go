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
)

type config struct {
	BaseURL              string
	APIKey               string
	IndexAssetID         int64
	ComponentAssetIDs    []int64
	UserID               int64
	OrderQuantity        int64
	ThresholdBps         int64
	EnableFXArb          bool
	FXDeviationThreshold float64
	FXSwapAmount         int64
	RefreshInterval      time.Duration
	RequestTimeout       time.Duration
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
	nav, ok := indexNAV(client, cfg)
	if ok {
		runIndexArb(client, cfg, nav)
	}
	if cfg.EnableFXArb {
		runFXArb(client, cfg)
	}
}

func indexNAV(client *bots.APIClient, cfg config) (int64, bool) {
	var total int64
	for _, assetID := range cfg.ComponentAssetIDs {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		snapshot, err := client.OrderBook(ctx, assetID, 1)
		cancel()
		if err != nil {
			log.Printf("arbitrageur: orderbook fetch failed asset=%d: %v", assetID, err)
			return 0, false
		}
		price := bots.MidPrice(snapshot, 0)
		if price <= 0 {
			return 0, false
		}
		total += price
	}
	if total <= 0 {
		return 0, false
	}
	return total, true
}

func runIndexArb(client *bots.APIClient, cfg config, nav int64) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	indexSnapshot, err := client.OrderBook(ctx, cfg.IndexAssetID, 1)
	cancel()
	if err != nil {
		log.Printf("arbitrageur: index orderbook fetch failed: %v", err)
		return
	}
	indexPrice := bots.MidPrice(indexSnapshot, 0)
	if indexPrice <= 0 {
		return
	}
	spread := (indexPrice - nav) * 10000 / nav
	switch {
	case spread > cfg.ThresholdBps:
		request := bots.IndexActionRequest{UserID: cfg.UserID, Quantity: cfg.OrderQuantity}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		_, err := client.IndexAction(ctx, cfg.IndexAssetID, "create", request)
		cancel()
		if err != nil {
			log.Printf("arbitrageur: index create failed: %v", err)
			return
		}
		sellReq := bots.OrderRequest{
			AssetID:  cfg.IndexAssetID,
			UserID:   cfg.UserID,
			Side:     "SELL",
			Type:     "LIMIT",
			Quantity: cfg.OrderQuantity,
			Price:    nav * 105 / 100, // 5% worse than NAV max
		}
		ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
		order, err := client.SubmitOrder(ctx, sellReq)
		cancel()
		if err != nil {
			log.Printf("arbitrageur: index sell failed: %v", err)
			return
		}
		log.Printf("arbitrageur: premium arb sell id=%d qty=%d spread_bps=%d", order.ID, order.Quantity, spread)
	case spread < -cfg.ThresholdBps:
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
		cost := indexPrice * cfg.OrderQuantity
		if cash < cost {
			log.Printf("arbitrageur: insufficient ARC balance for discount arb (need %d, have %d)", cost, cash)
			return
		}

		buyReq := bots.OrderRequest{
			AssetID:  cfg.IndexAssetID,
			UserID:   cfg.UserID,
			Side:     "BUY",
			Type:     "LIMIT",
			Quantity: cfg.OrderQuantity,
			Price:    nav * 105 / 100, // 5% worse than NAV max
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		order, err := client.SubmitOrder(ctx, buyReq)
		cancel()
		if err != nil {
			log.Printf("arbitrageur: index buy failed: %v", err)
			return
		}
		request := bots.IndexActionRequest{UserID: cfg.UserID, Quantity: cfg.OrderQuantity}
		ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
		_, err = client.IndexAction(ctx, cfg.IndexAssetID, "redeem", request)
		cancel()
		if err != nil {
			log.Printf("arbitrageur: index redeem failed: %v", err)
			return
		}
		log.Printf("arbitrageur: discount arb buy id=%d qty=%d spread_bps=%d", order.ID, order.Quantity, spread)
	}
}

func runFXArb(client *bots.APIClient, cfg config) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	rates, err := client.TheoreticalFXRates(ctx)
	cancel()
	if err != nil {
		log.Printf("arbitrageur: theoretical rates fetch failed: %v", err)
		return
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	pools, err := client.Pools(ctx)
	cancel()
	if err != nil {
		log.Printf("arbitrageur: pools fetch failed: %v", err)
		return
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	balances, err := client.Balances(ctx, cfg.UserID)
	cancel()
	if err != nil {
		log.Printf("arbitrageur: balances fetch failed: %v", err)
		return
	}

	balanceMap := make(map[string]int64)
	for _, b := range balances {
		balanceMap[b.Currency] = b.Amount
	}

	for _, rate := range rates {
		marketRate, ok := fxMarketRate(pools, rate.BaseCurrency, rate.QuoteCurrency)
		if !ok {
			continue
		}
		deviation := bots.FXDeviation(rate.Rate, marketRate)
		if deviation > -cfg.FXDeviationThreshold && deviation < cfg.FXDeviationThreshold {
			continue
		}
		from := rate.QuoteCurrency
		to := rate.BaseCurrency
		if deviation > cfg.FXDeviationThreshold {
			from = rate.BaseCurrency
			to = rate.QuoteCurrency
		}
		swapAmount := cfg.FXSwapAmount
		if balanceMap[from] < swapAmount {
			swapAmount = balanceMap[from]
		}
		if swapAmount < 100 {
			continue
		}
		req := bots.PoolSwapRequest{
			UserID:       cfg.UserID,
			FromCurrency: from,
			ToCurrency:   to,
			Amount:       swapAmount,
		}
		ctxSwap, cancelSwap := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		result, err := client.SwapPool(ctxSwap, 0, req)
		cancelSwap()
		if err != nil {
			log.Printf("arbitrageur: fx swap failed %s->%s: %v", from, to, err)
			continue
		}
		balanceMap[from] -= result.AmountIn
		balanceMap[to] += result.AmountOut
		log.Printf("arbitrageur: fx arb %s->%s in=%d out=%d", result.FromCurrency, result.ToCurrency, result.AmountIn, result.AmountOut)
	}
}

func fxMarketRate(pools []bots.LiquidityPool, baseCurrency, quoteCurrency string) (float64, bool) {
	for _, pool := range pools {
		if strings.EqualFold(pool.BaseCurrency, baseCurrency) && strings.EqualFold(pool.QuoteCurrency, quoteCurrency) {
			rate := bots.PoolRate(pool)
			if rate > 0 {
				return rate, true
			}
		}
	}
	for _, pool := range pools {
		if strings.EqualFold(pool.BaseCurrency, quoteCurrency) && strings.EqualFold(pool.QuoteCurrency, baseCurrency) {
			rate := bots.PoolRate(pool)
			if rate > 0 {
				return 1 / rate, true
			}
		}
	}
	return 0, false
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
	indexAssetID, err := bots.EnvInt64("INDEX_ASSET_ID", 0)
	if err != nil {
		return config{}, err
	}
	if indexAssetID <= 0 {
		return config{}, errors.New("INDEX_ASSET_ID must be positive")
	}
	componentIDs, err := bots.EnvInt64List("COMPONENT_ASSET_IDS", nil)
	if err != nil {
		return config{}, err
	}
	if len(componentIDs) == 0 {
		return config{}, errors.New("COMPONENT_ASSET_IDS is required")
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
	orderQuantity, err := bots.EnvInt64("ORDER_QUANTITY", 1)
	if err != nil {
		return config{}, err
	}
	if orderQuantity <= 0 {
		return config{}, errors.New("ORDER_QUANTITY must be positive")
	}
	thresholdBps, err := bots.EnvInt64("THRESHOLD_BPS", 20)
	if err != nil {
		return config{}, err
	}
	if thresholdBps <= 0 {
		return config{}, errors.New("THRESHOLD_BPS must be positive")
	}
	enableFXArb := false
	if raw := strings.TrimSpace(os.Getenv("ENABLE_FX_ARB")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return config{}, err
		}
		enableFXArb = value
	}
	fxDeviation, err := bots.EnvFloat64("FX_DEVIATION_THRESHOLD", 0.02)
	if err != nil {
		return config{}, err
	}
	if fxDeviation <= 0 {
		return config{}, errors.New("FX_DEVIATION_THRESHOLD must be positive")
	}
	fxSwapAmount, err := bots.EnvInt64("FX_SWAP_AMOUNT", 1000)
	if err != nil {
		return config{}, err
	}
	if fxSwapAmount <= 0 {
		return config{}, errors.New("FX_SWAP_AMOUNT must be positive")
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 8*time.Second)
	if err != nil {
		return config{}, err
	}
	if refreshInterval <= 0 {
		return config{}, errors.New("REFRESH_INTERVAL must be positive")
	}
	return config{
		BaseURL:              baseURL,
		APIKey:               apiKey,
		IndexAssetID:         indexAssetID,
		ComponentAssetIDs:    componentIDs,
		UserID:               userID,
		OrderQuantity:        orderQuantity,
		ThresholdBps:         thresholdBps,
		EnableFXArb:          enableFXArb,
		FXDeviationThreshold: fxDeviation,
		FXSwapAmount:         fxSwapAmount,
		RefreshInterval:      refreshInterval,
		RequestTimeout:       requestTimeout,
	}, nil
}
