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
	BaseURL         string
	APIKey          string
	AssetIDs        []int64
	UserID          int64
	Quantity        int64
	Timeframe       string
	CandleLimit     int
	TrendShort      int
	TrendLong       int
	BandPeriod      int
	BandSigma       float64
	PlaceTakeProfit bool
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
}

type orderState struct {
	buyID    int64
	sellID   int64
	lastBuy  int64
	lastSell int64
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	client := bots.NewAPIClient(cfg.BaseURL, cfg.APIKey, cfg.RequestTimeout)
	state := make(map[int64]*orderState)
	ticker := time.NewTicker(cfg.RefreshInterval)
	defer ticker.Stop()
	for {
		runOnce(client, cfg, state)
		<-ticker.C
	}
}

func runOnce(client *bots.APIClient, cfg config, state map[int64]*orderState) {
	for _, assetID := range cfg.AssetIDs {
		// Asset check
		ctxA, cancelA := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		_, errA := client.Asset(ctxA, assetID)
		cancelA()
		if errA != nil {
			continue // skip non-existent
		}

		entry := state[assetID]
		if entry == nil {
			entry = &orderState{}
			state[assetID] = entry
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		candles, err := client.Candles(ctx, assetID, cfg.Timeframe, cfg.CandleLimit)
		cancel()
		if err != nil {
			log.Printf("dip buyer: candles fetch failed asset=%d: %v", assetID, err)
			continue
		}
		signal, ok := bots.DipBuySignalFromCandles(candles, cfg.TrendShort, cfg.TrendLong, cfg.BandPeriod, cfg.BandSigma)
		if !ok {
			cancelOrder(client, cfg.RequestTimeout, assetID, entry.buyID)
			cancelOrder(client, cfg.RequestTimeout, assetID, entry.sellID)
			entry.buyID = 0
			entry.sellID = 0
			entry.lastBuy = 0
			entry.lastSell = 0
			continue
		}
		if entry.buyID != 0 && entry.lastBuy == signal.BuyPrice && (!cfg.PlaceTakeProfit || entry.lastSell == signal.TakeProfit) {
			continue
		}
		cancelOrder(client, cfg.RequestTimeout, assetID, entry.buyID)
		cancelOrder(client, cfg.RequestTimeout, assetID, entry.sellID)
		entry.buyID = 0
		entry.sellID = 0
		entry.lastBuy = signal.BuyPrice
		entry.lastSell = signal.TakeProfit

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
			if a.Asset.ID == assetID {
				inventory = a.Quantity
				break
			}
		}

		buyQty := cfg.Quantity
		if cash < signal.BuyPrice*buyQty {
			buyQty = cash / signal.BuyPrice
		}

		sellQty := cfg.Quantity
		if inventory < sellQty {
			sellQty = inventory
		}

		if buyQty > 0 {
			buyReq := bots.OrderRequest{
				AssetID:  assetID,
				UserID:   cfg.UserID,
				Side:     "BUY",
				Type:     "LIMIT",
				Quantity: buyQty,
				Price:    signal.BuyPrice,
			}
			ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
			order, err := client.SubmitOrder(ctx, buyReq)
			cancel()
			if err != nil {
				log.Printf("dip buyer: submit buy failed asset=%d: %v", assetID, err)
			} else {
				entry.buyID = order.ID
				log.Printf("dip buyer: buy order placed id=%d asset=%d qty=%d price=%d", order.ID, assetID, buyQty, order.Price)
			}
		}

		if cfg.PlaceTakeProfit && signal.TakeProfit > signal.BuyPrice && sellQty > 0 {
			sellReq := bots.OrderRequest{
				AssetID:  assetID,
				UserID:   cfg.UserID,
				Side:     "SELL",
				Type:     "LIMIT",
				Quantity: sellQty,
				Price:    signal.TakeProfit,
			}
			ctx, cancel = context.WithTimeout(context.Background(), cfg.RequestTimeout)
			order, err := client.SubmitOrder(ctx, sellReq)
			cancel()
			if err != nil {
				log.Printf("dip buyer: submit sell failed asset=%d: %v", assetID, err)
			} else {
				entry.sellID = order.ID
				log.Printf("dip buyer: take profit order placed id=%d asset=%d qty=%d price=%d", order.ID, assetID, sellQty, order.Price)
			}
		}
	}
}

func cancelOrder(client *bots.APIClient, timeout time.Duration, assetID int64, orderID int64) {
	if orderID == 0 || client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.CancelOrder(ctx, assetID, orderID); err != nil {
		log.Printf("dip buyer: cancel order %d failed: %v", orderID, err)
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
		timeframe = "5m"
	}
	candleLimit, err := bots.EnvInt("CANDLE_LIMIT", 40)
	if err != nil {
		return config{}, err
	}
	if candleLimit <= 0 {
		return config{}, errors.New("CANDLE_LIMIT must be positive")
	}
	trendShort, err := bots.EnvInt("TREND_SHORT", 5)
	if err != nil {
		return config{}, err
	}
	if trendShort <= 0 {
		return config{}, errors.New("TREND_SHORT must be positive")
	}
	trendLong, err := bots.EnvInt("TREND_LONG", 20)
	if err != nil {
		return config{}, err
	}
	if trendLong <= 0 {
		return config{}, errors.New("TREND_LONG must be positive")
	}
	bandPeriod, err := bots.EnvInt("BAND_PERIOD", 20)
	if err != nil {
		return config{}, err
	}
	if bandPeriod <= 0 {
		return config{}, errors.New("BAND_PERIOD must be positive")
	}
	bandSigma, err := bots.EnvFloat64("BAND_SIGMA", 2)
	if err != nil {
		return config{}, err
	}
	if bandSigma <= 0 {
		return config{}, errors.New("BAND_SIGMA must be positive")
	}
	placeTakeProfit := true
	if raw := strings.TrimSpace(os.Getenv("PLACE_TAKE_PROFIT")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return config{}, err
		}
		placeTakeProfit = parsed
	}
	refreshInterval, err := bots.EnvDuration("REFRESH_INTERVAL", 5*time.Second)
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
		TrendShort:      trendShort,
		TrendLong:       trendLong,
		BandPeriod:      bandPeriod,
		BandSigma:       bandSigma,
		PlaceTakeProfit: placeTakeProfit,
		RefreshInterval: refreshInterval,
		RequestTimeout:  requestTimeout,
	}, nil
}
