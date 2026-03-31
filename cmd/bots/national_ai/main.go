package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/bots"
)

type config struct {
	BaseURL            string
	APIKey             string
	UserID             int64
	SwapAmount         int64
	DeviationThreshold float64
	Jitter             time.Duration
	RefreshInterval    time.Duration
	RequestTimeout     time.Duration
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	client := bots.NewAPIClient(cfg.BaseURL, cfg.APIKey, cfg.RequestTimeout)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ticker := time.NewTicker(cfg.RefreshInterval)
	defer ticker.Stop()
	for {
		runOnce(client, cfg, rng)
		<-ticker.C
	}
}

func runOnce(client *bots.APIClient, cfg config, rng *rand.Rand) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	rates, err := client.TheoreticalFXRates(ctx)
	cancel()
	if err != nil {
		log.Printf("national ai: theoretical rates fetch failed: %v", err)
		return
	}
	ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
	pools, err := client.Pools(ctx)
	cancel()
	if err != nil {
		log.Printf("national ai: pools fetch failed: %v", err)
		return
	}
	for _, rate := range rates {
		marketRate, ok := marketRateForCurrency(pools, rate.BaseCurrency, rate.QuoteCurrency)
		if !ok {
			continue
		}
		deviation := bots.FXDeviation(rate.Rate, marketRate)
		if deviation > -cfg.DeviationThreshold && deviation < cfg.DeviationThreshold {
			continue
		}
		if cfg.Jitter > 0 {
			jitterNs := cfg.Jitter.Nanoseconds()
			if jitterNs > 0 {
				time.Sleep(time.Duration(rng.Int63n(jitterNs + 1)))
			}
		}
		from := rate.QuoteCurrency
		to := rate.BaseCurrency
		if deviation > cfg.DeviationThreshold {
			from = rate.BaseCurrency
			to = rate.QuoteCurrency
		}
		req := bots.PoolSwapRequest{
			UserID:       cfg.UserID,
			FromCurrency: from,
			ToCurrency:   to,
			Amount:       cfg.SwapAmount,
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		result, err := client.SwapPool(ctx, 0, req)
		cancel()
		if err != nil {
			log.Printf("national ai: swap failed %s->%s: %v", from, to, err)
			continue
		}
		log.Printf("national ai: intervened %s->%s in=%d out=%d", result.FromCurrency, result.ToCurrency, result.AmountIn, result.AmountOut)
	}
}

func marketRateForCurrency(pools []bots.LiquidityPool, baseCurrency, quoteCurrency string) (float64, bool) {
	for _, pool := range pools {
		if !strings.EqualFold(pool.BaseCurrency, baseCurrency) || !strings.EqualFold(pool.QuoteCurrency, quoteCurrency) {
			continue
		}
		rate := bots.PoolRate(pool)
		if rate > 0 {
			return rate, true
		}
	}
	for _, pool := range pools {
		if !strings.EqualFold(pool.BaseCurrency, quoteCurrency) || !strings.EqualFold(pool.QuoteCurrency, baseCurrency) {
			continue
		}
		rate := bots.PoolRate(pool)
		if rate > 0 {
			return 1 / rate, true
		}
	}
	return 0, false
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
	swapAmount, err := bots.EnvInt64("SWAP_AMOUNT", 1000)
	if err != nil {
		return config{}, err
	}
	if swapAmount <= 0 {
		return config{}, errors.New("SWAP_AMOUNT must be positive")
	}
	deviationThreshold, err := bots.EnvFloat64("DEVIATION_THRESHOLD", 0.1)
	if err != nil {
		return config{}, err
	}
	if deviationThreshold <= 0 {
		return config{}, errors.New("DEVIATION_THRESHOLD must be positive")
	}
	jitter, err := bots.EnvDuration("JITTER", 0)
	if err != nil {
		return config{}, err
	}
	if jitter < 0 {
		return config{}, errors.New("JITTER must be non-negative")
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 10*time.Second)
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
		BaseURL:            baseURL,
		APIKey:             apiKey,
		UserID:             userID,
		SwapAmount:         swapAmount,
		DeviationThreshold: deviationThreshold,
		Jitter:             jitter,
		RefreshInterval:    refreshInterval,
		RequestTimeout:     requestTimeout,
	}, nil
}
