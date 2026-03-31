package main

import (
	"context"
	"errors"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/bots"
)

type config struct {
	BaseURL          string
	APIKey           string
	AssetIDs         []int64
	UserID           int64
	BaseQuantity     int64
	Timeframe        string
	CandleLimit      int
	ShortPeriod      int
	LongPeriod       int
	BreakoutLookback int
	VolumeMultiplier float64
	TrendThreshold   float64
	Jitter           time.Duration
	RefreshInterval  time.Duration
	RequestTimeout   time.Duration
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
	for _, assetID := range cfg.AssetIDs {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		candles, err := client.Candles(ctx, assetID, cfg.Timeframe, cfg.CandleLimit)
		cancel()
		if err != nil {
			log.Printf("momentum: candles fetch failed asset=%d: %v", assetID, err)
			continue
		}
		signal, ok := bots.MomentumSignalFromCandles(candles, cfg.ShortPeriod, cfg.LongPeriod, cfg.BreakoutLookback, cfg.VolumeMultiplier, cfg.TrendThreshold)
		if !ok {
			continue
		}
		if cfg.Jitter > 0 {
			jitterNs := cfg.Jitter.Nanoseconds()
			if jitterNs > 0 {
				time.Sleep(time.Duration(rng.Int63n(jitterNs + 1)))
			}
		}
		multiplier := 1.0 + math.Min(signal.Strength*5, 2.0)
		quantity := bots.ClampQuantity(cfg.BaseQuantity, multiplier)
		req := bots.OrderRequest{
			AssetID:  assetID,
			UserID:   cfg.UserID,
			Side:     signal.Side,
			Type:     "MARKET",
			Quantity: quantity,
		}
		ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
		order, err := client.SubmitOrder(ctx, req)
		cancel()
		if err != nil {
			log.Printf("momentum: submit order failed asset=%d: %v", assetID, err)
			continue
		}
		log.Printf("momentum: order placed id=%d asset=%d side=%s qty=%d", order.ID, assetID, order.Side, order.Quantity)
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
	baseQuantity, err := bots.EnvInt64("ORDER_QUANTITY", 10)
	if err != nil {
		return config{}, err
	}
	if baseQuantity <= 0 {
		return config{}, errors.New("ORDER_QUANTITY must be positive")
	}
	timeframe := strings.TrimSpace(os.Getenv("TIMEFRAME"))
	if timeframe == "" {
		timeframe = "1m"
	}
	candleLimit, err := bots.EnvInt("CANDLE_LIMIT", 30)
	if err != nil {
		return config{}, err
	}
	if candleLimit <= 0 {
		return config{}, errors.New("CANDLE_LIMIT must be positive")
	}
	shortPeriod, err := bots.EnvInt("SHORT_PERIOD", 5)
	if err != nil {
		return config{}, err
	}
	if shortPeriod <= 0 {
		return config{}, errors.New("SHORT_PERIOD must be positive")
	}
	longPeriod, err := bots.EnvInt("LONG_PERIOD", 20)
	if err != nil {
		return config{}, err
	}
	if longPeriod <= 0 {
		return config{}, errors.New("LONG_PERIOD must be positive")
	}
	breakoutLookback, err := bots.EnvInt("BREAKOUT_LOOKBACK", 10)
	if err != nil {
		return config{}, err
	}
	if breakoutLookback <= 0 {
		return config{}, errors.New("BREAKOUT_LOOKBACK must be positive")
	}
	volumeMultiplier, err := bots.EnvFloat64("VOLUME_MULTIPLIER", 1.5)
	if err != nil {
		return config{}, err
	}
	if volumeMultiplier <= 0 {
		return config{}, errors.New("VOLUME_MULTIPLIER must be positive")
	}
	trendThreshold, err := bots.EnvFloat64("TREND_THRESHOLD", 0.01)
	if err != nil {
		return config{}, err
	}
	if trendThreshold < 0 {
		return config{}, errors.New("TREND_THRESHOLD must be non-negative")
	}
	jitter, err := bots.EnvDuration("JITTER", 0)
	if err != nil {
		return config{}, err
	}
	if jitter < 0 {
		return config{}, errors.New("JITTER must be non-negative")
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 3*time.Second)
	if err != nil {
		return config{}, err
	}
	if refreshInterval <= 0 {
		return config{}, errors.New("REFRESH_INTERVAL must be positive")
	}
	return config{
		BaseURL:          baseURL,
		APIKey:           apiKey,
		AssetIDs:         assetIDs,
		UserID:           userID,
		BaseQuantity:     baseQuantity,
		Timeframe:        timeframe,
		CandleLimit:      candleLimit,
		ShortPeriod:      shortPeriod,
		LongPeriod:       longPeriod,
		BreakoutLookback: breakoutLookback,
		VolumeMultiplier: volumeMultiplier,
		TrendThreshold:   trendThreshold,
		Jitter:           jitter,
		RefreshInterval:  refreshInterval,
		RequestTimeout:   requestTimeout,
	}, nil
}
