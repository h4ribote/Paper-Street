package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/bots"
)

const (
	scannerInitialBuffer = 64 * 1024
	scannerMaxBuffer     = 1024 * 1024
)

type config struct {
	BaseURL        string
	APIKey         string
	UserID         int64
	DefaultAssetID int64
	BaseQuantity   int64
	MinConfidence  float64
	Jitter         time.Duration
	RequestTimeout time.Duration
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	client := bots.NewAPIClient(cfg.BaseURL, cfg.APIKey, cfg.RequestTimeout)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, scannerInitialBuffer), scannerMaxBuffer)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event bots.NewsEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("invalid news payload: %v", err)
			continue
		}
		if event.AssetID == 0 {
			event.AssetID = cfg.DefaultAssetID
		}
		if event.AssetID == 0 {
			log.Printf("missing asset_id for event: %s", event.Headline)
			continue
		}
		reaction, ok := bots.ReactionOrder(event, cfg.BaseQuantity, cfg.MinConfidence)
		if !ok {
			continue
		}
		if cfg.Jitter > 0 {
			jitterNs := cfg.Jitter.Nanoseconds()
			if jitterNs > 0 {
				time.Sleep(time.Duration(rng.Int63n(jitterNs + 1)))
			}
		}
		req := bots.OrderRequest{
			AssetID:  event.AssetID,
			UserID:   cfg.UserID,
			Side:     reaction.Side,
			Type:     "MARKET",
			Quantity: reaction.Quantity,
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		order, err := client.SubmitOrder(ctx, req)
		cancel()
		if err != nil {
			log.Printf("submit order failed: %v", err)
			continue
		}
		log.Printf("news order placed id=%d side=%s qty=%d", order.ID, order.Side, order.Quantity)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("scanner error: %v", err)
	}
}

func loadConfig() (config, error) {
	baseURL := strings.TrimSpace(os.Getenv("API_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	if apiKey == "" {
		return config{}, errors.New("API_KEY is required")
	}
	userID, err := bots.EnvInt64("USER_ID", 1)
	if err != nil {
		return config{}, err
	}
	if userID <= 0 {
		return config{}, errors.New("USER_ID must be positive")
	}
	defaultAssetID, err := bots.EnvInt64("DEFAULT_ASSET_ID", 1)
	if err != nil {
		return config{}, err
	}
	baseQuantity, err := bots.EnvInt64("BASE_QUANTITY", 10)
	if err != nil {
		return config{}, err
	}
	if baseQuantity <= 0 {
		return config{}, errors.New("BASE_QUANTITY must be positive")
	}
	minConfidence, err := bots.EnvFloat64("MIN_CONFIDENCE", 0.1)
	if err != nil {
		return config{}, err
	}
	if minConfidence < 0 || minConfidence > 1 {
		return config{}, errors.New("MIN_CONFIDENCE must be between 0 and 1")
	}
	jitter, err := bots.EnvDuration("JITTER", 0)
	if err != nil {
		return config{}, err
	}
	if jitter < 0 {
		return config{}, errors.New("JITTER must be non-negative")
	}
	requestTimeout, err := bots.EnvDuration("REQUEST_TIMEOUT", 2*time.Second)
	if err != nil {
		return config{}, err
	}
	if requestTimeout <= 0 {
		return config{}, errors.New("REQUEST_TIMEOUT must be positive")
	}
	return config{
		BaseURL:        baseURL,
		APIKey:         apiKey,
		UserID:         userID,
		DefaultAssetID: defaultAssetID,
		BaseQuantity:   baseQuantity,
		MinConfidence:  minConfidence,
		Jitter:         jitter,
		RequestTimeout: requestTimeout,
	}, nil
}
