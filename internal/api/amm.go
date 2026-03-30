package api

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	tickSpacingLow      = int64(10)
	tickSpacingStandard = int64(50)
	fxBaseCurrency      = "ARC"
)

type liquiditySnapshot struct {
	baseline float64
	active   float64
	deltas   map[int64]float64
	ticks    []int64
}

type swapStepResult struct {
	pool        LiquidityPool
	from        string
	to          string
	amountIn    int64
	amountOut   int64
	feeAmount   int64
	updatedPool LiquidityPool
}

type swapRoutePlan struct {
	steps    []swapStepResult
	totalOut int64
	totalFee int64
	from     string
	to       string
	totalIn  int64
}

func tickSpacingForFee(feeBps int64) int64 {
	if feeBps == poolFeeLowBps {
		return tickSpacingLow
	}
	return tickSpacingStandard
}

func tickPrice(tick int64) float64 {
	return math.Pow(1.0001, float64(tick))
}

func tickSqrtPrice(tick int64) float64 {
	return math.Pow(1.0001, float64(tick)/2)
}

func tickFromSqrtPrice(sqrtPrice float64) int64 {
	if sqrtPrice <= 0 {
		return 0
	}
	price := sqrtPrice * sqrtPrice
	tick := math.Log(price) / math.Log(1.0001)
	return int64(math.Floor(tick))
}

func buildLiquiditySnapshot(pool LiquidityPool, positions []PoolPosition) liquiditySnapshot {
	deltas := make(map[int64]float64)
	var totalPositionLiquidity float64
	for _, position := range positions {
		if position.LowerTick >= position.UpperTick {
			continue
		}
		liquidity := float64(position.BaseAmount + position.QuoteAmount)
		if liquidity <= 0 {
			continue
		}
		totalPositionLiquidity += liquidity
		deltas[position.LowerTick] += liquidity
		deltas[position.UpperTick] -= liquidity
	}
	baseline := float64(pool.Liquidity) - totalPositionLiquidity
	if baseline < 0 {
		baseline = 0
	}
	ticks := make([]int64, 0, len(deltas))
	for tick := range deltas {
		ticks = append(ticks, tick)
	}
	sort.Slice(ticks, func(i, j int) bool { return ticks[i] < ticks[j] })
	active := baseline
	for _, tick := range ticks {
		if tick > pool.CurrentTick {
			break
		}
		active += deltas[tick]
	}
	return liquiditySnapshot{
		baseline: baseline,
		active:   active,
		deltas:   deltas,
		ticks:    ticks,
	}
}

