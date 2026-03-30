package api

import (
	"errors"
	"sort"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

const (
	marginLossCutBps   = int64(7_500)
	liquidationFeeBps  = int64(1_000)
	marginLeverageMin  = int64(1)
	marginLeverageMax  = int64(5)
	millisecondsPerDay = int64(24 * time.Hour / time.Millisecond)
	marginInterestTick = int64(2 * time.Hour / time.Millisecond)
	marginPriceMissing = int64(0)
)

type MarginPosition struct {
	ID              int64       `json:"id"`
	UserID          int64       `json:"user_id"`
	AssetID         int64       `json:"asset_id"`
	Side            engine.Side `json:"side"`
	Quantity        int64       `json:"quantity"`
	EntryPrice      int64       `json:"entry_price"`
	CurrentPrice    int64       `json:"current_price"`
	Leverage        int64       `json:"leverage"`
	MarginUsed      int64       `json:"margin_used"`
	BorrowedAmount  int64       `json:"borrowed_amount"`
	AccumulatedFees int64       `json:"accumulated_fees"`
	UnrealizedLoss  int64       `json:"unrealized_loss"`
	LossRatioBps    int64       `json:"loss_ratio_bps"`
	CreatedAt       int64       `json:"created_at"`
	UpdatedAt       int64       `json:"updated_at"`
	lastFeeAt       int64
}

type MarginLiquidation struct {
	ID              int64       `json:"id"`
	PositionID      int64       `json:"position_id"`
	UserID          int64       `json:"user_id"`
	AssetID         int64       `json:"asset_id"`
	Side            engine.Side `json:"side"`
	Quantity        int64       `json:"quantity"`
	LossRatioBps    int64       `json:"loss_ratio_bps"`
	LiquidationFee  int64       `json:"liquidation_fee"`
	RemainingMargin int64       `json:"remaining_margin"`
	OccurredAt      int64       `json:"occurred_at"`
}

func normalizeLeverage(leverage int64) int64 {
	if leverage < marginLeverageMin {
		return marginLeverageMin
	}
	return leverage
}

func requiredMargin(notional, leverage int64) (int64, error) {
	if notional <= 0 {
		return 0, errors.New("notional must be positive")
	}
	if leverage < marginLeverageMin {
		return 0, errors.New("invalid leverage")
	}
	if leverage == 1 {
		return notional, nil
	}
	margin := notional / leverage
	if notional%leverage != 0 {
		margin++
	}
	if margin <= 0 {
		return 0, errors.New("margin calculation failed")
	}
	return margin, nil
}

func (s *MarketStore) MarginPositions(userID int64) []MarginPosition {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().UnixMilli()
	positions := make([]MarginPosition, 0, len(s.marginPositions))
	for id, position := range s.marginPositions {
		if userID != 0 && position.UserID != userID {
			continue
		}
		position = s.refreshMarginPositionLocked(position, now)
		s.marginPositions[id] = position
		positions = append(positions, position)
	}
	sort.Slice(positions, func(i, j int) bool { return positions[i].ID < positions[j].ID })
	return positions
}

func (s *MarketStore) MarginLiquidations(userID int64) []MarginLiquidation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if userID == 0 {
		events := make([]MarginLiquidation, len(s.marginLiquidations))
		copy(events, s.marginLiquidations)
		return events
	}
	events := make([]MarginLiquidation, 0, len(s.marginLiquidations))
	for _, event := range s.marginLiquidations {
		if event.UserID == userID {
			events = append(events, event)
		}
	}
	return events
}

