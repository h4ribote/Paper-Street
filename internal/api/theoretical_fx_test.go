package api

import (
	"math"
	"testing"
	"time"
)

func TestComputeTheoreticalFXScoreMatchesExample(t *testing.T) {
	score := computeTheoreticalFXScore(300, 200, 400, 250, 300, 200)
	if math.Abs(score-1.4) > 0.0001 {
		t.Fatalf("expected theoretical score 1.40, got %.4f", score)
	}
	rate := int64(math.Round(score * float64(fxTheoreticalScale)))
	if rate != 140 {
		t.Fatalf("expected scaled rate 140, got %d", rate)
	}
}

func TestBuildTheoreticalFXRatesLocked(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	store.mu.Lock()
	store.macroIndicators = []MacroIndicator{
		{Country: fxArcadiaCountry, Type: macroTypeGDPGrowth, Value: 200},
		{Country: fxArcadiaCountry, Type: macroTypeInterest, Value: 250},
		{Country: fxArcadiaCountry, Type: macroTypeCPI, Value: 200},
		{Country: "Neo Venice", Type: macroTypeGDPGrowth, Value: 300},
		{Country: "Neo Venice", Type: macroTypeInterest, Value: 400},
		{Country: "Neo Venice", Type: macroTypeCPI, Value: 300},
	}
	rates := store.buildTheoreticalFXRatesLocked(now)
	store.mu.Unlock()
	if len(rates) != 1 {
		t.Fatalf("expected 1 theoretical FX rate, got %d", len(rates))
	}
	rate := rates[0]
	if rate.BaseCurrency != "VND" {
		t.Fatalf("expected base currency VND, got %q", rate.BaseCurrency)
	}
	if rate.QuoteCurrency != fxBaseCurrency {
		t.Fatalf("expected quote currency %s, got %q", fxBaseCurrency, rate.QuoteCurrency)
	}
	if rate.Rate != 140 {
		t.Fatalf("expected rate 140, got %d", rate.Rate)
	}
	if rate.UpdatedAt != now.UnixMilli() {
		t.Fatalf("expected updated_at %d, got %d", now.UnixMilli(), rate.UpdatedAt)
	}
}
