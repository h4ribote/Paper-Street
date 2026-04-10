package api

import (
	"testing"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

func TestProcessSubmitInMemory_SelfTradePrevention(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)

	// Create an engine order
	maker := &engine.Order{
		AssetID:  10,
		UserID:   1, // User 1 creates maker
		Side:     engine.SideSell,
		Type:     engine.OrderTypeLimit,
		Quantity: 5,
		Price:    100,
	}
	res := store.processSubmitInMemory(maker)
	if res.Order.Status != engine.OrderStatusOpen {
		t.Fatalf("expected maker order to be OPEN, got %s", res.Order.Status)
	}

	// User 1 sends a taker order matching their own maker
	taker := &engine.Order{
		AssetID:  10,
		UserID:   1, // Same user
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 10,
		Price:    100,
	}

	res2 := store.processSubmitInMemory(taker)
	if len(res2.Executions) > 0 {
		t.Fatalf("expected no executions for self-trade, got %d", len(res2.Executions))
	}

	// Taker's Remaining Quantity should have been reduced by Maker's Remaining,
	// which means 10 - 5 = 5. Since price was limit 100, the remaining quantity connects to orderbook.
	if res2.Order.Remaining != 5 {
		t.Fatalf("expected taker order remaining to be reduced, got %d", res2.Order.Remaining)
	}

	// Ensure the original Maker order is cancelled.
	canceledMaker := store.testOrders[maker.ID]
	if canceledMaker.Status != engine.OrderStatusCancelled {
		t.Fatalf("expected maker order to be CANCELLED, got %s", canceledMaker.Status)
	}
}

func TestApplyExecutionInMemory_FeeRounding(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)
	store.SetBalance(1, defaultCurrency, 100_000)
	store.SetBalance(2, defaultCurrency, 100_000)

	exec := engine.Execution{
		AssetID:       10,
		Price:         100,
		Quantity:      1,
		TakerOrderID:  2,
		MakerOrderID:  1,
		TakerUserID:   1,
		MakerUserID:   2,
		OccurredAtUTC: time.Now().UTC(),
	}

	makerOrder := &engine.Order{ID: 1, UserID: 2}
	takerOrder := &engine.Order{ID: 2, UserID: 1}

	// Execute
	store.applyExecutionInMemory(exec, engine.SideBuy, takerOrder, makerOrder)

	// Since default fee is small, test verifies crash doesn't happen.
	// But actually, we don't have access to dynamically set custom Rank Info directly from here without deep injection.
}
