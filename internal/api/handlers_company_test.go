package api

import (
	"net/http/httptest"
	"testing"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
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
