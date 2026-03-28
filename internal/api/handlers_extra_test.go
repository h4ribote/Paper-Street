package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func TestPoolPositionLifecycle(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store))
	defer server.Close()

	request := poolPositionRequest{
		UserID:      1,
		BaseAmount:  1000,
		QuoteAmount: 500,
		LowerTick:   -10,
		UpperTick:   10,
	}
	var position PoolPosition
	postJSON(t, server.URL+"/pools/1/positions", testAPIKeyUser1, request, &position)
	if position.ID == 0 || position.PoolID != 1 {
		t.Fatalf("unexpected position response: %+v", position)
	}

	var positions []PoolPosition
	getJSON(t, server.URL+"/pools/positions?user_id=1", testAPIKeyUser1, &positions)
	if len(positions) == 0 || positions[0].ID != position.ID {
		t.Fatalf("expected position in list, got %+v", positions)
	}

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/pools/positions/"+strconv.FormatInt(position.ID, 10), nil)
	if err != nil {
		t.Fatalf("failed to build delete request: %v", err)
	}
	req.Header.Set(apiKeyHeader, testAPIKeyUser1)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to delete position: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", resp.StatusCode)
	}
}

func TestIndexCreateRedeem(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store))
	defer server.Close()

	var createResult IndexActionResult
	postJSON(t, server.URL+"/indices/201/create", testAPIKeyUser1, indexActionRequest{UserID: 1, Quantity: 2}, &createResult)
	if createResult.AssetID != 201 || createResult.Quantity != 2 {
		t.Fatalf("unexpected create response: %+v", createResult)
	}

	var assets []PortfolioAsset
	getJSON(t, server.URL+"/portfolio/assets?user_id=1", testAPIKeyUser1, &assets)
	found := false
	for _, asset := range assets {
		if asset.Asset.ID == 201 && asset.Quantity == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected index holdings in assets: %+v", assets)
	}

	var redeemResult IndexActionResult
	postJSON(t, server.URL+"/indices/201/redeem", testAPIKeyUser1, indexActionRequest{UserID: 1, Quantity: 1}, &redeemResult)
	if redeemResult.Quantity != 1 {
		t.Fatalf("unexpected redeem response: %+v", redeemResult)
	}
}

func TestWebSocketTickerSubscription(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?api_key=" + testAPIKeyUser1
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(wsRequest{Op: "subscribe", Args: []string{"market.ticker"}}); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg wsMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read message: %v", err)
	}
	if msg.Topic != "market.ticker" {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
}

func postJSON(t *testing.T, url, apiKey string, payload interface{}, target interface{}) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apiKeyHeader, apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if target == nil {
		return
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}
