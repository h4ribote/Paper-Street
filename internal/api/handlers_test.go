package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	testAPIKeyUser1 = "00010203040506070809"
	testAPIKeyUser2 = "11111111111111111111"
)

func TestTradeFlowUpdatesMarketData(t *testing.T) {
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

	eng := engine.NewEngine(store)
	handler := NewRouter(eng, apiKeys, store)
	server := httptest.NewServer(handler)
	defer server.Close()

	submitOrder(t, server.URL, testAPIKeyUser2, orderRequest{
		AssetID:  101,
		UserID:   2,
		Side:     "SELL",
		Type:     "LIMIT",
		Quantity: 10,
		Price:    100,
	})
	submitOrder(t, server.URL, testAPIKeyUser1, orderRequest{
		AssetID:  101,
		UserID:   1,
		Side:     "BUY",
		Type:     "LIMIT",
		Quantity: 10,
		Price:    100,
	})

	var trades []engine.Execution
	getJSON(t, server.URL+"/market/trades/101?limit=1", testAPIKeyUser1, &trades)
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Price != 100 || trades[0].Quantity != 10 {
		t.Fatalf("unexpected trade %+v", trades[0])
	}

	var balances []models.Balance
	getJSON(t, server.URL+"/portfolio/balances?user_id=1", testAPIKeyUser1, &balances)
	usd := balanceAmount(balances, defaultCurrency)
	expectedCash := defaultCashBalance - 100*10
	if usd != expectedCash {
		t.Fatalf("expected cash %d, got %d", expectedCash, usd)
	}

	var assets []PortfolioAsset
	getJSON(t, server.URL+"/portfolio/assets?user_id=1", testAPIKeyUser1, &assets)
	if len(assets) == 0 || assets[0].Quantity != 10 {
		t.Fatalf("expected asset quantity 10, got %+v", assets)
	}
}

func TestHandleOrdersPagination(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)

	eng := engine.NewEngine(store)
	handler := NewRouter(eng, apiKeys, store)
	server := httptest.NewServer(handler)
	defer server.Close()

	for i := 0; i < 3; i++ {
		submitEngineOrder(t, eng, &engine.Order{
			AssetID:  101,
			UserID:   1,
			Side:     engine.SideBuy,
			Type:     engine.OrderTypeLimit,
			Quantity: 1,
			Price:    int64(100 + i),
		})
	}

	var all []engine.Order
	getJSON(t, server.URL+"/orders", testAPIKeyUser1, &all)
	if len(all) < 3 {
		t.Fatalf("expected at least 3 orders, got %d", len(all))
	}

	var page []engine.Order
	getJSON(t, server.URL+"/orders?limit=1&offset=1", testAPIKeyUser1, &page)
	if len(page) != 1 {
		t.Fatalf("expected 1 paged order, got %d", len(page))
	}
	if page[0].ID != all[1].ID {
		t.Fatalf("expected order id %d at offset 1, got %d", all[1].ID, page[0].ID)
	}
}

func TestHandleOrderByIDRequiresAssetID(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)

	eng := engine.NewEngine(store)
	handler := NewRouter(eng, apiKeys, store)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	created, err := eng.SubmitOrder(ctx, &engine.Order{
		AssetID:  101,
		UserID:   1,
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 1,
		Price:    100,
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/orders/%d", server.URL, created.Order.ID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set(apiKeyHeader, testAPIKeyUser1)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	var fetched engine.Order
	getJSON(t, fmt.Sprintf("%s/orders/%d?asset_id=101", server.URL, created.Order.ID), testAPIKeyUser1, &fetched)
	if fetched.ID != created.Order.ID || fetched.AssetID != 101 {
		t.Fatalf("unexpected order returned %+v", fetched)
	}
}

func TestHandleOrderBookDepthLimit(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	store.EnsureUser(2)

	eng := engine.NewEngine(store)
	handler := NewRouter(eng, apiKeys, store)
	server := httptest.NewServer(handler)
	defer server.Close()

	levelCount := maxOrderBookDepth + 5
	for i := 0; i < levelCount; i++ {
		submitEngineOrder(t, eng, &engine.Order{
			AssetID:  101,
			UserID:   1,
			Side:     engine.SideBuy,
			Type:     engine.OrderTypeLimit,
			Quantity: 1,
			Price:    int64(1000 + i),
		})
		submitEngineOrder(t, eng, &engine.Order{
			AssetID:  101,
			UserID:   2,
			Side:     engine.SideSell,
			Type:     engine.OrderTypeLimit,
			Quantity: 1,
			Price:    int64(2000 + i),
		})
	}

	var snapshot engine.OrderBookSnapshot
	getJSON(t, server.URL+"/market/orderbook/101?depth=9999", testAPIKeyUser1, &snapshot)
	if len(snapshot.Bids) != maxOrderBookDepth {
		t.Fatalf("expected %d bid levels, got %d", maxOrderBookDepth, len(snapshot.Bids))
	}
	if len(snapshot.Asks) != maxOrderBookDepth {
		t.Fatalf("expected %d ask levels, got %d", maxOrderBookDepth, len(snapshot.Asks))
	}
}

func submitOrder(t *testing.T, baseURL, apiKey string, order orderRequest) {
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
	req.Header.Set(apiKeyHeader, apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to submit order: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected order status: %d", resp.StatusCode)
	}
}

func submitEngineOrder(t *testing.T, eng *engine.Engine, order *engine.Order) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := eng.SubmitOrder(ctx, order); err != nil {
		t.Fatalf("failed to submit order: %v", err)
	}
}

func getJSON(t *testing.T, url, apiKey string, target interface{}) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set(apiKeyHeader, apiKey)
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
