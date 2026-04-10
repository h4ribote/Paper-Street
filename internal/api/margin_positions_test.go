package api

import (
	"context"
	"testing"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

func TestMarginPositionTopUp(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)
	eng := engine.NewEngine(nil, store)

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
	before := store.GetBalance(1, defaultCurrency)
	updated, err := store.AddMargin(1, position.ID, 50)
	if err != nil {
		t.Fatalf("failed to top up margin: %v", err)
	}
	if updated.MarginUsed != 250 {
		t.Fatalf("expected margin used 250, got %d", updated.MarginUsed)
	}
	if store.GetBalance(1, defaultCurrency) != before-50 {
		t.Fatalf("unexpected cash balance after top-up")
	}
}

func TestMarginLiquidationTriggered(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)
	store.EnsureUser(3)
	eng := engine.NewEngine(nil, store)

	store.EngineSubmitOrder = func(ctx context.Context, order *engine.Order) (engine.OrderResult, error) {
		return eng.SubmitOrder(ctx, order)
	}

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
		Price:    85,
	})
	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   3,
		Side:     engine.SideSell,
		Type:     engine.OrderTypeLimit,
		Quantity: 1,
		Price:    85,
	})

	// Wait for async liquidation to process
	for i := 0; i < 50; i++ {
		if len(store.MarginPositions(1)) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	positions = store.MarginPositions(1)
	if len(positions) != 0 {
		t.Fatalf("expected margin position to be liquidated, got %d", len(positions))
	}
	events := store.MarginLiquidations(1)
	if len(events) != 1 {
		t.Fatalf("expected 1 liquidation event, got %d", len(events))
	}

	remainingMargin := int64(50)              // 200 - 150 (loss)
	expectedFee := remainingMargin * 10 / 100 // 10% fee
	expectedPayout := remainingMargin - expectedFee

	event := events[0]
	if event.LiquidationFee != expectedFee {
		t.Fatalf("expected liquidation fee %d, got %d", expectedFee, event.LiquidationFee)
	}
	if event.RemainingMargin != expectedPayout {
		t.Fatalf("expected remaining margin %d, got %d", expectedPayout, event.RemainingMargin)
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

func TestMarginInterestAccrualUpdatesPool(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)
	eng := engine.NewEngine(nil, store)

	store.SetBalance(1, defaultCurrency, 2_000_000)

	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   1,
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 10_000,
		Price:    100,
		Leverage: 5,
	})
	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   2,
		Side:     engine.SideSell,
		Type:     engine.OrderTypeLimit,
		Quantity: 10_000,
		Price:    100,
	})

	positions := store.MarginPositions(1)
	if len(positions) != 1 {
		t.Fatalf("expected 1 margin position, got %d", len(positions))
	}
	position := positions[0]
	pool := store.marginPools[1]
	lastFeeAt := time.Now().UTC().Add(-time.Duration(marginInterestTick) * time.Millisecond).UnixMilli()
	setMarginPositionLastFeeAt(store, position.ID, lastFeeAt)
	poolBefore := pool.TotalCash

	positions = store.MarginPositions(1)
	if len(positions) != 1 {
		t.Fatalf("expected 1 margin position, got %d", len(positions))
	}
	updated := positions[0]
	elapsed := updated.UpdatedAt - lastFeeAt
	accrualMillis := (elapsed / marginInterestTick) * marginInterestTick
	expectedFee, ok := accruedMarginFee(position.BorrowedAmount, pool.CashRateBps, accrualMillis)
	if !ok || expectedFee <= 0 {
		t.Fatalf("expected positive fee, got %d", expectedFee)
	}
	poolAfter := store.marginPools[1]
	if poolAfter.TotalCash != poolBefore+expectedFee {
		t.Fatalf("expected pool cash %d, got %d", poolBefore+expectedFee, poolAfter.TotalCash)
	}
	if updated.AccumulatedFees != expectedFee {
		t.Fatalf("expected accumulated fees %d, got %d", expectedFee, updated.AccumulatedFees)
	}
	if updated.lastFeeAt != lastFeeAt+accrualMillis {
		t.Fatalf("expected lastFeeAt %d, got %d", lastFeeAt+accrualMillis, updated.lastFeeAt)
	}
}

func setMarginPositionLastFeeAt(store *MarketStore, positionID, lastFeeAt int64) {
	store.mu.Lock()
	position := store.marginPositions[positionID]
	position.lastFeeAt = lastFeeAt
	store.marginPositions[positionID] = position
	store.mu.Unlock()
}

func TestMarginMaintenanceLiquidatesOnFees(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)
	eng := engine.NewEngine(nil, store)

	store.EngineSubmitOrder = func(ctx context.Context, order *engine.Order) (engine.OrderResult, error) {
		return eng.SubmitOrder(ctx, order)
	}

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
	threshold := position.MarginUsed * marginLossCutBps / bpsDenominator
	setMarginPositionAccumulatedFees(store, position.ID, threshold+1)

	store.runMarginMaintenance()

	for i := 0; i < 50; i++ {
		store.mu.RLock()
		_, exists := store.marginPositions[position.ID]
		store.mu.RUnlock()
		if !exists {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	store.mu.RLock()
	_, exists := store.marginPositions[position.ID]
	store.mu.RUnlock()
	if exists {
		t.Fatalf("expected margin position to be liquidated")
	}
	events := store.MarginLiquidations(1)
	if len(events) != 1 {
		t.Fatalf("expected 1 liquidation event, got %d", len(events))
	}
	if events[0].PositionID != position.ID {
		t.Fatalf("unexpected liquidation position id: %d", events[0].PositionID)
	}
}

func setMarginPositionAccumulatedFees(store *MarketStore, positionID, fees int64) {
	store.mu.Lock()
	position := store.marginPositions[positionID]
	position.AccumulatedFees = fees
	store.marginPositions[positionID] = position
	store.mu.Unlock()
}
