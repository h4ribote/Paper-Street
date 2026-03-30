package api

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	poolFeeLowBps      = int64(4)
	poolFeeStandardBps = int64(20)
	indexFeeBps        = int64(10)
	bpsDenominator     = int64(10_000)
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
	if ok {
		pool.Liquidity -= position.BaseAmount + position.QuoteAmount
		s.pools[position.PoolID] = pool
	}
	s.ensureUserLocked(position.UserID)
	s.balances[position.UserID][pool.BaseCurrency] += position.BaseAmount
	s.balances[position.UserID][pool.QuoteCurrency] += position.QuoteAmount
	delete(s.poolPositions, positionID)
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
	return result, nil
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
	position := s.marginProviders[positionKey]
	prevCashTotal := pool.TotalCash
	prevAssetTotal := pool.TotalAssets
	var cashShares int64
	var assetShares int64
	if cashAmount > 0 {
		var err error
		cashShares, err = sharesForAmount(cashAmount, pool.TotalCashShares, prevCashTotal)
		if err != nil {
			return MarginSupplyResult{}, err
		}
		if cashShares <= 0 {
			return MarginSupplyResult{}, errors.New("cash amount too small for shares")
		}
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
	if assetAmount > 0 {
		var err error
		assetShares, err = sharesForAmount(assetAmount, pool.TotalAssetShares, prevAssetTotal)
		if err != nil {
			return MarginSupplyResult{}, err
		}
		if assetShares <= 0 {
			return MarginSupplyResult{}, errors.New("asset amount too small for shares")
		}
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
	amount, ok := safeMultiplyInt64(unitPrice, quantity)
	if !ok {
		return IndexActionResult{}, errors.New("amount overflow")
	}
	fee, err := calculateFeeBps(amount, definition.FeeBps)
	if err != nil {
		return IndexActionResult{}, err
	}
	total := amount + fee
	s.ensureUserLocked(userID)
	if isCreate {
		if s.balances[userID][defaultCurrency] < total {
			return IndexActionResult{}, errors.New("insufficient cash balance")
		}
		s.balances[userID][defaultCurrency] -= total
		s.positions[userID][assetID] += quantity
	} else {
		if s.positions[userID][assetID] < quantity {
			return IndexActionResult{}, errors.New("insufficient index holdings")
		}
		s.positions[userID][assetID] -= quantity
		s.balances[userID][defaultCurrency] += amount - fee
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
	pools := []LiquidityPool{
		{ID: 1, BaseCurrency: "ARC", QuoteCurrency: "VDP", FeeBps: poolFeeLowBps, Liquidity: 2_000_000, CurrentTick: 120},
		{ID: 2, BaseCurrency: "ARC", QuoteCurrency: "VDP", FeeBps: poolFeeStandardBps, Liquidity: 1_000_000, CurrentTick: 120},
		{ID: 3, BaseCurrency: "ARC", QuoteCurrency: "BRB", FeeBps: poolFeeLowBps, Liquidity: 900_000, CurrentTick: -50},
		{ID: 4, BaseCurrency: "ARC", QuoteCurrency: "BRB", FeeBps: poolFeeStandardBps, Liquidity: 1_200_000, CurrentTick: -50},
	}
	for _, pool := range pools {
		s.pools[pool.ID] = pool
		s.currencies[pool.BaseCurrency] = struct{}{}
		s.currencies[pool.QuoteCurrency] = struct{}{}
	}
}

func (s *MarketStore) seedMarginPools() {
	pools := []MarginPool{
		{ID: 1, AssetID: 101, TotalCash: 5_000_000, TotalAssets: 25_000, BorrowedCash: 1_000_000, BorrowedAssets: 4_000},
		{ID: 2, AssetID: 102, TotalCash: 4_000_000, TotalAssets: 18_000, BorrowedCash: 800_000, BorrowedAssets: 3_000},
	}
	for _, pool := range pools {
		pool = normalizeMarginPoolShares(pool)
		pool.CashRateBps, pool.AssetRateBps = marginRates(pool)
		s.marginPools[pool.ID] = pool
	}
}

func (s *MarketStore) seedIndexes() {
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
	return definition
}

func (s *MarketStore) indexUnitPriceLocked(definition IndexDefinition) int64 {
	var total int64
	for _, assetID := range definition.Components {
		price := s.lastPrices[assetID]
		if price == 0 {
			price = s.basePrices[assetID]
		}
		total += price
	}
	if total == 0 {
		total = defaultAssetPrice
	}
	return total
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
