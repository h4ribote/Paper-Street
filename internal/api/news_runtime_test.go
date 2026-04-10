package api

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

func TestRandomNewsItem(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 31, 12, 0, 0, 0, time.UTC)
	rng := rand.New(rand.NewSource(42))
	item, ok := store.randomNewsItem(now, rng)
	if !ok {
		t.Fatal("expected random news item")
	}
	if item.Headline == "" {
		t.Fatal("expected headline to be populated")
	}
	if item.Category == "" {
		t.Fatal("expected category to be populated")
	}
	if item.Impact == "" {
		t.Fatal("expected impact to be populated")
	}
	if item.Sentiment < -1 || item.Sentiment > 1 {
		t.Fatalf("unexpected sentiment range: %f", item.Sentiment)
	}
	if sentimentImpact(item.Sentiment) != item.Impact {
		t.Fatalf("expected impact %s to match sentiment %f", item.Impact, item.Sentiment)
	}
}

func TestApplyNewsImpactMovesPrice(t *testing.T) {
	store := NewMarketStore()
	eng := engine.NewEngine(nil, store)
	cfg := NewsEngineConfig{
		Interval:      time.Second,
		BaseQuantity:  10,
		MinConfidence: 0.1,
		ImpactFactor:  0.1,
		ImpactJitter:  0,
	}
	item := NewsItem{
		AssetID:   101,
		Sentiment: 0.5,
		Impact:    "POSITIVE",
	}
	store.mu.RLock()
	basePrice := store.marketPriceLocked(item.AssetID)
	store.mu.RUnlock()
	if basePrice == 0 {
		t.Fatal("expected base price for asset")
	}
	rng := rand.New(rand.NewSource(7))
	store.applyNewsImpact(item, eng, rng, cfg)
	store.mu.RLock()
	price := store.lastPrices[item.AssetID]
	store.mu.RUnlock()
	if price == 0 {
		t.Fatal("expected price impact to update last price")
	}
	delta := int64(math.Round(float64(basePrice) * cfg.ImpactFactor * item.Sentiment))
	expected := basePrice + delta
	if price != expected {
		t.Fatalf("expected price %d, got %d", expected, price)
	}
}

func TestNewsEngineReactsToNews(t *testing.T) {
	store := NewMarketStore()
	eng := engine.NewEngine(nil, store)
	store.EngineSubmitOrder = eng.SubmitOrder

	// Provide an asset that bots can react on.
	assetID := int64(101)

	// Since randomNewsItem checks what assets are available via store.assetIDs(), we inject 101.
	store.mu.Lock()
	if store.currencies == nil {
		store.currencies = make(map[string]struct{})
		store.currencies["ARC"] = struct{}{}
	}
	s := models.Asset{ID: assetID, Name: "Test Inc", Symbol: "TEST"}

	store.updateAssetLocked(s, 1000)
	store.mu.Unlock()

	// Update the price explicitly to be valid.
	store.AddExecution(engine.Execution{
		AssetID:  assetID,
		Price:    1000,
		Quantity: 1,
	})

	cfg := NewsEngineConfig{
		Interval:      10 * time.Millisecond,
		BaseQuantity:  50,
		MinConfidence: 0.1,
		ImpactFactor:  0.05,
		ImpactJitter:  0.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Need a dummy item matching our ID for deterministic testing, but StartNewsEngine uses random.
	// Oh actually StartNewsEngine generates a brand new item periodically using randomNewsItem()!
	// We'll let it run and wait a bit so a news item gets generated and bots jump in.
	go StartNewsEngine(ctx, store, eng, cfg)

	time.Sleep(100 * time.Millisecond) // Let engine tick a few times.

	executions := store.Executions(0, 100)
	hasReactorOrder := false
	hasLiquidityOrder := false
	for _, exec := range executions {
		if exec.TakerUserID == newsReactorUserID || exec.MakerUserID == newsReactorUserID {
			hasReactorOrder = true
		}
		if exec.TakerUserID == newsLiquidityUserID || exec.MakerUserID == newsLiquidityUserID {
			hasLiquidityOrder = true
		}
	}

	if !hasReactorOrder || !hasLiquidityOrder {
		t.Fatalf("Expected executions from bot reactors after news tick: hasReactorOrder=%t, hasLiquidityOrder=%t, len(executions)=%d", hasReactorOrder, hasLiquidityOrder, len(executions))
	}
}
