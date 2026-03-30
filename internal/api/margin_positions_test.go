package api

import (
	"testing"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

func TestMarginPositionTopUp(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)
	eng := engine.NewEngine(store)

	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   1,
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 10,
		Price:    100,
		Leverage: 5,
	})
	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   2,
		Side:     engine.SideSell,
		Type:     engine.OrderTypeLimit,
		Quantity: 10,
		Price:    100,
	})

	positions := store.MarginPositions(1)
	if len(positions) != 1 {
		t.Fatalf("expected 1 margin position, got %d", len(positions))
	}
	position := positions[0]
	if position.MarginUsed != 200 {
		t.Fatalf("unexpected margin used: %d", position.MarginUsed)
	}
	before := store.balances[1][defaultCurrency]
	updated, err := store.AddMargin(1, position.ID, 50)
	if err != nil {
		t.Fatalf("failed to top up margin: %v", err)
	}
	if updated.MarginUsed != 250 {
		t.Fatalf("expected margin used 250, got %d", updated.MarginUsed)
	}
	if store.balances[1][defaultCurrency] != before-50 {
		t.Fatalf("unexpected cash balance after top-up")
	}
}

func TestMarginLiquidationTriggered(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)
	store.EnsureUser(3)
	eng := engine.NewEngine(store)

	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   1,
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 10,
		Price:    100,
		Leverage: 5,
	})
	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   2,
		Side:     engine.SideSell,
		Type:     engine.OrderTypeLimit,
		Quantity: 10,
		Price:    100,
	})

	positions := store.MarginPositions(1)
	if len(positions) != 1 {
		t.Fatalf("expected 1 margin position, got %d", len(positions))
	}
	positionID := positions[0].ID

	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   2,
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 1,
		Price:    60,
	})
	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   3,
		Side:     engine.SideSell,
		Type:     engine.OrderTypeLimit,
		Quantity: 1,
		Price:    60,
	})

	positions = store.MarginPositions(1)
	if len(positions) != 0 {
		t.Fatalf("expected margin position to be liquidated, got %d", len(positions))
	}
	events := store.MarginLiquidations(1)
	if len(events) != 1 {
		t.Fatalf("expected 1 liquidation event, got %d", len(events))
	}
	if events[0].PositionID != positionID {
		t.Fatalf("unexpected liquidation position id: %d", events[0].PositionID)
	}
	if events[0].UserID != 1 {
		t.Fatalf("unexpected liquidation user id: %d", events[0].UserID)
	}
	if events[0].AssetID != 101 {
		t.Fatalf("unexpected liquidation asset id: %d", events[0].AssetID)
	}
	if events[0].Side != engine.SideBuy {
		t.Fatalf("unexpected liquidation side: %s", events[0].Side)
	}
	if events[0].Quantity != 10 {
		t.Fatalf("unexpected liquidation quantity: %d", events[0].Quantity)
	}
	if events[0].LossRatioBps < marginLossCutBps {
		t.Fatalf("expected loss ratio above threshold, got %d", events[0].LossRatioBps)
	}
}
