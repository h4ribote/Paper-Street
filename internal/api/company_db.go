package api

import (
	"context"
	"database/sql"
	"log"

	"time"

	"github.com/h4ribote/Paper-Street/internal/db"
	"github.com/h4ribote/Paper-Street/internal/models"
)

func (s *MarketStore) loadCompaniesFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	records, err := s.queries.ListCompanies(ctx)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	commodityID := s.firstCommodityAssetID()
	for _, record := range records {
		asset, ok := s.Asset(record.CompanyID)
		if !ok {
			asset = models.Asset{ID: record.CompanyID, Symbol: record.Symbol, Name: record.Name, Type: "STOCK", Sector: record.Sector}
			ctx, cancel := s.dbContext()
			_ = s.queries.UpsertAsset(ctx, asset, defaultAssetPrice)
			cancel()
		}
		state := &companyState{
			Company: Company{
				ID:     record.CompanyID,
				Name:   record.Name,
				Symbol: record.Symbol,
				Sector: record.Sector,
			},
			UserID:                record.UserID.Int64,
			Country:               record.Country,
			OutputAssetID:         commodityID,
			MaxProductionCapacity: record.MaxProductionCapacity,
			CurrentInventory:      record.CurrentInventory,
			LastCapexAt:           record.LastCapexAt,
			SharesIssued:          record.SharesIssued,
			SharesOutstanding:     record.SharesOutstanding,
			TreasuryShares:        record.TreasuryStock,
		}
		if state.UserID == 0 {
			state.UserID = asset.ID
		}
		if state.MaxProductionCapacity == 0 {
			state.MaxProductionCapacity = defaultMaxCapacity
		}
		if state.SharesIssued == 0 {
			state.SharesIssued = defaultSharesIssued
		}
		if state.SharesOutstanding == 0 && state.TreasuryShares == 0 {
			state.TreasuryShares = state.SharesIssued * defaultTreasuryBps / bpsDenominator
			state.SharesOutstanding = state.SharesIssued - state.TreasuryShares
		}
		s.companyStates[state.Company.ID] = state
		user := s.ensureUserLocked(state.UserID)
		if user.Role != "bot" {
			user.Role = "bot"
			ctx, cancel := s.dbContext()
			if s.queries != nil {
				_ = s.queries.UpsertUser(ctx, user, time.Now().UTC())
			}
			cancel()
		}
	}
	return nil
}

func (s *MarketStore) loadProductionRecipesFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	recipes, err := s.queries.ListProductionRecipes(ctx)
	if err != nil {
		return err
	}
	inputs, err := s.queries.ListProductionInputs(ctx)
	if err != nil {
		return err
	}
	inputMap := make(map[int64][]ProductionInput)
	for _, input := range inputs {
		inputMap[input.RecipeID] = append(inputMap[input.RecipeID], ProductionInput{
			AssetID:  input.InputAssetID,
			Quantity: input.InputQuantity,
		})
	}
	for _, recipe := range recipes {
		s.companyRecipes[recipe.CompanyID] = append(s.companyRecipes[recipe.CompanyID], ProductionRecipe{
			ID:             recipe.ID,
			CompanyID:      recipe.CompanyID,
			OutputAssetID:  recipe.OutputAssetID,
			OutputQuantity: recipe.OutputQuantity,
			Inputs:         inputMap[recipe.ID],
		})
	}
	return nil
}

func (s *MarketStore) loadFinancialReportsFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	reports, err := s.queries.ListFinancialReports(ctx)
	if err != nil {
		return err
	}
	for _, report := range reports {
		s.storeFinancialReportLocked(report.CompanyID, CompanyFinancialReport{
			CompanyID:       report.CompanyID,
			FiscalYear:      report.FiscalYear,
			FiscalQuarter:   report.FiscalQuarter,
			Revenue:         report.Revenue,
			NetIncome:       report.NetIncome,
			EPS:             report.EPS,
			Capex:           report.Capex,
			UtilizationRate: report.UtilizationRate,
			InventoryLevel:  report.InventoryLevel,
			Guidance:        report.Guidance,
			PublishedAt:     report.PublishedAt,
		})
	}
	return nil
}

