package bots

import "testing"

func TestBollingerBands(t *testing.T) {
	values := []int64{100, 102, 101, 103, 104}
	mid, upper, lower, ok := Bollinger(values, 5, 2)
	if !ok {
		t.Fatal("expected bollinger to compute")
	}
	if mid <= 0 || upper <= mid || lower >= mid {
		t.Fatalf("unexpected bands mid=%f upper=%f lower=%f", mid, upper, lower)
	}
}

func TestRSI(t *testing.T) {
	values := []int64{100, 101, 102, 103, 104, 105, 106, 107}
	rsi := RSI(values, 5)
	if rsi <= 50 {
		t.Fatalf("expected rsi > 50 for uptrend, got %f", rsi)
	}
}

func TestVolumeSpike(t *testing.T) {
	candles := []Candle{
		{Volume: 10},
		{Volume: 12},
		{Volume: 30},
	}
	if !VolumeSpike(candles, 3, 1.5) {
		t.Fatal("expected volume spike")
	}
}
