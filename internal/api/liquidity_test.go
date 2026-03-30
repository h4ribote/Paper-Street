package api

import (
	"math"
	"testing"
)

func TestSwapPoolTickMath(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)

	pool := store.pools[1]
	pool.CurrentTick = 0
	pool.Liquidity = 1_000_000
	pool.FeeBps = poolFeeLowBps
	store.pools[1] = pool

	startARC := store.balances[1]["ARC"]
	startVDP := store.balances[1]["VDP"]

	amountIn := int64(10_000)
	result, err := store.SwapPool(1, 1, "ARC", "VDP", amountIn)
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}

	fee := amountIn * poolFeeLowBps / bpsDenominator
	amountAfterFee := float64(amountIn - fee)
	expectedTargetSqrt := 1 / (1 - amountAfterFee/float64(pool.Liquidity))
	expectedOut := float64(pool.Liquidity) * (expectedTargetSqrt - 1)
	expectedOutInt := int64(math.Floor(expectedOut))

	if result.AmountOut != expectedOutInt {
		t.Fatalf("unexpected amountOut: got %d, want %d", result.AmountOut, expectedOutInt)
	}
	if store.pools[1].CurrentTick <= 0 {
		t.Fatalf("expected tick to increase, got %d", store.pools[1].CurrentTick)
	}
	if store.pools[1].Liquidity != pool.Liquidity+result.FeeAmount {
		t.Fatalf("expected pool liquidity to include fees")
	}
	if store.balances[1]["ARC"] != startARC-amountIn {
		t.Fatalf("unexpected ARC balance: %d", store.balances[1]["ARC"])
	}
	if store.balances[1]["VDP"] != startVDP+result.AmountOut {
		t.Fatalf("unexpected VDP balance: %d", store.balances[1]["VDP"])
	}
}

func TestSwapPoolRouterMultiHop(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)

	startVDP := store.balances[1]["VDP"]
	startBRB := store.balances[1]["BRB"]
	startARC := store.balances[1]["ARC"]

	amountIn := int64(50_000)
	result, err := store.SwapPool(0, 1, "VDP", "BRB", amountIn)
	if err != nil {
		t.Fatalf("router swap failed: %v", err)
	}
	if result.PoolID != 0 {
		t.Fatalf("expected routed swap pool id 0, got %d", result.PoolID)
	}
	if result.AmountOut <= 0 {
		t.Fatalf("expected positive amountOut, got %d", result.AmountOut)
	}
	if result.FeeAmount <= 0 {
		t.Fatalf("expected positive fee, got %d", result.FeeAmount)
	}
	if store.balances[1]["VDP"] != startVDP-amountIn {
		t.Fatalf("unexpected VDP balance: %d", store.balances[1]["VDP"])
	}
	if store.balances[1]["BRB"] != startBRB+result.AmountOut {
		t.Fatalf("unexpected BRB balance: %d", store.balances[1]["BRB"])
	}
	if store.balances[1]["ARC"] != startARC {
		t.Fatalf("expected ARC balance to be unchanged, got %d", store.balances[1]["ARC"])
	}
}
