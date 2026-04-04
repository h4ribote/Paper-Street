package api

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	poolFeeLowBps        = int64(4)
	poolFeeStandardBps   = int64(20)
	indexFeeBps          = int64(10)
	indexArbBandBps      = int64(20)
	bpsDenominator       = int64(10_000)
	defaultPoolLiquidity = int64(15_000_000)
	// Margin rate model parameters (basis points per day).
	marginBaseRateBps  = int64(10)  // base rate at 0% utilization
	marginSlopeBps     = int64(40)  // slope until the kink point
	marginJumpBps      = int64(500) // slope after the kink point
	marginKinkPointBps = int64(7000)
)

type LiquidityPool struct {
	ID            int64  `json:"id"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	FeeBps        int64  `json:"fee_bps"`
	Liquidity     int64  `json:"liquidity"`
	CurrentTick   int64  `json:"current_tick"`
}

type PoolPosition struct {
	ID          int64 `json:"id"`
	PoolID      int64 `json:"pool_id"`
	UserID      int64 `json:"user_id"`
	BaseAmount  int64 `json:"base_amount"`
	QuoteAmount int64 `json:"quote_amount"`
	LowerTick   int64 `json:"lower_tick"`
	UpperTick   int64 `json:"upper_tick"`
	CreatedAt   int64 `json:"created_at"`
}

type PoolSwapResult struct {
	PoolID       int64  `json:"pool_id"`
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	AmountIn     int64  `json:"amount_in"`
	AmountOut    int64  `json:"amount_out"`
	FeeAmount    int64  `json:"fee_amount"`
}

type MarginPool struct {
	ID               int64 `json:"id"`
	AssetID          int64 `json:"asset_id"`
	TotalCash        int64 `json:"total_cash"`
	TotalAssets      int64 `json:"total_assets"`
	BorrowedCash     int64 `json:"borrowed_cash"`
	BorrowedAssets   int64 `json:"borrowed_assets"`
	TotalCashShares  int64 `json:"total_cash_shares"`
	TotalAssetShares int64 `json:"total_asset_shares"`
	CashRateBps      int64 `json:"cash_rate_bps"`
	AssetRateBps     int64 `json:"asset_rate_bps"`
}

type MarginProviderPosition struct {
	ID          int64 `json:"id"`
	PoolID      int64 `json:"pool_id"`
	UserID      int64 `json:"user_id"`
	CashShares  int64 `json:"cash_shares"`
	AssetShares int64 `json:"asset_shares"`
	CreatedAt   int64 `json:"created_at"`
}

type MarginSupplyResult struct {
	Pool     MarginPool             `json:"pool"`
	Position MarginProviderPosition `json:"position"`
}

type marginProviderKey struct {
	PoolID int64
	UserID int64
}

type IndexDefinition struct {
	AssetID    int64   `json:"asset_id"`
	Components []int64 `json:"components"`
	FeeBps     int64   `json:"fee_bps"`
}

type IndexActionResult struct {
	AssetID     int64   `json:"asset_id"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   int64   `json:"unit_price"`
	FeeAmount   int64   `json:"fee_amount"`
	TotalAmount int64   `json:"total_amount"`
	Components  []int64 `json:"components"`
	Action      string  `json:"action"`
}

func (s *MarketStore) Pools() []LiquidityPool {
	s.mu.RLock()
	pools := make([]LiquidityPool, 0, len(s.pools))
	for _, pool := range s.pools {
		pools = append(pools, pool)
	}
	s.mu.RUnlock()
	sort.Slice(pools, func(i, j int) bool { return pools[i].ID < pools[j].ID })
	return pools
}

func (s *MarketStore) Pool(poolID int64) (LiquidityPool, bool) {
	if poolID == 0 {
		return LiquidityPool{}, false
	}
	s.mu.RLock()
	pool, ok := s.pools[poolID]
	s.mu.RUnlock()
	return pool, ok
}

func (s *MarketStore) PoolPositions(userID int64) []PoolPosition {
	s.mu.RLock()
	positions := make([]PoolPosition, 0, len(s.poolPositions))
	for _, position := range s.poolPositions {
		if userID != 0 && position.UserID != userID {
			continue
		}
		positions = append(positions, position)
	}
	s.mu.RUnlock()
	sort.Slice(positions, func(i, j int) bool { return positions[i].ID < positions[j].ID })
	return positions
}