func (s *MarketStore) AddMargin(userID, positionID, amount int64) (MarginPosition, error) {
	if positionID == 0 {
		return MarginPosition{}, errors.New("position_id required")
	}
	if amount <= 0 {
		return MarginPosition{}, errors.New("amount must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	position, ok := s.marginPositions[positionID]
	if !ok {
		return MarginPosition{}, errors.New("margin position not found")
	}
	if userID != 0 && position.UserID != userID {
		return MarginPosition{}, errors.New("position does not belong to user")
	}
	s.ensureUserLocked(position.UserID)
	if s.balances[position.UserID][defaultCurrency] < amount {
		return MarginPosition{}, errors.New("insufficient cash balance")
	}
	now := time.Now().UTC().UnixMilli()
	position = s.refreshMarginPositionLocked(position, now)
	position.MarginUsed += amount
	position.LossRatioBps = s.lossRatioBps(position.MarginUsed, position.UnrealizedLoss+position.AccumulatedFees)
	position.UpdatedAt = now
	s.balances[position.UserID][defaultCurrency] -= amount
	s.marginPositions[position.ID] = position
	return position, nil
}

func (s *MarketStore) refreshMarginPositionLocked(position MarginPosition, now int64) MarginPosition {
	position = s.accrueMarginFeesLocked(position, now)
	currentPrice := s.priceForAssetLocked(position.AssetID)
	loss := s.unrealizedLoss(position, currentPrice)
	lossRatioBps := s.lossRatioBps(position.MarginUsed, loss+position.AccumulatedFees)
	position.CurrentPrice = currentPrice
	position.UnrealizedLoss = loss
	position.LossRatioBps = lossRatioBps
	position.UpdatedAt = now
	return position
}

func (s *MarketStore) priceForAssetLocked(assetID int64) int64 {
	if assetID == 0 {
		return marginPriceMissing
	}
	price := s.lastPrices[assetID]
	if price == 0 {
		price = s.basePrices[assetID]
	}
	return price
}

func (s *MarketStore) accrueMarginFeesLocked(position MarginPosition, now int64) MarginPosition {
	if position.lastFeeAt == 0 {
		position.lastFeeAt = now
		return position
	}
	if now <= position.lastFeeAt {
		return position
	}
	if position.BorrowedAmount <= 0 {
		position.lastFeeAt = now
		return position
	}
	elapsed := now - position.lastFeeAt
	accruals := elapsed / marginInterestTick
	if accruals <= 0 {
		return position
	}
	accrualMillis := accruals * marginInterestTick
	rate := s.marginBorrowRateLocked(position)
	if rate <= 0 || accrualMillis <= 0 {
		position.lastFeeAt += accrualMillis
		return position
	}
	borrowedValue := position.BorrowedAmount
	assetBorrowed := int64(0)
	if position.Side == engine.SideSell {
		var ok bool
		borrowedValue, ok = safeMultiplyInt64(position.BorrowedAmount, position.EntryPrice)
		if !ok {
			position.lastFeeAt += accrualMillis
			return position
		}
		assetBorrowed = position.BorrowedAmount
	}
	fee, ok := accruedMarginFee(borrowedValue, rate, accrualMillis)
	if !ok {
		position.lastFeeAt += accrualMillis
		return position
	}
	if fee > 0 {
		position.AccumulatedFees += fee
	}
	assetFee := int64(0)
	poolCashFee := fee
	if assetBorrowed > 0 {
		assetFee, ok = accruedMarginFee(assetBorrowed, rate, accrualMillis)
		if !ok {
			position.lastFeeAt += accrualMillis
			return position
		}
		poolCashFee = 0
	}
	if poolCashFee > 0 || assetFee > 0 {
		s.applyMarginInterestLocked(position, poolCashFee, assetFee)
	}
	position.lastFeeAt += accrualMillis
	return position
}

func (s *MarketStore) marginBorrowRateLocked(position MarginPosition) int64 {
	pool, ok := s.marginPoolForAssetLocked(position.AssetID)
	if !ok {
		return 0
	}
	rate := pool.CashRateBps
	if position.Side == engine.SideSell {
		rate = pool.AssetRateBps
	}
	return s.marginRateForUserLocked(position.UserID, rate)
}

func (s *MarketStore) unrealizedLoss(position MarginPosition, currentPrice int64) int64 {
	if currentPrice <= 0 || position.EntryPrice <= 0 || position.Quantity <= 0 {
		return 0
	}
	lossPerUnit := int64(0)
	if position.Side == engine.SideSell {
		if currentPrice > position.EntryPrice {
			lossPerUnit = currentPrice - position.EntryPrice
		}
	} else {
		if currentPrice < position.EntryPrice {
			lossPerUnit = position.EntryPrice - currentPrice
		}
	}
	if lossPerUnit <= 0 {
		return 0
	}
	loss, ok := safeMultiplyInt64(lossPerUnit, position.Quantity)
	if !ok {
		return 0
	}
	return loss
}

func (s *MarketStore) lossRatioBps(marginUsed, totalLoss int64) int64 {
	if marginUsed <= 0 || totalLoss <= 0 {
		return 0
	}
	numerator, ok := safeMultiplyInt64(totalLoss, bpsDenominator)
	if !ok {
		// Overflow implies the loss ratio exceeds 100%, so cap at the denominator.
		return bpsDenominator
	}
	return numerator / marginUsed
}

func (s *MarketStore) marginPoolForAssetLocked(assetID int64) (MarginPool, bool) {
	for _, pool := range s.marginPools {
		if pool.AssetID == assetID {
			return pool, true
		}
	}
	return MarginPool{}, false
}

func (s *MarketStore) applyMarginInterestLocked(position MarginPosition, cashFee, assetFee int64) {
	pool, ok := s.marginPoolForAssetLocked(position.AssetID)
	if !ok {
		return
	}
	if cashFee > 0 {
		pool.TotalCash += cashFee
	}
	if assetFee > 0 {
		pool.TotalAssets += assetFee
	}
	if cashFee > 0 || assetFee > 0 {
		pool.CashRateBps, pool.AssetRateBps = marginRates(pool)
		s.marginPools[pool.ID] = pool
	}
}

func accruedMarginFee(amount, rate, elapsed int64) (int64, bool) {
	if amount <= 0 || rate <= 0 || elapsed <= 0 {
		return 0, true
	}
	dailyFee, ok := safeMultiplyInt64(amount, rate)
	if !ok {
		return 0, false
	}
	dailyFee = dailyFee / bpsDenominator
	if dailyFee <= 0 {
		return 0, true
	}
	fee, ok := safeMultiplyInt64(dailyFee, elapsed)
	if !ok {
		return 0, false
	}
	fee = fee / millisecondsPerDay
	return fee, true
}

func (s *MarketStore) canBorrowMarginLocked(assetID int64, side engine.Side, amount int64) error {
	if amount <= 0 {
		return nil
	}
	pool, ok := s.marginPoolForAssetLocked(assetID)
	if !ok {
		return errors.New("margin pool not found")
	}
	if side == engine.SideBuy {
		if pool.TotalCash-pool.BorrowedCash < amount {
			return errors.New("insufficient margin cash")
		}
		return nil
	}
	if pool.TotalAssets-pool.BorrowedAssets < amount {
		return errors.New("insufficient margin assets")
	}
	return nil
}

func (s *MarketStore) applyMarginBorrowLocked(assetID int64, side engine.Side, amount int64) error {
	if amount <= 0 {
		return nil
	}
	pool, ok := s.marginPoolForAssetLocked(assetID)
	if !ok {
		return errors.New("margin pool not found")
	}
	if side == engine.SideBuy {
		pool.BorrowedCash += amount
	} else {
		pool.BorrowedAssets += amount
	}
	pool.CashRateBps, pool.AssetRateBps = marginRates(pool)
	s.marginPools[pool.ID] = pool
	return nil
}

func (s *MarketStore) repayMarginBorrowLocked(position MarginPosition) {
	if position.BorrowedAmount <= 0 {
		return
	}
	pool, ok := s.marginPoolForAssetLocked(position.AssetID)
	if !ok {
		return
	}
	if position.Side == engine.SideBuy {
		if pool.BorrowedCash >= position.BorrowedAmount {
			pool.BorrowedCash -= position.BorrowedAmount
		} else {
			pool.BorrowedCash = 0
		}
	} else {
		if pool.BorrowedAssets >= position.BorrowedAmount {
			pool.BorrowedAssets -= position.BorrowedAmount
		} else {
			pool.BorrowedAssets = 0
		}
	}
	pool.CashRateBps, pool.AssetRateBps = marginRates(pool)
	s.marginPools[pool.ID] = pool
}

func (s *MarketStore) openMarginPositionLocked(userID, assetID int64, side engine.Side, quantity, price, marginUsed, leverage, borrowed, now int64) MarginPosition {
	s.nextMarginPositionID++
	position := MarginPosition{
		ID:             s.nextMarginPositionID,
		UserID:         userID,
		AssetID:        assetID,
		Side:           side,
		Quantity:       quantity,
		EntryPrice:     price,
		CurrentPrice:   price,
		Leverage:       leverage,
		MarginUsed:     marginUsed,
		BorrowedAmount: borrowed,
		CreatedAt:      now,
		UpdatedAt:      now,
		lastFeeAt:      now,
	}
	s.marginPositions[position.ID] = position
	return position
}

func (s *MarketStore) checkMarginLiquidationsLocked(assetID int64) {
	if len(s.marginPositions) == 0 {
		return
	}
	now := time.Now().UTC().UnixMilli()
	for id, position := range s.marginPositions {
		if assetID != 0 && position.AssetID != assetID {
			continue
		}
		position = s.refreshMarginPositionLocked(position, now)
		if position.LossRatioBps >= marginLossCutBps {
			s.liquidateMarginPositionLocked(position, now)
			delete(s.marginPositions, id)
			continue
		}
		s.marginPositions[id] = position
	}
}

func (s *MarketStore) liquidateMarginPositionLocked(position MarginPosition, now int64) {
	totalLoss := position.UnrealizedLoss + position.AccumulatedFees
	remaining := position.MarginUsed - totalLoss
	if remaining < 0 {
		remaining = 0
	}
	fee := int64(0)
	payout := int64(0)
	if remaining > 0 {
		product, ok := safeMultiplyInt64(remaining, liquidationFeeBps)
		if ok {
			fee = product / bpsDenominator
		}
		if fee > remaining {
			fee = remaining
		}
		payout = remaining - fee
	}
	s.ensureUserLocked(position.UserID)
	if payout > 0 {
		s.balances[position.UserID][defaultCurrency] += payout
	}
	s.repayMarginBorrowLocked(position)
	s.nextLiquidationID++
	event := MarginLiquidation{
		ID:              s.nextLiquidationID,
		PositionID:      position.ID,
		UserID:          position.UserID,
		AssetID:         position.AssetID,
		Side:            position.Side,
		Quantity:        position.Quantity,
		LossRatioBps:    position.LossRatioBps,
		LiquidationFee:  fee,
		RemainingMargin: payout,
		OccurredAt:      now,
	}
	s.marginLiquidations = append(s.marginLiquidations, event)
}
