package api

import (
	"context"
	"log"
	"time"

	"github.com/h4ribote/Paper-Street/internal/db"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func (s *MarketStore) loadLiquidityStateFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	pools, err := s.queries.ListLiquidityPools(ctx)
	if err != nil {
		return err
	}
	for _, pool := range pools {
		s.pools[pool.PoolID] = LiquidityPool{
			ID:            pool.PoolID,
			BaseCurrency:  "ARC",
			QuoteCurrency: pool.QuoteCurrency,
			FeeBps:        pool.FeeBps,
			Liquidity:     pool.Liquidity,
			CurrentTick:   pool.CurrentTick,
		}
		s.currencies["ARC"] = struct{}{}
		s.currencies[pool.QuoteCurrency] = struct{}{}
	}
	positions, err := s.queries.ListLiquidityPositions(ctx)
	if err != nil {
		return err
	}
	for _, position := range positions {
		s.poolPositions[position.ID] = PoolPosition{
			ID:          position.ID,
			PoolID:      position.PoolID,
			UserID:      position.UserID,
			BaseAmount:  position.BaseAmount,
			QuoteAmount: position.QuoteAmount,
			LowerTick:   position.LowerTick,
			UpperTick:   position.UpperTick,
			CreatedAt:   position.CreatedAt,
		}
		if position.ID > s.nextPoolPosID {
			s.nextPoolPosID = position.ID
		}
	}
	return nil
}

func (s *MarketStore) loadMarginStateFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	pools, err := s.queries.ListMarginPools(ctx)
	if err != nil {
		return err
	}
	for _, pool := range pools {
		s.marginPools[pool.PoolID] = MarginPool{
			ID:               pool.PoolID,
			AssetID:          pool.AssetID,
			TotalCash:        pool.TotalCash,
			TotalAssets:      pool.TotalAssets,
			BorrowedCash:     pool.BorrowedCash,
			BorrowedAssets:   pool.BorrowedAssets,
			TotalCashShares:  pool.TotalCashShares,
			TotalAssetShares: pool.TotalAssetShares,
			CashRateBps:      pool.CashRateBps,
			AssetRateBps:     pool.AssetRateBps,
		}
	}
	providers, err := s.queries.ListMarginPoolProviders(ctx)
	if err != nil {
		return err
	}
	for _, provider := range providers {
		s.marginProviders[marginProviderKey{PoolID: provider.PoolID, UserID: provider.UserID}] = MarginProviderPosition{
			ID:          provider.ID,
			PoolID:      provider.PoolID,
			UserID:      provider.UserID,
			CashShares:  provider.CashShares,
			AssetShares: provider.AssetShares,
			CreatedAt:   provider.UpdatedAt,
		}
		if provider.ID > s.nextMarginPosID {
			s.nextMarginPosID = provider.ID
		}
	}
	positions, err := s.queries.ListMarginPositions(ctx)
	if err != nil {
		return err
	}
	for _, position := range positions {
		side := engine.SideBuy
		if position.Side == "SHORT" {
			side = engine.SideSell
		}
		marginPosition := MarginPosition{
			ID:           position.ID,
			UserID:       position.UserID,
			AssetID:      position.AssetID,
			Side:         side,
			Quantity:     position.Quantity,
			EntryPrice:   position.EntryPrice,
			CurrentPrice: position.CurrentPrice,
			Leverage:     position.Leverage,
			MarginUsed:   position.MarginUsed,
			CreatedAt:    position.CreatedAt,
			UpdatedAt:    position.UpdatedAt,
			lastFeeAt:    position.UpdatedAt,
		}
		if position.UnrealizedPL < 0 {
			marginPosition.UnrealizedLoss = -position.UnrealizedPL
		}
		s.marginPositions[position.ID] = marginPosition
		if position.ID > s.nextMarginPositionID {
			s.nextMarginPositionID = position.ID
		}
	}
	return nil
}

func (s *MarketStore) loadContractsFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	contracts, err := s.queries.ListContracts(ctx)
	if err != nil {
		return err
	}
	for _, record := range contracts {
		s.contracts[record.ID] = &Contract{
			ID:            record.ID,
			Title:         record.Title,
			AssetID:       record.AssetID,
			TotalRequired: record.TotalRequired,
			Delivered:     record.Delivered,
			PricePerUnit:  record.UnitPrice,
			MinRank:       record.MinRank,
			DeadlineAt:    record.ExpiresAt,
			XPPerUnit:     record.XPPerUnit,
		}
		if record.ID > s.nextContractID {
			s.nextContractID = record.ID
		}
	}
	deliveries, err := s.queries.ListContractDeliveries(ctx)
	if err != nil {
		return err
	}
	for _, delivery := range deliveries {
		progress := s.ensureContractProgressLocked(delivery.UserID)
		progress[delivery.ContractID] += delivery.Quantity
	}
	return nil
}

