package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

const testAPIKey = "00010203040506070809"

func TestTradeFlowUpdatesMarketData(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKey); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKey, 1)
	store.EnsureUser(1)
	store.EnsureUser(2)

	eng := engine.NewEngine(store)
	handler := NewRouter(eng, apiKeys, store)
	server := httptest.NewServer(handler)
	defer server.Close()

	submitOrder(t, server.URL, orderRequest{
		AssetID:  101,
		UserID:   2,
		Side:     "SELL",
		Type:     "LIMIT",
		Quantity: 10,
		Price:    100,
	})
	submitOrder(t, server.URL, orderRequest{
		AssetID:  101,
		UserID:   1,
		Side:     "BUY",
		Type:     "LIMIT",
		Quantity: 10,
		Price:    100,
	})

	var trades []engine.Execution
	getJSON(t, server.URL+"/market/trades/101?limit=1", &trades)
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Price != 100 || trades[0].Quantity != 10 {
		t.Fatalf("unexpected trade %+v", trades[0])
	}

	var balances []models.Balance
	getJSON(t, server.URL+"/portfolio/balances?user_id=1", &balances)
	usd := balanceAmount(balances, defaultCurrency)
	expectedCash := defaultCashBalance - 100*10
	if usd != expectedCash {
		t.Fatalf("expected cash %d, got %d", expectedCash, usd)
	}

	var assets []PortfolioAsset
	getJSON(t, server.URL+"/portfolio/assets?user_id=1", &assets)
	if len(assets) == 0 || assets[0].Quantity != 10 {
		t.Fatalf("expected asset quantity 10, got %+v", assets)
	}
}

func submitOrder(t *testing.T, baseURL string, order orderRequest) {
	t.Helper()
	payload, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("failed to marshal order: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/orders", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apiKeyHeader, testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to submit order: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected order status: %d", resp.StatusCode)
	}
}

func getJSON(t *testing.T, url string, target interface{}) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set(apiKeyHeader, testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func balanceAmount(balances []models.Balance, currency string) int64 {
	for _, balance := range balances {
		if balance.Currency == currency {
			return balance.Amount
		}
	}
	return 0
}
