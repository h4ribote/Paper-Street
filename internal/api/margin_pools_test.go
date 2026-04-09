package api

import "testing"

func TestMarginPoolSupplyWithdrawShares(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)

	poolID := int64(1)
	startCash := store.GetBalance(1, defaultCurrency)
	initialPool := store.marginPools[poolID]
	initialCashShares := initialPool.TotalCashShares
	if initialCashShares == 0 && initialPool.TotalCash > 0 {
		initialCashShares = initialPool.TotalCash
	}

	result, err := store.SupplyMarginPool(poolID, 1, 1_000, 0)
	if err != nil {
		t.Fatalf("failed to supply margin pool: %v", err)
	}
	if result.Position.CashShares != 1_000 {
		t.Fatalf("expected 1000 cash shares, got %d", result.Position.CashShares)
	}
	if result.Pool.TotalCash != initialPool.TotalCash+1_000 {
		t.Fatalf("unexpected pool cash: %d", result.Pool.TotalCash)
	}
	if result.Pool.TotalCashShares != initialCashShares+1_000 {
		t.Fatalf("unexpected pool cash shares: %d", result.Pool.TotalCashShares)
	}
	if store.GetBalance(1, defaultCurrency) != startCash-1_000 {
		t.Fatalf("unexpected cash balance after supply")
	}

	result, err = store.WithdrawMarginPool(poolID, 1, 400, 0)
	if err != nil {
		t.Fatalf("failed to withdraw margin pool: %v", err)
	}
	if result.Position.CashShares != 600 {
		t.Fatalf("expected 600 cash shares, got %d", result.Position.CashShares)
	}
	if result.Pool.TotalCash != initialPool.TotalCash+600 {
		t.Fatalf("unexpected pool cash after withdraw: %d", result.Pool.TotalCash)
	}
	if result.Pool.TotalCashShares != initialCashShares+600 {
		t.Fatalf("unexpected pool cash shares after withdraw: %d", result.Pool.TotalCashShares)
	}
	if store.GetBalance(1, defaultCurrency) != startCash-600 {
		t.Fatalf("unexpected cash balance after withdraw")
	}

	if _, err := store.WithdrawMarginPool(poolID, 1, 700, 0); err == nil {
		t.Fatalf("expected error when withdrawing more than shares")
	}
}
