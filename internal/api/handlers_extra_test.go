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

type wsRawMessage struct {
	Topic string          `json:"topic"`
	Data  json.RawMessage `json:"data"`
	TS    int64           `json:"ts"`
}

func TestAuthCallbackIssuesAPIKey(t *testing.T) {
	const (
		tokenCode    = "test-code"
		accessToken  = "test-token"
		discordID    = "123456789012345678"
		displayName  = "Paper Street User"
		clientID     = "client-id"
		clientSecret = "client-secret"
		redirectURI  = "http://localhost:8000/auth/callback"
	)
	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth2/token":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected token method: %s", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("failed to parse token form: %v", err)
			}
			if got := r.PostForm.Get("code"); got != tokenCode {
				t.Fatalf("unexpected oauth code: %s", got)
			}
			if got := r.PostForm.Get("client_id"); got != clientID {
				t.Fatalf("unexpected client id: %s", got)
			}
			if got := r.PostForm.Get("client_secret"); got != clientSecret {
				t.Fatalf("unexpected client secret: %s", got)
			}
			if got := r.PostForm.Get("redirect_uri"); got != redirectURI {
				t.Fatalf("unexpected redirect uri: %s", got)
			}
			respondJSON(w, http.StatusOK, discordTokenResponse{AccessToken: accessToken, TokenType: "Bearer"})
		case "/api/users/@me":
			if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
				t.Fatalf("unexpected authorization header: %s", got)
			}
			respondJSON(w, http.StatusOK, discordUserResponse{ID: discordID, Username: "paper-street", GlobalName: displayName})
		default:
			http.NotFound(w, r)
		}
	}))
	defer discordServer.Close()

	t.Setenv("DISCORD_CLIENT_ID", clientID)
	t.Setenv("DISCORD_CLIENT_SECRET", clientSecret)
	t.Setenv("DISCORD_REDIRECT_URI", redirectURI)
	t.Setenv("DISCORD_API_BASE_URL", discordServer.URL+"/api")

	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, "admin"))
	defer server.Close()

	resp, err := http.Get(server.URL + "/auth/callback?code=" + tokenCode)
	if err != nil {
		t.Fatalf("failed to call auth callback: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	var payload authResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.APIKey == "" || len(payload.APIKey) != auth.APIKeyHexLength {
		t.Fatalf("unexpected api key: %q", payload.APIKey)
	}
	if !apiKeys.ContainsHex(payload.APIKey) {
		t.Fatalf("expected api key to be cached")
	}
	userID, err := strconv.ParseInt(discordID, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse discord id: %v", err)
	}
	user, ok := store.User(userID)
	if !ok {
		t.Fatalf("expected user to be created")
	}
	if user.Username != displayName {
		t.Fatalf("expected username %q, got %q", displayName, user.Username)
	}

	resp2, err := http.Get(server.URL + "/auth/callback?code=" + tokenCode)
	if err != nil {
		t.Fatalf("failed to call auth callback again: %v", err)
	}
	defer resp2.Body.Close()
	var payload2 authResponse
	if err := json.NewDecoder(resp2.Body).Decode(&payload2); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if payload2.APIKey != payload.APIKey {
		t.Fatalf("expected api key to be reused")
	}
}