func (s *MarketStore) loadWorldFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	seasons, err := s.queries.ListSeasons(ctx)
	if err != nil {
		return err
	}
	if len(seasons) > 0 {
		s.seasons = make([]Season, 0, len(seasons))
		for _, season := range seasons {
			if !season.IsActive {
				continue
			}
			s.seasons = append(s.seasons, Season{
				Name:    season.Name,
				Theme:   season.Theme,
				StartAt: season.StartAt,
				EndAt:   season.EndAt,
			})
		}
	}
	regions, err := s.queries.ListRegions(ctx)
	if err != nil {
		return err
	}
	if len(regions) > 0 {
		s.regions = make([]Region, 0, len(regions))
		for _, region := range regions {
			s.regions = append(s.regions, Region{
				ID:          region.ID,
				Name:        region.Name,
				Description: region.Description,
			})
		}
	}
	events, err := s.queries.ListWorldEvents(ctx)
	if err != nil {
		return err
	}
	if len(events) > 0 {
		s.worldEvents = make([]WorldEvent, 0, len(events))
		for _, event := range events {
			s.worldEvents = append(s.worldEvents, WorldEvent{
				ID:          event.ID,
				Name:        event.Name,
				Description: event.Description,
				StartsAt:    event.StartsAt,
				EndsAt:      event.EndsAt,
			})
		}
	}
	return nil
}

func (s *MarketStore) loadMacroIndicatorsFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	records, err := s.queries.ListMacroIndicators(ctx)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	indicators := make([]MacroIndicator, 0, len(records))
	var maxQuarter int64
	var maxWeek int64
	for _, record := range records {
		indicators = append(indicators, MacroIndicator{
			Country:     record.Country,
			Type:        record.Type,
			Value:       record.Value,
			PublishedAt: record.PublishedAt,
		})
		t := time.UnixMilli(record.PublishedAt).UTC()
		q := macroPeriodIndex(t, macroQuarterPeriod)
		w := macroPeriodIndex(t, macroWeekPeriod)
		if q > maxQuarter {
			maxQuarter = q
		}
		if w > maxWeek {
			maxWeek = w
		}
	}
	s.macroIndicators = indicators
	s.macroQuarterIndex = maxQuarter
	s.macroWeekIndex = maxWeek
	s.refreshTheoreticalFXRatesLocked(time.Now().UTC())
	return nil
}

