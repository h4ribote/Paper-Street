package bots

import "testing"

func TestMomentumSignal(t *testing.T) {
	candles := []Candle{
		{High: 100, Low: 90, Close: 95, Volume: 10},
		{High: 110, Low: 100, Close: 105, Volume: 12},
		{High: 120, Low: 110, Close: 115, Volume: 50},
	}
	signal, ok := MomentumSignalFromCandles(candles, 2, 3, 2, 1.2, 0.01)
	if !ok {
		t.Fatal("expected momentum signal")
	}
	if signal.Side != "BUY" {
		t.Fatalf("expected BUY, got %s", signal.Side)
	}
}

func TestDipBuySignal(t *testing.T) {
	candles := []Candle{
		{High: 110, Low: 100, Close: 105, Volume: 10},
		{High: 115, Low: 102, Close: 108, Volume: 11},
		{High: 112, Low: 99, Close: 100, Volume: 12},
		{High: 120, Low: 105, Close: 110, Volume: 13},
		{High: 118, Low: 104, Close: 104, Volume: 14},
	}
	signal, ok := DipBuySignalFromCandles(candles, 2, 4, 5, 2)
	if !ok {
		t.Fatal("expected dip buy signal")
	}
	if signal.BuyPrice <= 0 {
		t.Fatal("expected positive buy price")
	}
}

func TestReversalSignal(t *testing.T) {
	candles := []Candle{
		{High: 100, Low: 90, Close: 95, Volume: 10},
		{High: 120, Low: 110, Close: 118, Volume: 10},
		{High: 140, Low: 130, Close: 135, Volume: 10},
		{High: 200, Low: 180, Close: 195, Volume: 10},
		{High: 210, Low: 190, Close: 205, Volume: 10},
		{High: 360, Low: 330, Close: 350, Volume: 10},
	}
	signal, ok := ReversalSignalFromCandles(candles, 3, 1)
	if !ok {
		t.Fatal("expected reversal signal")
	}
	if signal.Side != "SELL" {
		t.Fatalf("expected SELL, got %s", signal.Side)
	}
}

func TestGridLevels(t *testing.T) {
	levels := GridLevels(100, 50, 2)
	if len(levels) != 4 {
		t.Fatalf("expected 4 levels, got %d", len(levels))
	}
	if levels[0].Side != "BUY" || levels[1].Side != "SELL" {
		t.Fatalf("unexpected grid sides: %#v", levels)
	}
}

func TestImpactAdjustedPrice(t *testing.T) {
	price := ImpactAdjustedPrice(100, 0.02, 1000, 10000, 0.5, "BUY")
	if price <= 100 {
		t.Fatalf("expected impact price above mid, got %d", price)
	}
	price = ImpactAdjustedPrice(100, 0.02, 1000, 10000, 0.5, "SELL")
	if price >= 100 {
		t.Fatalf("expected impact price below mid, got %d", price)
	}
}

func TestYieldPreference(t *testing.T) {
	choice := YieldPreference(0.05, 0.01, 0.01)
	if choice != "BOND" {
		t.Fatalf("expected bond preference, got %s", choice)
	}
}

func TestConsumerLuxuryShare(t *testing.T) {
	share := ConsumerLuxuryShare(120)
	if share <= 0.2 {
		t.Fatalf("expected higher luxury share, got %f", share)
	}
}
