package api

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
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
	eng := engine.NewEngine(store)
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
	rng := rand.New(rand.NewSource(7))
	store.applyNewsImpact(item, eng, rng, cfg)
	store.mu.RLock()
	price := store.lastPrices[item.AssetID]
	store.mu.RUnlock()
	if price == 0 {
		t.Fatal("expected price impact to update last price")
	}
	store.mu.RLock()
	basePrice := store.basePrices[item.AssetID]
	store.mu.RUnlock()
	if basePrice == 0 {
		t.Fatal("expected base price for asset")
	}
	delta := int64(math.Round(float64(basePrice) * cfg.ImpactFactor * item.Sentiment))
	expected := basePrice + delta
	if price != expected {
		t.Fatalf("expected price %d, got %d", expected, price)
	}
}
