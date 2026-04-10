package api

import (
	"math"
	"strings"
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

	startARC := store.GetBalance(1, "ARC")
	startVDP := store.GetBalance(1, "VDP")

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
	if store.GetBalance(1, "ARC") != startARC-amountIn {
		t.Fatalf("unexpected ARC balance: %d", store.GetBalance(1, "ARC"))
	}
	if store.GetBalance(1, "VDP") != startVDP+result.AmountOut {
		t.Fatalf("unexpected VDP balance: %d", store.GetBalance(1, "VDP"))
	}
}

func TestUtilizationRate_KinkJumpModel(t *testing.T) {
	tests := []struct {
		name     string
		borrowed int64
		total    int64
		expected int64
	}{
		{
			name:     "0% utilization",
			borrowed: 0,
			total:    1000,
			expected: 10, // marginBaseRateBps
		},
		{
			name:     "35% utilization (halfway to kink)",
			borrowed: 350,
			total:    1000,
			expected: 30, // 10 + 40 * (3500/7000)
		},
		{
			name:     "70% utilization (at kink point)",
			borrowed: 700,
			total:    1000,
			expected: 50, // 10 + 40
		},
		{
			name:     "80% utilization (past kink point)",
			borrowed: 800,
			total:    1000,
			expected: 100, // 50 + 500 * (1000/10000)
		},
		{
			name:     "100% utilization (max)",
			borrowed: 1000,
			total:    1000,
			expected: 200, // 50 + 500 * (3000/10000)
		},
		{
			name:     "total <= 0",
			borrowed: 0,
			total:    0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := utilizationRate(tt.borrowed, tt.total)
			if rate != tt.expected {
				t.Errorf("utilizationRate(%d, %d) = %d, want %d", tt.borrowed, tt.total, rate, tt.expected)
			}
		})
	}
}

func TestSwapPoolRouterMultiHop(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)

	store.SetBalance(1, "VDP", 100_000)

	startVDP := store.GetBalance(1, "VDP")
	startBRB := store.GetBalance(1, "BRB")
	startARC := store.GetBalance(1, "ARC")

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
	if store.GetBalance(1, "VDP") != startVDP-amountIn {
		t.Fatalf("unexpected VDP balance: %d", store.GetBalance(1, "VDP"))
	}
	if store.GetBalance(1, "BRB") != startBRB+result.AmountOut {
		t.Fatalf("unexpected BRB balance: %d", store.GetBalance(1, "BRB"))
	}
	if store.GetBalance(1, "ARC") != startARC {
		t.Fatalf("expected ARC balance to be unchanged, got %d", store.GetBalance(1, "ARC"))
	}
}

func TestPoolPositionCollectsFeesOnClose(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	store.EnsureUser(2)

	pool := store.pools[1]
	spacing := tickSpacingForFee(pool.FeeBps)
	lower := pool.CurrentTick - spacing*2
	upper := pool.CurrentTick + spacing*2

	store.SetBalance(1, pool.BaseCurrency, 20_000)
	store.SetBalance(1, pool.QuoteCurrency, 20_000)
	store.SetBalance(2, pool.BaseCurrency, 20_000)
	store.SetBalance(2, pool.QuoteCurrency, 20_000)
	startBase := store.GetBalance(1, pool.BaseCurrency)
	startQuote := store.GetBalance(1, pool.QuoteCurrency)

	position, err := store.CreatePoolPosition(pool.ID, 1, 5_000, 5_000, lower, upper)
	if err != nil {
		t.Fatalf("create pool position failed: %v", err)
	}
	if len(store.PoolPositions(1)) != 1 {
		t.Fatalf("expected exactly one pool position, got %d", len(store.PoolPositions(1)))
	}

	result, err := store.SwapPool(pool.ID, 2, pool.BaseCurrency, pool.QuoteCurrency, 10_000)
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	if !strings.EqualFold(result.FromCurrency, pool.BaseCurrency) {
		t.Fatalf("expected fee currency to match base currency, got %s", result.FromCurrency)
	}
	if result.FeeAmount <= 0 {
		t.Fatalf("expected positive fee amount, got %d", result.FeeAmount)
	}

	if _, err := store.ClosePoolPosition(1, position.ID); err != nil {
		t.Fatalf("close pool position failed: %v", err)
	}

	if store.GetBalance(1, pool.BaseCurrency) != startBase+result.FeeAmount {
		t.Fatalf("expected base balance to include fees, got %d", store.GetBalance(1, pool.BaseCurrency))
	}
	if store.GetBalance(1, pool.QuoteCurrency) != startQuote {
		t.Fatalf("expected quote balance to return to original amount, got %d", store.GetBalance(1, pool.QuoteCurrency))
	}
}

func TestClosePoolPositionFailsWhenPoolMissing(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)
	pool := store.pools[1]
	spacing := tickSpacingForFee(pool.FeeBps)
	lower := pool.CurrentTick - spacing*2
	upper := pool.CurrentTick + spacing*2

	store.SetBalance(1, pool.BaseCurrency, 10_000)
	store.SetBalance(1, pool.QuoteCurrency, 10_000)

	position, err := store.CreatePoolPosition(pool.ID, 1, 1_000, 1_000, lower, upper)
	if err != nil {
		t.Fatalf("create pool position failed: %v", err)
	}

	store.mu.Lock()
	delete(store.pools, pool.ID)
	baseBefore := store.GetBalance(1, pool.BaseCurrency)
	quoteBefore := store.GetBalance(1, pool.QuoteCurrency)
	store.mu.Unlock()

	_, err = store.ClosePoolPosition(1, position.ID)
	if err == nil {
		t.Fatalf("expected error when pool is missing")
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.GetBalance(1, pool.BaseCurrency) != baseBefore {
		t.Fatalf("expected base balance unchanged, got %d", store.GetBalance(1, pool.BaseCurrency))
	}
	if store.GetBalance(1, pool.QuoteCurrency) != quoteBefore {
		t.Fatalf("expected quote balance unchanged, got %d", store.GetBalance(1, pool.QuoteCurrency))
	}
}
