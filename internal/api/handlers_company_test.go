package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

func TestCompanyCapitalStructureEndpoint(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	var structure CompanyCapitalStructure
	getJSON(t, server.URL+"/companies/101/capital-structure", testAPIKeyUser1, &structure)
	store.mu.RLock()
	state := store.companyStates[101]
	expectedIssued := int64(0)
	expectedOutstanding := int64(0)
	expectedTreasury := int64(0)
	if state != nil {
		expectedIssued = state.SharesIssued
		expectedOutstanding = state.SharesOutstanding
		expectedTreasury = state.TreasuryShares
	}
	store.mu.RUnlock()
	if structure.SharesIssued != expectedIssued {
		t.Fatalf("expected shares issued %d, got %d", expectedIssued, structure.SharesIssued)
	}
	if structure.SharesOutstanding != expectedOutstanding {
		t.Fatalf("expected shares outstanding %d, got %d", expectedOutstanding, structure.SharesOutstanding)
	}
	if structure.TreasuryShares != expectedTreasury {
		t.Fatalf("expected treasury shares %d, got %d", expectedTreasury, structure.TreasuryShares)
	}
}

func TestCompanyFinancingAndBuybackEndpoints(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	if err := apiKeys.AddHex(testAPIKeyUser2); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.RegisterAPIKey(testAPIKeyUser2, 2)
	store.EnsureUser(1)
	store.EnsureUser(2)
	store.mu.Lock()
	store.balances[1][defaultCurrency] = 100_000
	store.mu.Unlock()
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	var finance CompanyFinancingResult
	postJSON(t, server.URL+"/companies/101/financing/initiate", testAPIKeyUser1, CompanyFinancingRequest{TargetAmount: 1000000}, &finance)
	if finance.SharesOutstanding <= 0 {
		t.Fatalf("expected financing to increase outstanding shares")
	}

	submitOrder(t, server.URL, testAPIKeyUser2, orderRequest{
		AssetID:  101,
		UserID:   2,
		Side:     "SELL",
		Type:     "LIMIT",
		Quantity: 200,
		Price:    100,
	})
	submitOrder(t, server.URL, testAPIKeyUser1, orderRequest{
		AssetID:  101,
		UserID:   1,
		Side:     "BUY",
		Type:     "LIMIT",
		Quantity: 200,
		Price:    100,
	})

	var buyback CompanyBuybackResult
	postJSON(t, server.URL+"/companies/101/buyback/authorize", testAPIKeyUser1, CompanyBuybackRequest{Budget: 100000}, &buyback)
	if buyback.SharesRepurchased <= 0 {
		t.Fatalf("expected buyback to repurchase shares, got %d", buyback.SharesRepurchased)
	}
}

func TestCompanySimulationEndpoint(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	var result CompanySimulationResult
	postJSON(t, server.URL+"/companies/101/simulate", testAPIKeyUser1, companySimulationRequest{Quarters: 1}, &result)
	if result.CompanyID != 101 {
		t.Fatalf("expected company id 101, got %d", result.CompanyID)
	}
	if result.Production <= 0 {
		t.Fatalf("expected production to be positive, got %d", result.Production)
	}
	if result.Report.CompanyID != 101 {
		t.Fatalf("expected report for company 101, got %d", result.Report.CompanyID)
	}
}

func TestCompanyProcurementRespectsCashBalance(t *testing.T) {
	store := NewMarketStore()
	inputAsset := models.Asset{ID: 104, Symbol: "INP", Name: "Input", Type: "COMMODITY", Sector: "METAL"}
	store.mu.Lock()
	state := store.companyStates[101]
	if state == nil {
		store.mu.Unlock()
		t.Fatal("expected company state for 101")
	}
	inputPrice := int64(10)
	inputQuantity := int64(2)
	store.assets[inputAsset.ID] = inputAsset
	store.basePrices[inputAsset.ID] = inputPrice
	state.MaxProductionCapacity = 10
	store.companyRecipes[state.Company.ID] = []ProductionRecipe{
		{
			ID:             1,
			CompanyID:      state.Company.ID,
			OutputAssetID:  state.OutputAssetID,
			OutputQuantity: 1,
			Inputs: []ProductionInput{
				{AssetID: inputAsset.ID, Quantity: inputQuantity},
			},
		},
	}
	store.positions[state.UserID][inputAsset.ID] = 0
	cashBalance := int64(50)
	store.balances[state.UserID][defaultCurrency] = cashBalance
	availableInputs := store.positions[state.UserID][inputAsset.ID]
	store.mu.Unlock()

	result, err := store.SimulateCompanyQuarter(101, time.Now().UTC())
	if err != nil {
		t.Fatalf("simulate company quarter: %v", err)
	}
	maxAffordableInputs := cashBalance / inputPrice
	expectedProduction := (availableInputs + maxAffordableInputs) / inputQuantity
	if result.Production != expectedProduction {
		t.Fatalf("expected production to be %d with limited cash, got %d", expectedProduction, result.Production)
	}
	store.mu.RLock()
	inputBalance := store.positions[state.UserID][inputAsset.ID]
	store.mu.RUnlock()
	if inputBalance != 0 {
		t.Fatalf("expected input inventory to be fully consumed, got %d", inputBalance)
	}
}