func (s *MarketStore) loadCompanyDividendsFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	records, err := s.queries.ListCompanyDividends(ctx)
	if err != nil {
		return err
	}
	for _, record := range records {
		s.companyDividends[record.CompanyID] = append(s.companyDividends[record.CompanyID], CompanyDividendRecord{
			CompanyID:          record.CompanyID,
			AssetID:            record.AssetID,
			FiscalYear:         record.FiscalYear,
			FiscalQuarter:      record.FiscalQuarter,
			NetIncome:          record.NetIncome,
			PayoutRatioBps:     record.PayoutRatioBps,
			DividendPerShare:   record.DividendPerShare,
			CompanyPayout:      record.CompanyPayout,
			PoolPayout:         record.PoolPayout,
			SpotPayout:         record.SpotPayout,
			MarginLongPayout:   record.MarginLongPayout,
			MarginShortCharge:  record.MarginShortCharge,
			EligibleSpotShares: record.EligibleSpotShares,
			EligibleLongShares: record.EligibleLongShares,
			PoolShares:         record.PoolShares,
			CreatedAt:          record.CreatedAt,
		})
	}
	return nil
}

func (s *MarketStore) persistCompanyState(state *companyState) {
	if s.queries == nil || state == nil {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	record := db.CompanyRecord{
		CompanyID:             state.Company.ID,
		Name:                  state.Company.Name,
		Symbol:                state.Company.Symbol,
		UserID:                sql.NullInt64{Int64: state.UserID, Valid: state.UserID != 0},
		MaxProductionCapacity: state.MaxProductionCapacity,
		CurrentInventory:      state.CurrentInventory,
		LastCapexAt:           state.LastCapexAt,
		SharesIssued:          state.SharesIssued,
		SharesOutstanding:     state.SharesOutstanding,
		TreasuryStock:         state.TreasuryShares,
	}
	if err := s.queries.UpsertCompany(ctx, record); err != nil {
		log.Printf("db upsert company %d: %v", state.Company.ID, err)
	}
}

func (s *MarketStore) persistFinancialReport(report CompanyFinancialReport) {
	if s.queries == nil || report.CompanyID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	record := db.FinancialReportRecord{
		CompanyID:       report.CompanyID,
		FiscalYear:      report.FiscalYear,
		FiscalQuarter:   report.FiscalQuarter,
		Revenue:         report.Revenue,
		NetIncome:       report.NetIncome,
		EPS:             report.EPS,
		Capex:           report.Capex,
		UtilizationRate: report.UtilizationRate,
		InventoryLevel:  report.InventoryLevel,
		Guidance:        report.Guidance,
		PublishedAt:     report.PublishedAt,
	}
	if err := s.queries.UpsertFinancialReport(ctx, record); err != nil {
		log.Printf("db upsert financial report %d: %v", report.CompanyID, err)
	}
}

func (s *MarketStore) persistCompanyDividend(record CompanyDividendRecord) {
	if s.queries == nil || record.CompanyID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	dividend := db.CompanyDividendRecord{
		CompanyID:          record.CompanyID,
		AssetID:            record.AssetID,
		FiscalYear:         record.FiscalYear,
		FiscalQuarter:      record.FiscalQuarter,
		NetIncome:          record.NetIncome,
		PayoutRatioBps:     record.PayoutRatioBps,
		DividendPerShare:   record.DividendPerShare,
		CompanyPayout:      record.CompanyPayout,
		PoolPayout:         record.PoolPayout,
		SpotPayout:         record.SpotPayout,
		MarginLongPayout:   record.MarginLongPayout,
		MarginShortCharge:  record.MarginShortCharge,
		EligibleSpotShares: record.EligibleSpotShares,
		EligibleLongShares: record.EligibleLongShares,
		PoolShares:         record.PoolShares,
		CreatedAt:          record.CreatedAt,
	}
	if err := s.queries.UpsertCompanyDividend(ctx, dividend); err != nil {
		log.Printf("db upsert company dividend %d: %v", record.CompanyID, err)
	}
}