func computePoolSwap(pool LiquidityPool, positions []PoolPosition, from, to string, amount int64, feeBps int64) (PoolSwapResult, LiquidityPool, error) {
	if amount <= 0 {
		return PoolSwapResult{}, LiquidityPool{}, errors.New("amount must be positive")
	}
	fee, err := calculateFeeBps(amount, feeBps)
	if err != nil {
		return PoolSwapResult{}, LiquidityPool{}, err
	}
	amountAfterFee := amount - fee
	if amountAfterFee <= 0 {
		return PoolSwapResult{}, LiquidityPool{}, errors.New("amount too small after fees")
	}
	snapshot := buildLiquiditySnapshot(pool, positions)
	if snapshot.active <= 0 {
		return PoolSwapResult{}, LiquidityPool{}, errors.New("insufficient liquidity")
	}
	currentTick := pool.CurrentTick
	currentSqrt := tickSqrtPrice(currentTick)
	liquidity := snapshot.active
	remaining := float64(amountAfterFee)
	var amountOut float64
	if stringsEqualFold(from, pool.BaseCurrency) && stringsEqualFold(to, pool.QuoteCurrency) {
		nextIndex := sort.Search(len(snapshot.ticks), func(i int) bool {
			return snapshot.ticks[i] > currentTick
		})
		for remaining > 0 {
			if nextIndex >= len(snapshot.ticks) {
				denom := 1/currentSqrt - remaining/liquidity
				if denom <= 0 {
					return PoolSwapResult{}, LiquidityPool{}, errors.New("insufficient liquidity to complete swap at current tick range")
				}
				targetSqrt := 1 / denom
				amountOut += liquidity * (targetSqrt - currentSqrt)
				currentSqrt = targetSqrt
				remaining = 0
				break
			}
			nextTick := snapshot.ticks[nextIndex]
			nextSqrt := tickSqrtPrice(nextTick)
			amountToNext := liquidity * (1/currentSqrt - 1/nextSqrt)
			if amountToNext < 0 {
				return PoolSwapResult{}, LiquidityPool{}, errors.New("invalid tick traversal")
			}
			if remaining <= amountToNext {
				denom := 1/currentSqrt - remaining/liquidity
				if denom <= 0 {
					return PoolSwapResult{}, LiquidityPool{}, errors.New("insufficient liquidity to complete swap at current tick range")
				}
				targetSqrt := 1 / denom
				amountOut += liquidity * (targetSqrt - currentSqrt)
				currentSqrt = targetSqrt
				remaining = 0
				break
			}
			amountOut += liquidity * (nextSqrt - currentSqrt)
			remaining -= amountToNext
			currentSqrt = nextSqrt
			currentTick = nextTick
			liquidity += snapshot.deltas[nextTick]
			if liquidity <= 0 {
				return PoolSwapResult{}, LiquidityPool{}, errors.New("insufficient liquidity")
			}
			nextIndex++
		}
	} else if stringsEqualFold(from, pool.QuoteCurrency) && stringsEqualFold(to, pool.BaseCurrency) {
		prevIndex := sort.Search(len(snapshot.ticks), func(i int) bool {
			return snapshot.ticks[i] >= currentTick
		}) - 1
		for remaining > 0 {
			if prevIndex < 0 {
				targetSqrt := currentSqrt - remaining/liquidity
				if targetSqrt <= 0 {
					return PoolSwapResult{}, LiquidityPool{}, errors.New("insufficient liquidity to complete swap at current tick range")
				}
				amountOut += liquidity * (1/targetSqrt - 1/currentSqrt)
				currentSqrt = targetSqrt
				remaining = 0
				break
			}
			prevTick := snapshot.ticks[prevIndex]
			prevSqrt := tickSqrtPrice(prevTick)
			amountToPrev := liquidity * (currentSqrt - prevSqrt)
			if amountToPrev < 0 {
				return PoolSwapResult{}, LiquidityPool{}, errors.New("invalid tick traversal")
			}
			if remaining <= amountToPrev {
				targetSqrt := currentSqrt - remaining/liquidity
				if targetSqrt <= 0 {
					return PoolSwapResult{}, LiquidityPool{}, errors.New("insufficient liquidity to complete swap at current tick range")
				}
				amountOut += liquidity * (1/targetSqrt - 1/currentSqrt)
				currentSqrt = targetSqrt
				remaining = 0
				break
			}
			amountOut += liquidity * (1/prevSqrt - 1/currentSqrt)
			remaining -= amountToPrev
			currentSqrt = prevSqrt
			currentTick = prevTick
			liquidity -= snapshot.deltas[prevTick]
			if liquidity <= 0 {
				return PoolSwapResult{}, LiquidityPool{}, errors.New("insufficient liquidity")
			}
			prevIndex--
		}
	} else {
		return PoolSwapResult{}, LiquidityPool{}, errors.New("currency pair not supported by pool")
	}
	if amountOut <= 0 || math.IsNaN(amountOut) || math.IsInf(amountOut, 0) {
		return PoolSwapResult{}, LiquidityPool{}, errors.New("swap produced invalid output")
	}
	amountOutInt := int64(math.Floor(amountOut))
	if amountOutInt <= 0 {
		return PoolSwapResult{}, LiquidityPool{}, errors.New("swap output too small")
	}
	pool.CurrentTick = tickFromSqrtPrice(currentSqrt)
	pool.Liquidity += fee
	result := PoolSwapResult{
		PoolID:       pool.ID,
		FromCurrency: strings.ToUpper(strings.TrimSpace(from)),
		ToCurrency:   strings.ToUpper(strings.TrimSpace(to)),
		AmountIn:     amount,
		AmountOut:    amountOutInt,
		FeeAmount:    fee,
	}
	return result, pool, nil
}

func (s *MarketStore) poolPositionsForPoolLocked(poolID int64) []PoolPosition {
	positions := make([]PoolPosition, 0)
	for _, position := range s.poolPositions {
		if position.PoolID == poolID {
			positions = append(positions, position)
		}
	}
	return positions
}