func TestCompanyDividendDistributionCoversSpotPoolAndMargin(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)
	store.mu.Lock()
	state := store.companyStates[101]
	if state == nil {
		store.mu.Unlock()
		t.Fatal("expected company state for 101")
	}
	state.MaxProductionCapacity = 1
	state.OutputAssetID = 0
	state.CurrentInventory = 0
	store.companyRecipes[state.Company.ID] = []ProductionRecipe{{ID: 1, CompanyID: state.Company.ID, OutputAssetID: 0, OutputQuantity: 1}}
	store.balances[state.UserID][defaultCurrency] = 10_000_000
	spotUser := int64(3001)
	poolUser := int64(3002)
	store.ensureUserLocked(spotUser)
	store.ensureUserLocked(poolUser)
	store.positions[spotUser][101] = 100
	if _, ok := store.assetAcquiredAt[spotUser]; !ok {
		store.assetAcquiredAt[spotUser] = make(map[int64]int64)
	}
	store.assetAcquiredAt[spotUser][101] = now.Add(-45 * 24 * time.Hour).UnixMilli()
	store.positions[poolUser][101] = 1_000
	poolID := int64(0)
	for id, pool := range store.marginPools {
		if pool.AssetID == 101 {
			poolID = id
			pool.TotalAssets = 200
			pool.BorrowedAssets = 40
			pool.TotalCash = 100_000
			pool.TotalAssetShares = 200
			pool.TotalCashShares = 100_000
			pool.CashRateBps, pool.AssetRateBps = marginRates(pool)
			store.marginPools[id] = pool
			break
		}
	}
	if poolID == 0 {
		store.mu.Unlock()
		t.Fatal("expected margin pool for asset 101")
	}
	key := marginProviderKey{PoolID: poolID, UserID: poolUser}
	store.marginProviders[key] = MarginProviderPosition{
		ID:          9001,
		PoolID:      poolID,
		UserID:      poolUser,
		CashShares:  100_000,
		AssetShares: 200,
		CreatedAt:   now.Add(-50 * 24 * time.Hour).UnixMilli(),
	}
	store.marginPositions[5001] = MarginPosition{
		ID:             5001,
		UserID:         4001,
		AssetID:        101,
		Side:           engine.SideBuy,
		Quantity:       50,
		EntryPrice:     10_000,
		CurrentPrice:   10_000,
		Leverage:       2,
		MarginUsed:     5_000,
		BorrowedAmount: 5_000,
		CreatedAt:      now.Add(-20 * 24 * time.Hour).UnixMilli(),
		UpdatedAt:      now.Add(-20 * 24 * time.Hour).UnixMilli(),
	}
	store.marginPositions[5002] = MarginPosition{
		ID:             5002,
		UserID:         4002,
		AssetID:        101,
		Side:           engine.SideSell,
		Quantity:       30,
		EntryPrice:     10_000,
		CurrentPrice:   10_000,
		Leverage:       2,
		MarginUsed:     5_000,
		BorrowedAmount: 30,
		CreatedAt:      now.Add(-20 * 24 * time.Hour).UnixMilli(),
		UpdatedAt:      now.Add(-20 * 24 * time.Hour).UnixMilli(),
	}
	report := CompanyFinancialReport{
		CompanyID:     101,
		FiscalYear:    2026,
		FiscalQuarter: 1,
		NetIncome:     2_000_000_000,
		EPS:           10,
		PublishedAt:   now.UnixMilli(),
	}
	store.storeFinancialReportLocked(101, report)
	companyStartCash := store.balances[state.UserID][defaultCurrency]
	spotStartCash := store.balances[spotUser][defaultCurrency]
	longStartMargin := store.marginPositions[5001].MarginUsed
	shortStartFees := store.marginPositions[5002].AccumulatedFees
	poolStartCash := store.marginPools[poolID].TotalCash
	store.applyCompanyDividendLocked(state, report, 2_000_000_000, now)
	store.mu.Unlock()

	store.mu.RLock()
	records := store.companyDividends[101]
	if len(records) == 0 {
		store.mu.RUnlock()
		t.Fatalf("expected dividend record for company 101")
	}
	record := records[len(records)-1]
	if record.CompanyPayout <= 0 {
		store.mu.RUnlock()
		t.Fatalf("expected positive company payout, got %d", record.CompanyPayout)
	}
	if got := store.balances[101][defaultCurrency]; got >= companyStartCash {
		store.mu.RUnlock()
		t.Fatalf("expected company cash to decrease after dividends, start=%d got=%d", companyStartCash, got)
	}
	if got := store.balances[spotUser][defaultCurrency]; got <= spotStartCash {
		store.mu.RUnlock()
		t.Fatalf("expected spot holder to receive dividend, start=%d got=%d", spotStartCash, got)
	}
	if got := store.marginPositions[5001].MarginUsed; got <= longStartMargin {
		store.mu.RUnlock()
		t.Fatalf("expected margin long to receive dividend credit, start=%d got=%d", longStartMargin, got)
	}
	if got := store.marginPositions[5002].AccumulatedFees; got <= shortStartFees {
		store.mu.RUnlock()
		t.Fatalf("expected margin short to be charged dividend, start=%d got=%d", shortStartFees, got)
	}
	if got := store.marginPools[poolID].TotalCash; got <= poolStartCash {
		store.mu.RUnlock()
		t.Fatalf("expected margin pool cash to increase, start=%d got=%d", poolStartCash, got)
	}
	if record.PoolPayout <= 0 {
		store.mu.RUnlock()
		t.Fatalf("expected positive pool payout, got %d", record.PoolPayout)
	}
	if record.MarginLongPayout <= 0 {
		store.mu.RUnlock()
		t.Fatalf("expected positive margin long payout, got %d", record.MarginLongPayout)
	}
	if record.MarginShortCharge <= 0 {
		store.mu.RUnlock()
		t.Fatalf("expected positive margin short charge, got %d", record.MarginShortCharge)
	}
	store.mu.RUnlock()
}