func TestPoolPositionLifecycle(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	store.mu.Lock()
	store.balances[1]["VDP"] = 1_000
	store.mu.Unlock()
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
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
	store.mu.Lock()
	store.positions[1][101] = 2
	store.positions[1][102] = 2
	store.positions[1][103] = 2
	store.theoreticalFXRates = []TheoreticalFXRate{
		{BaseCurrency: "BRB", QuoteCurrency: fxBaseCurrency, Rate: fxTheoreticalScale},
		{BaseCurrency: "DRL", QuoteCurrency: fxBaseCurrency, Rate: fxTheoreticalScale},
	}
	definition := store.ensureIndexLocked(201)
	nav := store.indexUnitPriceLocked(definition)
	band, err := calculateFeeBps(nav, indexArbBandBps)
	if err != nil {
		store.mu.Unlock()
		t.Fatalf("failed to calculate arbitrage band: %v", err)
	}
	store.lastPrices[201] = nav + band + 1
	store.mu.Unlock()
	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	var createResult IndexActionResult
	postJSON(t, server.URL+"/indices/201/create", testAPIKeyUser1, indexActionRequest{UserID: 1, Quantity: 2}, &createResult)
	if createResult.AssetID != 201 || createResult.Quantity != 2 {
		t.Fatalf("unexpected create response: %+v", createResult)
	}
	store.mu.RLock()
	if store.positions[1][201] != 2 {
		store.mu.RUnlock()
		t.Fatalf("expected index holdings 2, got %d", store.positions[1][201])
	}
	if store.positions[1][101] != 0 || store.positions[1][102] != 0 || store.positions[1][103] != 0 {
		store.mu.RUnlock()
		t.Fatalf("expected component holdings to be delivered, got %+v", store.positions[1])
	}
	store.mu.RUnlock()

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

	store.mu.Lock()
	store.lastPrices[201] = nav - band - 1
	store.mu.Unlock()

	var redeemResult IndexActionResult
	postJSON(t, server.URL+"/indices/201/redeem", testAPIKeyUser1, indexActionRequest{UserID: 1, Quantity: 1}, &redeemResult)
	if redeemResult.Quantity != 1 {
		t.Fatalf("unexpected redeem response: %+v", redeemResult)
	}
	store.mu.RLock()
	if store.positions[1][201] != 1 {
		store.mu.RUnlock()
		t.Fatalf("expected index holdings 1, got %d", store.positions[1][201])
	}
	if store.positions[1][101] != 1 || store.positions[1][102] != 1 || store.positions[1][103] != 1 {
		store.mu.RUnlock()
		t.Fatalf("expected component holdings returned, got %+v", store.positions[1])
	}
	store.mu.RUnlock()
}

func TestIndexUnitPriceUsesFXRates(t *testing.T) {
	store := NewMarketStore()
	store.mu.Lock()
	store.theoreticalFXRates = []TheoreticalFXRate{
		{BaseCurrency: "BRB", QuoteCurrency: fxBaseCurrency, Rate: 150},
		{BaseCurrency: "DRL", QuoteCurrency: fxBaseCurrency, Rate: 200},
	}
	definition := store.ensureIndexLocked(201)
	price := store.indexUnitPriceLocked(definition)
	base101 := store.basePrices[101]
	base102 := store.basePrices[102]
	base103 := store.basePrices[103]
	expected := int64(0)
	expected += base101
	expected += (base102 * 150) / fxTheoreticalScale
	expected += (base103 * 200) / fxTheoreticalScale
	store.mu.Unlock()

	if price != expected {
		t.Fatalf("expected index price %d, got %d", expected, price)
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
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
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
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	var msg wsMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read message: %v", err)
	}
	if msg.Topic != "market.ticker" {
		t.Fatalf("unexpected topic: %s", msg.Topic)
	}
}

func TestWebSocketOrderBookDelta(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)
	eng := engine.NewEngine(store)
	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   1,
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 1,
		Price:    100,
	})
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?api_key=" + testAPIKeyUser1
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(wsRequest{Op: "subscribe", Args: []string{"market.orderbook.101"}}); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	var snapshotMsg wsRawMessage
	if err := conn.ReadJSON(&snapshotMsg); err != nil {
		t.Fatalf("failed to read snapshot: %v", err)
	}
	if snapshotMsg.Topic != "market.orderbook.101" {
		t.Fatalf("unexpected topic: %s", snapshotMsg.Topic)
	}
	var snapshot engine.OrderBookSnapshot
	if err := json.Unmarshal(snapshotMsg.Data, &snapshot); err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}
	if len(snapshot.Bids) != 1 || snapshot.Bids[0].Price != 100 {
		t.Fatalf("unexpected snapshot bids: %+v", snapshot.Bids)
	}

	submitEngineOrder(t, eng, &engine.Order{
		AssetID:  101,
		UserID:   1,
		Side:     engine.SideBuy,
		Type:     engine.OrderTypeLimit,
		Quantity: 1,
		Price:    110,
	})

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	var deltaMsg wsRawMessage
	if err := conn.ReadJSON(&deltaMsg); err != nil {
		t.Fatalf("failed to read delta: %v", err)
	}
	var delta engine.OrderBookSnapshot
	if err := json.Unmarshal(deltaMsg.Data, &delta); err != nil {
		t.Fatalf("failed to decode delta: %v", err)
	}
	if len(delta.Bids) != 1 || delta.Bids[0].Price != 110 {
		t.Fatalf("expected delta with new price 110, got %+v", delta.Bids)
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