func (s *MarketStore) planDirectSwapLocked(from, to string, amount int64, userID int64) (swapRoutePlan, bool, error) {
	candidates := make([]swapStepResult, 0)
	liquidityWeights := make([]float64, 0)
	for _, pool := range s.pools {
		if !(stringsEqualFold(from, pool.BaseCurrency) && stringsEqualFold(to, pool.QuoteCurrency)) &&
			!(stringsEqualFold(from, pool.QuoteCurrency) && stringsEqualFold(to, pool.BaseCurrency)) {
			continue
		}
		positions := s.poolPositionsForPoolLocked(pool.ID)
		snapshot := buildLiquiditySnapshot(pool, positions)
		if snapshot.active <= 0 {
			continue
		}
		candidates = append(candidates, swapStepResult{pool: pool})
		liquidityWeights = append(liquidityWeights, snapshot.active)
	}
	if len(candidates) == 0 {
		return swapRoutePlan{}, false, nil
	}
	var bestPlan swapRoutePlan
	var bestOut int64
	var bestFee int64
	var bestSet bool
	for i := range candidates {
		pool := candidates[i].pool
		feeBps := s.fxFeeBpsForUserLocked(userID, pool.FeeBps)
		positions := s.poolPositionsForPoolLocked(pool.ID)
		result, updatedPool, err := computePoolSwap(pool, positions, from, to, amount, feeBps)
		if err != nil {
			continue
		}
		plan := swapRoutePlan{
			steps: []swapStepResult{{
				pool:        pool,
				from:        result.FromCurrency,
				to:          result.ToCurrency,
				amountIn:    result.AmountIn,
				amountOut:   result.AmountOut,
				feeAmount:   result.FeeAmount,
				updatedPool: updatedPool,
			}},
			totalOut: result.AmountOut,
			totalFee: result.FeeAmount,
			from:     result.FromCurrency,
			to:       result.ToCurrency,
			totalIn:  amount,
		}
		if !bestSet || result.AmountOut > bestOut || (result.AmountOut == bestOut && result.FeeAmount < bestFee) {
			bestPlan = plan
			bestOut = result.AmountOut
			bestFee = result.FeeAmount
			bestSet = true
		}
	}
	if !bestSet {
		return swapRoutePlan{}, false, errors.New("no viable swap route")
	}
	if len(candidates) == 1 {
		return bestPlan, true, nil
	}
	totalWeight := 0.0
	for _, weight := range liquidityWeights {
		totalWeight += weight
	}
	if totalWeight <= 0 {
		return bestPlan, true, nil
	}
	remaining := amount
	splitSteps := make([]swapStepResult, 0)
	var splitOut int64
	var splitFee int64
	splitValid := true
	for i, candidate := range candidates {
		pool := candidate.pool
		var portion int64
		if i == len(candidates)-1 {
			portion = remaining
		} else {
			portion = int64(math.Floor(float64(amount) * (liquidityWeights[i] / totalWeight)))
			if portion < 0 {
				portion = 0
			}
			if portion > remaining {
				portion = remaining
			}
		}
		remaining -= portion
		if portion == 0 {
			continue
		}
		feeBps := s.fxFeeBpsForUserLocked(userID, pool.FeeBps)
		positions := s.poolPositionsForPoolLocked(pool.ID)
		result, updatedPool, err := computePoolSwap(pool, positions, from, to, portion, feeBps)
		if err != nil {
			splitValid = false
			break
		}
		splitSteps = append(splitSteps, swapStepResult{
			pool:        pool,
			from:        result.FromCurrency,
			to:          result.ToCurrency,
			amountIn:    result.AmountIn,
			amountOut:   result.AmountOut,
			feeAmount:   result.FeeAmount,
			updatedPool: updatedPool,
		})
		splitOut += result.AmountOut
		splitFee += result.FeeAmount
	}
	if !splitValid || len(splitSteps) == 0 {
		return bestPlan, true, nil
	}
	if splitOut > bestOut || (splitOut == bestOut && splitFee < bestFee) {
		return swapRoutePlan{
			steps:    splitSteps,
			totalOut: splitOut,
			totalFee: splitFee,
			from:     strings.ToUpper(strings.TrimSpace(from)),
			to:       strings.ToUpper(strings.TrimSpace(to)),
			totalIn:  amount,
		}, true, nil
	}
	return bestPlan, true, nil
}

func (s *MarketStore) planMultiHopLocked(from, to string, amount int64, userID int64) (swapRoutePlan, bool, error) {
	if stringsEqualFold(from, fxBaseCurrency) || stringsEqualFold(to, fxBaseCurrency) {
		return swapRoutePlan{}, false, nil
	}
	firstPlan, ok, err := s.planDirectSwapLocked(from, fxBaseCurrency, amount, userID)
	if err != nil || !ok {
		return swapRoutePlan{}, false, err
	}
	secondPlan, ok, err := s.planDirectSwapLocked(fxBaseCurrency, to, firstPlan.totalOut, userID)
	if err != nil || !ok {
		return swapRoutePlan{}, false, err
	}
	steps := append([]swapStepResult{}, firstPlan.steps...)
	steps = append(steps, secondPlan.steps...)
	return swapRoutePlan{
		steps:    steps,
		totalOut: secondPlan.totalOut,
		totalFee: firstPlan.totalFee + secondPlan.totalFee,
		from:     strings.ToUpper(strings.TrimSpace(from)),
		to:       strings.ToUpper(strings.TrimSpace(to)),
		totalIn:  amount,
	}, true, nil
}

func (s *MarketStore) planSwapRouteLocked(from, to string, amount int64, userID int64) (swapRoutePlan, error) {
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return swapRoutePlan{}, errors.New("invalid currencies")
	}
	if stringsEqualFold(from, to) {
		return swapRoutePlan{}, errors.New("invalid currencies")
	}
	directPlan, directOK, directErr := s.planDirectSwapLocked(from, to, amount, userID)
	if directErr != nil {
		return swapRoutePlan{}, directErr
	}
	multiPlan, multiOK, multiErr := s.planMultiHopLocked(from, to, amount, userID)
	if multiErr != nil {
		return swapRoutePlan{}, multiErr
	}
	if directOK && multiOK {
		if multiPlan.totalOut > directPlan.totalOut {
			return multiPlan, nil
		}
		if multiPlan.totalOut == directPlan.totalOut && multiPlan.totalFee < directPlan.totalFee {
			return multiPlan, nil
		}
		return directPlan, nil
	}
	if directOK {
		return directPlan, nil
	}
	if multiOK {
		return multiPlan, nil
	}
	return swapRoutePlan{}, fmt.Errorf("no swap route for %s/%s", from, to)
}
