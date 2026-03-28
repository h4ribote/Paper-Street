package bots

import (
	"testing"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

func TestMidPriceFromBook(t *testing.T) {
	snapshot := engine.OrderBookSnapshot{
		Bids: []engine.Level{{Price: 100, Quantity: 5}},
		Asks: []engine.Level{{Price: 110, Quantity: 3}},
	}
	mid := MidPrice(snapshot, 0)
	if mid != 105 {
		t.Fatalf("expected mid 105, got %d", mid)
	}
}

func TestMidPriceFallbacks(t *testing.T) {
	snapshot := engine.OrderBookSnapshot{LastPrice: 120}
	mid := MidPrice(snapshot, 0)
	if mid != 120 {
		t.Fatalf("expected mid 120, got %d", mid)
	}
	mid = MidPrice(engine.OrderBookSnapshot{}, 90)
	if mid != 90 {
		t.Fatalf("expected mid 90 fallback, got %d", mid)
	}
}

func TestQuoteFromMid(t *testing.T) {
	quote := QuoteFromMid(100, 50)
	if quote.BidPrice != 100 || quote.AskPrice != 101 {
		t.Fatalf("expected bid 100 ask 101, got bid %d ask %d", quote.BidPrice, quote.AskPrice)
	}
}

func TestQuoteFromMidWithSpread(t *testing.T) {
	quote := QuoteFromMid(20000, 50)
	if quote.BidPrice != 19950 || quote.AskPrice != 20050 {
		t.Fatalf("expected bid 19950 ask 20050, got bid %d ask %d", quote.BidPrice, quote.AskPrice)
	}
}