func (s *MarketStore) persistLiquidityPool(pool LiquidityPool) {
	if s.queries == nil || pool.ID == 0 {
		return
	}
	record := db.LiquidityPoolRecord{
		PoolID:        pool.ID,
		QuoteCurrency: pool.QuoteCurrency,
		FeeBps:        pool.FeeBps,
		CurrentTick:   pool.CurrentTick,
		Liquidity:     pool.Liquidity,
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertLiquidityPool(ctx, record, time.Now().UTC()); err != nil {
		log.Printf("db upsert liquidity pool %d: %v", pool.ID, err)
	}
}

func (s *MarketStore) persistPoolPosition(position PoolPosition) {
	if s.queries == nil || position.ID == 0 {
		return
	}
	record := db.LiquidityPositionRecord{
		ID:          position.ID,
		PoolID:      position.PoolID,
		UserID:      position.UserID,
		LowerTick:   position.LowerTick,
		UpperTick:   position.UpperTick,
		BaseAmount:  position.BaseAmount,
		QuoteAmount: position.QuoteAmount,
		CreatedAt:   position.CreatedAt,
		UpdatedAt:   time.Now().UTC().UnixMilli(),
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertLiquidityPosition(ctx, record, time.Now().UTC()); err != nil {
		log.Printf("db upsert liquidity position %d: %v", position.ID, err)
	}
}

func (s *MarketStore) deletePoolPosition(positionID int64) {
	if s.queries == nil || positionID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.DeleteLiquidityPosition(ctx, positionID); err != nil {
		log.Printf("db delete liquidity position %d: %v", positionID, err)
	}
}

func (s *MarketStore) persistMarginPool(pool MarginPool) {
	if s.queries == nil || pool.ID == 0 {
		return
	}
	record := db.MarginPoolRecord{
		PoolID:           pool.ID,
		AssetID:          pool.AssetID,
		Currency:         defaultCurrency,
		TotalCash:        pool.TotalCash,
		BorrowedCash:     pool.BorrowedCash,
		TotalAssets:      pool.TotalAssets,
		BorrowedAssets:   pool.BorrowedAssets,
		CashRateBps:      pool.CashRateBps,
		AssetRateBps:     pool.AssetRateBps,
		TotalCashShares:  pool.TotalCashShares,
		TotalAssetShares: pool.TotalAssetShares,
		UpdatedAt:        time.Now().UTC().UnixMilli(),
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertMarginPool(ctx, record, time.Now().UTC()); err != nil {
		log.Printf("db upsert margin pool %d: %v", pool.ID, err)
	}
}

func (s *MarketStore) persistMarginProvider(position MarginProviderPosition) {
	if s.queries == nil || position.PoolID == 0 || position.UserID == 0 {
		return
	}
	record := db.MarginPoolProviderRecord{
		ID:          position.ID,
		PoolID:      position.PoolID,
		UserID:      position.UserID,
		CashShares:  position.CashShares,
		AssetShares: position.AssetShares,
		UpdatedAt:   time.Now().UTC().UnixMilli(),
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertMarginPoolProvider(ctx, record, time.Now().UTC()); err != nil {
		log.Printf("db upsert margin provider %d/%d: %v", position.PoolID, position.UserID, err)
	}
}

func (s *MarketStore) persistMarginPosition(position MarginPosition) {
	if s.queries == nil || position.ID == 0 {
		return
	}
	side := "LONG"
	if position.Side == engine.SideSell {
		side = "SHORT"
	}
	record := db.MarginPositionRecord{
		ID:           position.ID,
		UserID:       position.UserID,
		AssetID:      position.AssetID,
		Side:         side,
		Quantity:     position.Quantity,
		EntryPrice:   position.EntryPrice,
		CurrentPrice: position.CurrentPrice,
		Leverage:     position.Leverage,
		MarginUsed:   position.MarginUsed,
		UnrealizedPL: -position.UnrealizedLoss,
		CreatedAt:    position.CreatedAt,
		UpdatedAt:    position.UpdatedAt,
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertMarginPosition(ctx, record); err != nil {
		log.Printf("db upsert margin position %d: %v", position.ID, err)
	}
}

func (s *MarketStore) deleteMarginPosition(positionID int64) {
	if s.queries == nil || positionID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.DeleteMarginPosition(ctx, positionID); err != nil {
		log.Printf("db delete margin position %d: %v", positionID, err)
	}
}

func (s *MarketStore) persistContract(contract *Contract) {
	if s.queries == nil || contract == nil || contract.ID == 0 {
		return
	}
	status := "ACTIVE"
	nowMillis := time.Now().UTC().UnixMilli()
	if contract.Delivered >= contract.TotalRequired {
		status = "COMPLETED"
	} else if contract.DeadlineAt > 0 && nowMillis > contract.DeadlineAt {
		status = "EXPIRED"
	}
	record := db.ContractRecord{
		ID:            contract.ID,
		Title:         contract.Title,
		AssetID:       contract.AssetID,
		TotalRequired: contract.TotalRequired,
		Delivered:     contract.Delivered,
		UnitPrice:     contract.PricePerUnit,
		XPPerUnit:     contract.XPPerUnit,
		MinRank:       contract.MinRank,
		Status:        status,
		StartAt:       nowMillis,
		ExpiresAt:     contract.DeadlineAt,
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertContract(ctx, record, time.Now().UTC()); err != nil {
		log.Printf("db upsert contract %d: %v", contract.ID, err)
	}
}

func (s *MarketStore) deleteContract(contractID int64) {
	if s.queries == nil || contractID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.DeleteContract(ctx, contractID); err != nil {
		log.Printf("db delete contract %d: %v", contractID, err)
	}
}

func (s *MarketStore) persistContractDelivery(contractID, userID, quantity, payoutAmount, xpGained, deliveredAt int64) {
	if s.queries == nil || contractID == 0 || userID == 0 || quantity <= 0 {
		return
	}
	record := db.ContractDeliveryRecord{
		ContractID:   contractID,
		UserID:       userID,
		Quantity:     quantity,
		PayoutAmount: payoutAmount,
		XPGained:     xpGained,
		DeliveredAt:  deliveredAt,
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.InsertContractDelivery(ctx, record, time.Now().UTC()); err != nil {
		log.Printf("db insert contract delivery %d/%d: %v", contractID, userID, err)
	}
}

func (s *MarketStore) persistWorldState() {
	if s.queries == nil {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	for i, season := range s.seasons {
		record := db.SeasonRecord{
			ID:       int64(i + 1),
			Name:     season.Name,
			Theme:    season.Theme,
			StartAt:  season.StartAt,
			EndAt:    season.EndAt,
			IsActive: true,
		}
		if err := s.queries.UpsertSeason(ctx, record); err != nil {
			log.Printf("db upsert season %d: %v", record.ID, err)
		}
	}
	for _, region := range s.regions {
		record := db.RegionRecord{
			ID:          region.ID,
			Name:        region.Name,
			Description: region.Description,
		}
		if err := s.queries.UpsertRegion(ctx, record); err != nil {
			log.Printf("db upsert region %d: %v", record.ID, err)
		}
	}
	for _, event := range s.worldEvents {
		record := db.WorldEventRecord{
			ID:          event.ID,
			Name:        event.Name,
			Description: event.Description,
			StartsAt:    event.StartsAt,
			EndsAt:      event.EndsAt,
		}
		if err := s.queries.UpsertWorldEvent(ctx, record); err != nil {
			log.Printf("db upsert world event %d: %v", record.ID, err)
		}
	}
}

func (s *MarketStore) persistMacroIndicatorsLocked() {
	if s.queries == nil {
		return
	}
	records := make([]db.MacroIndicatorRecord, 0, len(s.macroIndicators))
	for _, indicator := range s.macroIndicators {
		records = append(records, db.MacroIndicatorRecord{
			Country:     indicator.Country,
			Type:        indicator.Type,
			Value:       indicator.Value,
			PublishedAt: indicator.PublishedAt,
		})
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.ReplaceMacroIndicators(ctx, records); err != nil {
		log.Printf("db replace macro indicators: %v", err)
	}
}