func TestCompanyDividendsEndpoint(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)
	store.mu.Lock()
	state := store.companyStates[101]
	if state == nil {
		store.mu.Unlock()
		t.Fatal("expected company state for 101")
	}
	state.MaxProductionCapacity = 1
	state.OutputAssetID = 0
	store.companyRecipes[state.Company.ID] = []ProductionRecipe{{ID: 1, CompanyID: state.Company.ID, OutputAssetID: 0, OutputQuantity: 1}}
	holder := int64(3101)
	store.ensureUserLocked(holder)
	store.positions[holder][101] = 10
	if _, ok := store.assetAcquiredAt[holder]; !ok {
		store.assetAcquiredAt[holder] = make(map[int64]int64)
	}
	store.assetAcquiredAt[holder][101] = now.Add(-10 * 24 * time.Hour).UnixMilli()
	store.balances[state.UserID][defaultCurrency] = 2_000_000_000
	report := CompanyFinancialReport{
		CompanyID:     101,
		FiscalYear:    2026,
		FiscalQuarter: 1,
		NetIncome:     1_000_000_000,
		EPS:           5,
		PublishedAt:   now.UnixMilli(),
	}
	if state.SharesOutstanding <= 0 {
		store.mu.Unlock()
		t.Fatalf("expected positive shares outstanding")
	}
	if payout := store.companyPayoutRatioBpsLocked(state, report, 1_000_000_000); payout <= 0 {
		store.mu.Unlock()
		t.Fatalf("expected positive payout ratio, got %d", payout)
	}
	store.storeFinancialReportLocked(101, report)
	store.applyCompanyDividendLocked(state, report, 1_000_000_000, now)
	if len(store.companyDividends[101]) == 0 {
		store.mu.Unlock()
		t.Fatalf("expected in-store dividend records before endpoint call")
	}
	store.mu.Unlock()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	var records []CompanyDividendRecord
	getJSON(t, server.URL+"/companies/101/dividends", testAPIKeyUser1, &records)
	if len(records) == 0 {
		t.Fatalf("expected dividend records from endpoint")
	}
	if records[0].CompanyID != 101 {
		t.Fatalf("expected company_id 101, got %d", records[0].CompanyID)
	}
}
