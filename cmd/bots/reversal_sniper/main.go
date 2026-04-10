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
	BaseURL         string
	APIKey          string
	AssetIDs        []int64
	UserID          int64
	Quantity        int64
	Timeframe       string
	CandleLimit     int
	RSIPeriod       int
	BandSigma       float64
	Jitter          time.Duration
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
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
		// Asset check
		ctxA, cancelA := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		_, errA := client.Asset(ctxA, assetID)
		cancelA()
		if errA != nil {
			continue // skip non-existent
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		candles, err := client.Candles(ctx, assetID, cfg.Timeframe, cfg.CandleLimit)
		cancel()
		if err != nil {
			log.Printf("reversal sniper: candles fetch failed asset=%d: %v", assetID, err)
			continue
		}
		signal, ok := bots.ReversalSignalFromCandles(candles, cfg.RSIPeriod, cfg.BandSigma)
		if !ok {
			continue
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
			if a.AssetID == assetID {
				inventory = a.Quantity
				break
			}
		}

		qty := cfg.Quantity
		lastPrice := candles[0].Close
		if signal.Side == "BUY" {
			if cash < lastPrice*qty {
				qty = cash / lastPrice
			}
		} else {
			if inventory < qty {
				qty = inventory
			}
		}

		if qty <= 0 {
			continue
		}

		if cfg.Jitter > 0 {
			jitterNs := cfg.Jitter.Nanoseconds()
			if jitterNs > 0 {
				time.Sleep(time.Duration(rng.Int63n(jitterNs + 1)))
			}
		}
		req := bots.OrderRequest{
			AssetID:  assetID,
			UserID:   cfg.UserID,
			Side:     signal.Side,
			Type:     "MARKET",
			Quantity: qty,
		}
		ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
		order, err := client.SubmitOrder(ctx, req)
		cancel()
		if err != nil {
			log.Printf("reversal sniper: submit order failed asset=%d: %v", assetID, err)
			continue
		}
		log.Printf("reversal sniper: order placed id=%d asset=%d side=%s qty=%d", order.ID, assetID, order.Side, order.Quantity)
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
	rsiPeriod, err := bots.EnvInt("RSI_PERIOD", 14)
	if err != nil {
		return config{}, err
	}
	if rsiPeriod <= 0 {
		return config{}, errors.New("RSI_PERIOD must be positive")
	}
	bandSigma, err := bots.EnvFloat64("BAND_SIGMA", 3)
	if err != nil {
		return config{}, err
	}
	if bandSigma <= 0 {
		return config{}, errors.New("BAND_SIGMA must be positive")
	}
	jitter, err := bots.EnvDuration("JITTER", 0)
	if err != nil {
		return config{}, err
	}
	if jitter < 0 {
		return config{}, errors.New("JITTER must be non-negative")
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
		AssetIDs:        assetIDs,
		UserID:          userID,
		Quantity:        quantity,
		Timeframe:       timeframe,
		CandleLimit:     candleLimit,
		RSIPeriod:       rsiPeriod,
		BandSigma:       bandSigma,
		Jitter:          jitter,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}