func (s *MarketStore) CreatePoolPosition(poolID, userID, baseAmount, quoteAmount, lowerTick, upperTick int64) (PoolPosition, error) {
	if poolID == 0 {
		return PoolPosition{}, errors.New("pool_id required")
	}
	if userID == 0 {
		return PoolPosition{}, errors.New("user_id required")
	}
	if baseAmount <= 0 && quoteAmount <= 0 {
		return PoolPosition{}, errors.New("base_amount or quote_amount required")
	}
	if lowerTick >= upperTick {
		return PoolPosition{}, errors.New("invalid tick range")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pool, ok := s.pools[poolID]
	if !ok {
		return PoolPosition{}, errors.New("pool not found")
	}
	tickSpacing := tickSpacingForFee(pool.FeeBps)
	if lowerTick%tickSpacing != 0 || upperTick%tickSpacing != 0 {
		return PoolPosition{}, fmt.Errorf("tick spacing must align to %d", tickSpacing)
	}
	s.ensureUserLocked(userID)
	if baseAmount > 0 {
		if s.balances[userID][pool.BaseCurrency] < baseAmount {
			return PoolPosition{}, errors.New("insufficient base currency balance")
		}
		s.balances[userID][pool.BaseCurrency] -= baseAmount
	}
	if quoteAmount > 0 {
		if s.balances[userID][pool.QuoteCurrency] < quoteAmount {
			return PoolPosition{}, errors.New("insufficient quote currency balance")
		}
		s.balances[userID][pool.QuoteCurrency] -= quoteAmount
	}
	s.nextPoolPosID++
	position := PoolPosition{
		ID:          s.nextPoolPosID,
		PoolID:      poolID,
		UserID:      userID,
		BaseAmount:  baseAmount,
		QuoteAmount: quoteAmount,
		LowerTick:   lowerTick,
		UpperTick:   upperTick,
		CreatedAt:   time.Now().UTC().UnixMilli(),
	}
	pool.Liquidity += baseAmount + quoteAmount
	s.pools[poolID] = pool
	s.poolPositions[position.ID] = position
	poolSnapshot := pool
	positionSnapshot := position
	go s.persistLiquidityPool(poolSnapshot)
	go s.persistPoolPosition(positionSnapshot)
	return position, nil
}

func (s *MarketStore) ClosePoolPosition(userID, positionID int64) (PoolPosition, error) {
	if positionID == 0 {
		return PoolPosition{}, errors.New("position_id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	position, ok := s.poolPositions[positionID]
	if !ok {
		return PoolPosition{}, errors.New("position not found")
	}
	if userID != 0 && position.UserID != userID {
		return PoolPosition{}, errors.New("position does not belong to user")
	}
	pool, ok := s.pools[position.PoolID]
	if !ok {
		return PoolPosition{}, fmt.Errorf("pool %d not found for position %d", position.PoolID, positionID)
	}
	pool.Liquidity -= position.BaseAmount + position.QuoteAmount
	s.pools[position.PoolID] = pool
	s.ensureUserLocked(position.UserID)
	s.balances[position.UserID][pool.BaseCurrency] += position.BaseAmount
	s.balances[position.UserID][pool.QuoteCurrency] += position.QuoteAmount
	delete(s.poolPositions, positionID)
	poolSnapshot := pool
	go s.persistLiquidityPool(poolSnapshot)
	go s.deletePoolPosition(positionID)
	return position, nil
}

func (s *MarketStore) SwapPool(poolID, userID int64, fromCurrency, toCurrency string, amount int64) (PoolSwapResult, error) {
	if userID == 0 {
		return PoolSwapResult{}, errors.New("user_id required")
	}
	if amount <= 0 {
		return PoolSwapResult{}, errors.New("amount must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	from := strings.ToUpper(strings.TrimSpace(fromCurrency))
	to := strings.ToUpper(strings.TrimSpace(toCurrency))
	if from == "" || to == "" || from == to {
		return PoolSwapResult{}, errors.New("invalid currencies")
	}
	s.ensureUserLocked(userID)
	if s.balances[userID][from] < amount {
		return PoolSwapResult{}, errors.New("insufficient balance")
	}
	if poolID == 0 {
		plan, err := s.planSwapRouteLocked(from, to, amount, userID)
		if err != nil {
			return PoolSwapResult{}, err
		}
		for _, step := range plan.steps {
			if s.balances[userID][step.from] < step.amountIn {
				return PoolSwapResult{}, errors.New("insufficient balance for routed swap")
			}
			s.balances[userID][step.from] -= step.amountIn
			s.balances[userID][step.to] += step.amountOut
			s.pools[step.pool.ID] = step.updatedPool
			s.distributePoolFeesLocked(step.updatedPool, step.from, step.feeAmount)
			poolSnapshot := s.pools[step.pool.ID]
			go s.persistLiquidityPool(poolSnapshot)
		}
		return PoolSwapResult{
			PoolID:       0,
			FromCurrency: plan.from,
			ToCurrency:   plan.to,
			AmountIn:     plan.totalIn,
			AmountOut:    plan.totalOut,
			FeeAmount:    plan.totalFee,
		}, nil
	}
	pool, ok := s.pools[poolID]
	if !ok {
		return PoolSwapResult{}, errors.New("pool not found")
	}
	valid := (from == pool.BaseCurrency && to == pool.QuoteCurrency) || (from == pool.QuoteCurrency && to == pool.BaseCurrency)
	if !valid {
		return PoolSwapResult{}, errors.New("currency pair not supported by pool")
	}
	feeBps := pool.FeeBps
	if userID != 0 {
		feeBps = s.fxFeeBpsForUserLocked(userID, feeBps)
	}
	positions := s.poolPositionsForPoolLocked(poolID)
	result, updatedPool, err := computePoolSwap(pool, positions, from, to, amount, feeBps)
	if err != nil {
		return PoolSwapResult{}, err
	}
	s.balances[userID][from] -= amount
	s.balances[userID][to] += result.AmountOut
	s.pools[poolID] = updatedPool
	s.distributePoolFeesLocked(updatedPool, from, result.FeeAmount)
	poolSnapshot := s.pools[poolID]
	go s.persistLiquidityPool(poolSnapshot)
	return result, nil
}

func (s *MarketStore) distributePoolFeesLocked(pool LiquidityPool, feeCurrency string, feeAmount int64) {
	if feeAmount <= 0 {
		return
	}
	feeCurrency = strings.ToUpper(strings.TrimSpace(feeCurrency))
	if feeCurrency == "" {
		return
	}
	baseCurrency := strings.ToUpper(strings.TrimSpace(pool.BaseCurrency))
	quoteCurrency := strings.ToUpper(strings.TrimSpace(pool.QuoteCurrency))
	if feeCurrency != baseCurrency && feeCurrency != quoteCurrency {
		return
	}
	type feeCandidate struct {
		id        int64
		liquidity int64
		share     int64
		remainder int64
	}
	candidates := make([]feeCandidate, 0)
	var totalLiquidity int64
	for _, position := range s.poolPositions {
		if position.PoolID != pool.ID {
			continue
		}
		if pool.CurrentTick < position.LowerTick || pool.CurrentTick >= position.UpperTick {
			continue
		}
		liquidity := position.BaseAmount + position.QuoteAmount
		if liquidity <= 0 {
			continue
		}
		candidates = append(candidates, feeCandidate{id: position.ID, liquidity: liquidity})
		totalLiquidity += liquidity
	}
	if totalLiquidity <= 0 {
		return
	}
	var distributed int64
	for i, candidate := range candidates {
		if candidate.liquidity <= 0 {
			continue
		}
		var share int64
		var remainder int64
		numerator, ok := safeMultiplyInt64(feeAmount, candidate.liquidity)
		if ok {
			share = numerator / totalLiquidity
			remainder = numerator % totalLiquidity
		} else {
			bigNumerator := new(big.Int).Mul(big.NewInt(feeAmount), big.NewInt(candidate.liquidity))
			bigTotal := big.NewInt(totalLiquidity)
			bigShare := new(big.Int)
			bigRemainder := new(big.Int)
			bigShare.DivMod(bigNumerator, bigTotal, bigRemainder)
			share = bigShare.Int64()
			remainder = bigRemainder.Int64()
		}
		candidates[i].share = share
		candidates[i].remainder = remainder
		distributed += share
	}
	remainder := feeAmount - distributed
	if remainder > 0 {
		if len(candidates) == 1 {
			candidates[0].share += remainder
		} else {
			sort.Slice(candidates, func(i, j int) bool {
				if candidates[i].remainder == candidates[j].remainder {
					return candidates[i].liquidity > candidates[j].liquidity
				}
				return candidates[i].remainder > candidates[j].remainder
			})
			limit := int(remainder)
			if limit > len(candidates) {
				limit = len(candidates)
			}
			for i := 0; i < limit; i++ {
				candidates[i].share++
			}
		}
	}
	for _, candidate := range candidates {
		if candidate.share <= 0 {
			continue
		}
		position := s.poolPositions[candidate.id]
		if feeCurrency == baseCurrency {
			position.BaseAmount += candidate.share
		} else {
			position.QuoteAmount += candidate.share
		}
		s.poolPositions[candidate.id] = position
		positionSnapshot := position
		go s.persistPoolPosition(positionSnapshot)
	}
}

func (s *MarketStore) MarginPools() []MarginPool {
	s.mu.RLock()
	pools := make([]MarginPool, 0, len(s.marginPools))
	for _, pool := range s.marginPools {
		pools = append(pools, pool)
	}
	s.mu.RUnlock()
	sort.Slice(pools, func(i, j int) bool { return pools[i].ID < pools[j].ID })
	return pools
}

func (s *MarketStore) MarginPoolsForUser(userID int64) []MarginPool {
	if userID == 0 {
		return s.MarginPools()
	}
	s.mu.Lock()
	s.ensureUserLocked(userID)
	pools := make([]MarginPool, 0, len(s.marginPools))
	for _, pool := range s.marginPools {
		pool.CashRateBps = s.marginRateForUserLocked(userID, pool.CashRateBps)
		pool.AssetRateBps = s.marginRateForUserLocked(userID, pool.AssetRateBps)
		pools = append(pools, pool)
	}
	s.mu.Unlock()
	sort.Slice(pools, func(i, j int) bool { return pools[i].ID < pools[j].ID })
	return pools
}

func (s *MarketStore) MarginPool(poolID int64) (MarginPool, bool) {
	if poolID == 0 {
		return MarginPool{}, false
	}
	s.mu.RLock()
	pool, ok := s.marginPools[poolID]
	s.mu.RUnlock()
	return pool, ok
}

func (s *MarketStore) MarginPoolForUser(poolID, userID int64) (MarginPool, bool) {
	if poolID == 0 {
		return MarginPool{}, false
	}
	if userID == 0 {
		return s.MarginPool(poolID)
	}
	s.mu.Lock()
	s.ensureUserLocked(userID)
	pool, ok := s.marginPools[poolID]
	if ok {
		pool.CashRateBps = s.marginRateForUserLocked(userID, pool.CashRateBps)
		pool.AssetRateBps = s.marginRateForUserLocked(userID, pool.AssetRateBps)
	}
	s.mu.Unlock()
	return pool, ok
}

func (s *MarketStore) SupplyMarginPool(poolID, userID, cashAmount, assetAmount int64) (MarginSupplyResult, error) {
	return s.updateMarginPool(poolID, userID, cashAmount, assetAmount, true)
}

func (s *MarketStore) WithdrawMarginPool(poolID, userID, cashAmount, assetAmount int64) (MarginSupplyResult, error) {
	return s.updateMarginPool(poolID, userID, cashAmount, assetAmount, false)
}

func (s *MarketStore) updateMarginPool(poolID, userID, cashAmount, assetAmount int64, isSupply bool) (MarginSupplyResult, error) {
	if poolID == 0 {
		return MarginSupplyResult{}, errors.New("pool_id required")
	}
	if userID == 0 {
		return MarginSupplyResult{}, errors.New("user_id required")
	}
	if cashAmount <= 0 && assetAmount <= 0 {
		return MarginSupplyResult{}, errors.New("cash_amount or asset_amount required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pool, ok := s.marginPools[poolID]
	if !ok {
		return MarginSupplyResult{}, errors.New("margin pool not found")
	}
	s.ensureUserLocked(userID)
	pool = normalizeMarginPoolShares(pool)
	positionKey := marginProviderKey{PoolID: poolID, UserID: userID}
	var position MarginProviderPosition
	currentCashTotal := pool.TotalCash
	currentAssetTotal := pool.TotalAssets
	var cashShares int64
	var assetShares int64
	if cashAmount > 0 {
		var err error
		cashShares, err = sharesForAmount(cashAmount, pool.TotalCashShares, currentCashTotal)
		if err != nil {
			return MarginSupplyResult{}, err
		}
		if cashShares <= 0 {
			return MarginSupplyResult{}, errors.New("cash amount too small for shares")
		}
	}
	if assetAmount > 0 {
		var err error
		assetShares, err = sharesForAmount(assetAmount, pool.TotalAssetShares, currentAssetTotal)
		if err != nil {
			return MarginSupplyResult{}, err
		}
		if assetShares <= 0 {
			return MarginSupplyResult{}, errors.New("asset amount too small for shares")
		}
	}
	if !isSupply {
		position = s.marginProviders[positionKey]
	}
	if assetAmount > 0 {
		if !isSupply {
			if position.ID == 0 || position.AssetShares < assetShares {
				return MarginSupplyResult{}, errors.New("insufficient asset shares")
			}
			if pool.TotalAssets-pool.BorrowedAssets < assetAmount {
				return MarginSupplyResult{}, errors.New("insufficient pool assets")
			}
		} else if s.positions[userID][pool.AssetID] < assetAmount {
			return MarginSupplyResult{}, errors.New("insufficient asset balance")
		}
	}
	if cashAmount > 0 {
		if !isSupply {
			if position.ID == 0 || position.CashShares < cashShares {
				return MarginSupplyResult{}, errors.New("insufficient cash shares")
			}
			if pool.TotalCash-pool.BorrowedCash < cashAmount {
				return MarginSupplyResult{}, errors.New("insufficient pool cash")
			}
		} else if s.balances[userID][defaultCurrency] < cashAmount {
			return MarginSupplyResult{}, errors.New("insufficient cash balance")
		}
	}
	if isSupply {
		position = s.marginProviders[positionKey]
	}
	if position.ID == 0 {
		if !isSupply {
			return MarginSupplyResult{}, errors.New("margin provider not found")
		}
		s.nextMarginPosID++
		position = MarginProviderPosition{
			ID:        s.nextMarginPosID,
			PoolID:    poolID,
			UserID:    userID,
			CreatedAt: time.Now().UTC().UnixMilli(),
		}
	}
	if cashAmount > 0 {
		if isSupply {
			s.balances[userID][defaultCurrency] -= cashAmount
			pool.TotalCash += cashAmount
			position.CashShares += cashShares
			pool.TotalCashShares += cashShares
		} else {
			pool.TotalCash -= cashAmount
			s.balances[userID][defaultCurrency] += cashAmount
			position.CashShares -= cashShares
			pool.TotalCashShares -= cashShares
		}
	}
	if assetAmount > 0 {
		if isSupply {
			s.positions[userID][pool.AssetID] -= assetAmount
			pool.TotalAssets += assetAmount
			position.AssetShares += assetShares
			pool.TotalAssetShares += assetShares
		} else {
			pool.TotalAssets -= assetAmount
			s.positions[userID][pool.AssetID] += assetAmount
			position.AssetShares -= assetShares
			pool.TotalAssetShares -= assetShares
		}
	}
	pool.CashRateBps, pool.AssetRateBps = marginRates(pool)
	s.marginProviders[positionKey] = position
	s.marginPools[poolID] = pool
	poolSnapshot := pool
	positionSnapshot := position
	go s.persistMarginPool(poolSnapshot)
	go s.persistMarginProvider(positionSnapshot)
	return MarginSupplyResult{Pool: pool, Position: position}, nil
}

func (s *MarketStore) CreateIndex(userID, assetID, quantity int64) (IndexActionResult, error) {
	return s.updateIndexHoldings(userID, assetID, quantity, true)
}

func (s *MarketStore) RedeemIndex(userID, assetID, quantity int64) (IndexActionResult, error) {
	return s.updateIndexHoldings(userID, assetID, quantity, false)
}

func (s *MarketStore) updateIndexHoldings(userID, assetID, quantity int64, isCreate bool) (IndexActionResult, error) {
	if assetID == 0 {
		return IndexActionResult{}, errors.New("asset_id required")
	}
	if userID == 0 {
		return IndexActionResult{}, errors.New("user_id required")
	}
	if quantity <= 0 {
		return IndexActionResult{}, errors.New("quantity must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	definition := s.ensureIndexLocked(assetID)
	unitPrice := s.indexUnitPriceLocked(definition)
	if unitPrice <= 0 {
		return IndexActionResult{}, errors.New("index price unavailable")
	}
	if err := s.ensureIndexArbitrageLocked(assetID, unitPrice, isCreate); err != nil {
		return IndexActionResult{}, err
	}
	amount, ok := safeMultiplyInt64(unitPrice, quantity)
	if !ok {
		return IndexActionResult{}, errors.New("amount overflow")
	}
	fee, err := calculateFeeBps(amount, definition.FeeBps)
	if err != nil {
		return IndexActionResult{}, err
	}
	s.ensureUserLocked(userID)
	inventory := s.ensureIndexInventoryLocked(assetID)
	if isCreate {
		if s.balances[userID][defaultCurrency] < fee {
			return IndexActionResult{}, errors.New("insufficient cash balance for fee")
		}
		for _, componentID := range definition.Components {
			if s.positions[userID][componentID] < quantity {
				return IndexActionResult{}, errors.New("insufficient component holdings")
			}
		}
		s.balances[userID][defaultCurrency] -= fee
		for _, componentID := range definition.Components {
			s.positions[userID][componentID] -= quantity
			inventory[componentID] += quantity
		}
		s.positions[userID][assetID] += quantity
	} else {
		if s.positions[userID][assetID] < quantity {
			return IndexActionResult{}, errors.New("insufficient index holdings")
		}
		if s.balances[userID][defaultCurrency] < fee {
			return IndexActionResult{}, errors.New("insufficient cash balance for fee")
		}
		for _, componentID := range definition.Components {
			if inventory[componentID] < quantity {
				return IndexActionResult{}, errors.New("insufficient index inventory")
			}
		}
		s.positions[userID][assetID] -= quantity
		s.balances[userID][defaultCurrency] -= fee
		for _, componentID := range definition.Components {
			inventory[componentID] -= quantity
			s.positions[userID][componentID] += quantity
		}
	}
	total := amount + fee
	if !isCreate {
		total = amount - fee
	}
	action := "create"
	if !isCreate {
		action = "redeem"
	}
	return IndexActionResult{
		AssetID:     assetID,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
		FeeAmount:   fee,
		TotalAmount: total,
		Components:  definition.Components,
		Action:      action,
	}, nil
}

func (s *MarketStore) seedPools() {
	pairs := []struct {
		quote string
		tick  int64
	}{
		{quote: "VDP", tick: 120},
		{quote: "BRB", tick: -50},
		{quote: "DRL", tick: 35},
		{quote: "VND", tick: 80},
		{quote: "ZMR", tick: -20},
		{quote: "RVD", tick: 15},
	}
	poolID := int64(1)
	for _, pair := range pairs {
		pools := []LiquidityPool{
			{ID: poolID, BaseCurrency: "ARC", QuoteCurrency: pair.quote, FeeBps: poolFeeLowBps, Liquidity: defaultPoolLiquidity, CurrentTick: pair.tick},
			{ID: poolID + 1, BaseCurrency: "ARC", QuoteCurrency: pair.quote, FeeBps: poolFeeStandardBps, Liquidity: defaultPoolLiquidity, CurrentTick: pair.tick},
		}
		poolID += 2
		for _, pool := range pools {
			if _, exists := s.pools[pool.ID]; exists {
				continue
			}
			s.pools[pool.ID] = pool
			s.currencies[pool.BaseCurrency] = struct{}{}
			s.currencies[pool.QuoteCurrency] = struct{}{}
			poolSnapshot := pool
			go s.persistLiquidityPool(poolSnapshot)
		}
	}
}

func (s *MarketStore) seedMarginPools() {
	if len(s.companyStates) == 0 {
		return
	}
	assetIDs := make([]int64, 0, len(s.companyStates))
	for assetID := range s.companyStates {
		assetIDs = append(assetIDs, assetID)
	}
	sort.Slice(assetIDs, func(i, j int) bool { return assetIDs[i] < assetIDs[j] })
	totalCash := int64(20_000_000)
	perPoolCash := totalCash
	if len(assetIDs) > 0 {
		perPoolCash = totalCash / int64(len(assetIDs))
		if perPoolCash == 0 {
			perPoolCash = totalCash
		}
	}
	poolID := int64(1)
	for _, assetID := range assetIDs {
		state := s.companyStates[assetID]
		if state == nil {
			continue
		}
		totalAssets := state.SharesIssued * 30 / 100
		pool := MarginPool{
			ID:             poolID,
			AssetID:        assetID,
			TotalCash:      perPoolCash,
			TotalAssets:    totalAssets,
			BorrowedCash:   0,
			BorrowedAssets: 0,
		}
		pool = normalizeMarginPoolShares(pool)
		pool.CashRateBps, pool.AssetRateBps = marginRates(pool)
		if _, exists := s.marginPools[pool.ID]; exists {
			poolID++
			continue
		}
		s.marginPools[pool.ID] = pool
		poolSnapshot := pool
		go s.persistMarginPool(poolSnapshot)
		poolID++
	}
}

func (s *MarketStore) seedIndexes() {
	if _, ok := s.indexes[201]; ok {
		// Already loaded (e.g. from DB) — skip re-seeding.
		return
	}
	indexAsset := models.Asset{
		ID:     201,
		Symbol: "TRI",
		Name:   "TriCore Index",
		Type:   "INDEX",
		Sector: "MIXED",
	}
	s.assets[indexAsset.ID] = indexAsset
	definition := IndexDefinition{
		AssetID:    indexAsset.ID,
		Components: []int64{101, 102, 103},
		FeeBps:     indexFeeBps,
	}
	s.indexes[indexAsset.ID] = definition
	s.basePrices[indexAsset.ID] = s.indexUnitPriceLocked(definition)
	if s.queries != nil {
		ctx, cancel := s.dbContext()
		defer cancel()
		if err := s.queries.UpsertAsset(ctx, indexAsset, s.basePrices[indexAsset.ID]); err != nil {
			log.Printf("db upsert asset %d: %v", indexAsset.ID, err)
			return
		}
	}
	s.persistIndex(definition)
}

func (s *MarketStore) ensureIndexLocked(assetID int64) IndexDefinition {
	if def, ok := s.indexes[assetID]; ok {
		return def
	}
	components := make([]int64, 0, len(s.assets))
	for id, asset := range s.assets {
		if asset.Type == "INDEX" {
			continue
		}
		components = append(components, id)
	}
	sort.Slice(components, func(i, j int) bool { return components[i] < components[j] })
	definition := IndexDefinition{
		AssetID:    assetID,
		Components: components,
		FeeBps:     indexFeeBps,
	}
	s.indexes[assetID] = definition
	asset := s.ensureAssetLocked(assetID)
	asset.Type = "INDEX"
	asset.Sector = stringOrDefault(asset.Sector, "MIXED")
	if asset.Symbol == "" || strings.HasPrefix(asset.Symbol, "ASSET-") {
		asset.Symbol = fmt.Sprintf("INDEX-%d", assetID)
	}
	s.assets[assetID] = asset
	s.basePrices[assetID] = s.indexUnitPriceLocked(definition)
	go s.persistIndex(definition)
	return definition
}

func (s *MarketStore) ensureIndexInventoryLocked(assetID int64) map[int64]int64 {
	inventory, ok := s.indexHoldings[assetID]
	if !ok {
		inventory = make(map[int64]int64)
		s.indexHoldings[assetID] = inventory
	}
	return inventory
}

func (s *MarketStore) ensureIndexArbitrageLocked(assetID, unitPrice int64, isCreate bool) error {
	band, err := calculateFeeBps(unitPrice, indexArbBandBps)
	if err != nil {
		return err
	}
	upper, ok := safeAddInt64(unitPrice, band)
	if !ok {
		return errors.New("arbitrage band overflow")
	}
	lower, ok := safeAddInt64(unitPrice, -band)
	if !ok {
		return errors.New("arbitrage band overflow")
	}
	marketPrice := s.marketPriceLocked(assetID)
	if isCreate {
		if marketPrice <= upper {
			return errors.New("index price must exceed arbitrage band for creation")
		}
		return nil
	}
	if marketPrice >= lower {
		return errors.New("index price must fall below arbitrage band for redemption")
	}
	return nil
}

func (s *MarketStore) assetCurrencyLocked(assetID int64) string {
	if state := s.companyStates[assetID]; state != nil {
		return currencyForCountry(state.Country, fxBaseCurrency)
	}
	if asset, ok := s.assets[assetID]; ok {
		if asset.Sector != "" {
			country := s.defaultCountryForSector(asset.Sector)
			return currencyForCountry(country, fxBaseCurrency)
		}
	}
	return fxBaseCurrency
}

func (s *MarketStore) indexFXRateLocked(currency string, rateByCurrency map[string]int64) int64 {
	if currency == "" || stringsEqualFold(currency, fxBaseCurrency) {
		return fxTheoreticalScale
	}
	rate := rateByCurrency[strings.ToUpper(strings.TrimSpace(currency))]
	if rate == 0 {
		return fxTheoreticalScale
	}
	return rate
}

func (s *MarketStore) indexUnitPriceLocked(definition IndexDefinition) int64 {
	rateByCurrency := make(map[string]int64, len(s.theoreticalFXRates))
	for _, rate := range s.theoreticalFXRates {
		if rate.Rate <= 0 || !stringsEqualFold(rate.QuoteCurrency, fxBaseCurrency) {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(rate.BaseCurrency))
		if key == "" {
			continue
		}
		rateByCurrency[key] = rate.Rate
	}
	var total int64
	for _, assetID := range definition.Components {
		price := s.marketPriceLocked(assetID)
		converted := price
		currency := s.assetCurrencyLocked(assetID)
		if currency != "" && !stringsEqualFold(currency, fxBaseCurrency) {
			rate := s.indexFXRateLocked(currency, rateByCurrency)
			product, ok := safeMultiplyInt64(price, rate)
			if ok {
				converted = product / fxTheoreticalScale
			}
		}
		next, ok := safeAddInt64(total, converted)
		if !ok {
			return defaultAssetPrice
		}
		total = next
	}
	if total == 0 {
		total = defaultAssetPrice
	}
	return total
}

// IndexInfo holds the definition and current NAV of an index.
type IndexInfo struct {
	Definition IndexDefinition `json:"definition"`
	NAV        int64           `json:"nav"`
}

// Index returns the definition and current NAV for a single index asset, or false if not found.
func (s *MarketStore) Index(assetID int64) (IndexInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	def, ok := s.indexes[assetID]
	if !ok {
		return IndexInfo{}, false
	}
	return IndexInfo{Definition: def, NAV: s.indexUnitPriceLocked(def)}, true
}

// Indexes returns definition and current NAV for all known index assets.
func (s *MarketStore) Indexes() []IndexInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]IndexInfo, 0, len(s.indexes))
	for _, def := range s.indexes {
		result = append(result, IndexInfo{Definition: def, NAV: s.indexUnitPriceLocked(def)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Definition.AssetID < result[j].Definition.AssetID })
	return result
}

func marginRates(pool MarginPool) (cashRate int64, assetRate int64) {
	cashRate = utilizationRate(pool.BorrowedCash, pool.TotalCash)
	assetRate = utilizationRate(pool.BorrowedAssets, pool.TotalAssets)
	return cashRate, assetRate
}

func normalizeMarginPoolShares(pool MarginPool) MarginPool {
	if pool.TotalCashShares == 0 && pool.TotalCash > 0 {
		pool.TotalCashShares = pool.TotalCash
	}
	if pool.TotalAssetShares == 0 && pool.TotalAssets > 0 {
		pool.TotalAssetShares = pool.TotalAssets
	}
	return pool
}

// utilizationRate returns the daily rate in basis points using a kinked model:
// base + slope*(util/kink) for util <= kink, and base + slope + jump*(util-kink) beyond it.
func utilizationRate(borrowed, total int64) int64 {
	if total <= 0 {
		return 0
	}
	utilBps := borrowed * bpsDenominator / total
	if utilBps <= marginKinkPointBps {
		product, ok := safeMultiplyInt64(marginSlopeBps, utilBps)
		if !ok {
			return marginBaseRateBps
		}
		return marginBaseRateBps + product/marginKinkPointBps
	}
	excess := utilBps - marginKinkPointBps
	// Jump multiplier applies to the absolute utilization delta in bps (normalized by 10,000).
	product, ok := safeMultiplyInt64(marginJumpBps, excess)
	if !ok {
		return marginBaseRateBps + marginSlopeBps
	}
	return marginBaseRateBps + marginSlopeBps + product/bpsDenominator
}

func sharesForAmount(amount, totalShares, totalLiquidity int64) (int64, error) {
	if totalShares <= 0 || totalLiquidity <= 0 {
		return amount, nil
	}
	product, ok := safeMultiplyInt64(amount, totalShares)
	if !ok {
		return 0, errors.New("share overflow")
	}
	return product / totalLiquidity, nil
}

func calculateFeeBps(amount, bps int64) (int64, error) {
	fee, ok := safeMultiplyInt64(amount, bps)
	if !ok {
		return 0, errors.New("fee overflow")
	}
	return fee / bpsDenominator, nil
}
