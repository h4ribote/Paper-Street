package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/auth"
)

type authResponse struct {
	APIKey string      `json:"api_key"`
	User   interface{} `json:"user"`
}

type statusResponse struct {
	Status string `json:"status"`
}

type poolPositionRequest struct {
	UserID      int64 `json:"user_id"`
	BaseAmount  int64 `json:"base_amount"`
	QuoteAmount int64 `json:"quote_amount"`
	LowerTick   int64 `json:"lower_tick"`
	UpperTick   int64 `json:"upper_tick"`
}

type poolSwapRequest struct {
	UserID       int64  `json:"user_id"`
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	Amount       int64  `json:"amount"`
}

type marginPoolRequest struct {
	UserID      int64 `json:"user_id"`
	CashAmount  int64 `json:"cash_amount"`
	AssetAmount int64 `json:"asset_amount"`
}

type indexActionRequest struct {
	UserID   int64 `json:"user_id"`
	Quantity int64 `json:"quantity"`
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.issueAuthKey(w, r)
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.issueAuthKey(w, r)
}

func (s *Server) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil || s.APIKeys == nil {
		respondError(w, http.StatusInternalServerError, "auth store unavailable")
		return
	}
	oldKey := strings.TrimSpace(r.Header.Get(apiKeyHeader))
	if oldKey == "" {
		respondError(w, http.StatusUnauthorized, "API key required")
		return
	}
	user, ok := s.Store.UserForAPIKey(oldKey)
	if !ok {
		respondError(w, http.StatusUnauthorized, "API key not associated with user")
		return
	}
	newKey, err := generateAPIKeyHex()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}
	if err := s.APIKeys.RemoveHex(oldKey); err != nil {
		respondError(w, http.StatusBadRequest, "invalid API key")
		return
	}
	s.Store.UnregisterAPIKey(oldKey)
	if err := s.APIKeys.AddHex(newKey); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to register API key")
		return
	}
	if user.ID != 0 {
		s.Store.RegisterAPIKey(newKey, user.ID)
	}
	respondJSON(w, http.StatusOK, authResponse{APIKey: newKey, User: user})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil || s.APIKeys == nil {
		respondError(w, http.StatusInternalServerError, "auth store unavailable")
		return
	}
	apiKey := strings.TrimSpace(r.Header.Get(apiKeyHeader))
	if apiKey == "" {
		respondError(w, http.StatusUnauthorized, "API key required")
		return
	}
	if err := s.APIKeys.RemoveHex(apiKey); err != nil {
		respondError(w, http.StatusBadRequest, "invalid API key")
		return
	}
	s.Store.UnregisterAPIKey(apiKey)
	respondJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	if userID == 0 {
		respondError(w, http.StatusBadRequest, "user_id required")
		return
	}
	user, _ := s.Store.User(userID)
	respondJSON(w, http.StatusOK, user)
}

func (s *Server) handleAssetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	id, err := parseID(strings.TrimPrefix(r.URL.Path, "/assets/"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}
	asset, ok := s.Store.Asset(id)
	if !ok {
		respondError(w, http.StatusNotFound, "asset not found")
		return
	}
	respondJSON(w, http.StatusOK, asset)
}

func (s *Server) handleTrades(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}
	assetID, err := parseID(strings.TrimPrefix(r.URL.Path, "/market/trades/"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}
	limit := parseLimit(r, 50)
	respondJSON(w, http.StatusOK, s.Store.Executions(assetID, limit))
}

func (s *Server) handleCandles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []Candle{})
		return
	}
	assetID, err := parseID(strings.TrimPrefix(r.URL.Path, "/market/candles/"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}
	timeframe := time.Minute
	if value := r.URL.Query().Get("timeframe"); value != "" {
		if parsed, ok := parseTimeframe(value); ok {
			timeframe = parsed
		}
	}
	limit := parseLimit(r, 60)
	startTime, _ := parseUnixMillis(r.URL.Query().Get("start_time"))
	endTime, _ := parseUnixMillis(r.URL.Query().Get("end_time"))
	respondJSON(w, http.StatusOK, s.Store.Candles(assetID, timeframe, startTime, endTime, limit))
}

func (s *Server) handleTicker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []Ticker{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Tickers())
}

func (s *Server) handleNews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []NewsItem{})
		return
	}
	limit := parseLimit(r, 50)
	respondJSON(w, http.StatusOK, s.Store.News(limit))
}

func (s *Server) handleMacroIndicators(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []MacroIndicator{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.MacroIndicators())
}

func (s *Server) handlePortfolioBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Balances(userID))
}

func (s *Server) handlePortfolioAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.PortfolioAssets(userID))
}

func (s *Server) handlePortfolioPositions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Positions(userID))
}

func (s *Server) handlePortfolioHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}
	limit := parseLimit(r, 100)
	respondJSON(w, http.StatusOK, s.Store.TradeHistory(userID, limit))
}

func (s *Server) handlePortfolioPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Performance(userID))
}

func (s *Server) handlePools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []LiquidityPool{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Pools())
}

func (s *Server) handlePoolByID(w http.ResponseWriter, r *http.Request) {
	poolID, segments, err := parsePathID(r.URL.Path, "/pools/")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid pool id")
		return
	}
	if len(segments) == 0 {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondJSON(w, http.StatusOK, LiquidityPool{})
			return
		}
		pool, ok := s.Store.Pool(poolID)
		if !ok {
			respondError(w, http.StatusNotFound, "pool not found")
			return
		}
		respondJSON(w, http.StatusOK, pool)
		return
	}
	switch segments[0] {
	case "positions":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		var payload poolPositionRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		userID := payload.UserID
		if userID == 0 {
			userID = s.userIDFromRequest(r)
		}
		position, err := s.Store.CreatePoolPosition(poolID, userID, payload.BaseAmount, payload.QuoteAmount, payload.LowerTick, payload.UpperTick)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, position)
	case "swap":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		var payload poolSwapRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		userID := payload.UserID
		if userID == 0 {
			userID = s.userIDFromRequest(r)
		}
		result, err := s.Store.SwapPool(poolID, userID, payload.FromCurrency, payload.ToCurrency, payload.Amount)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	default:
		respondError(w, http.StatusNotFound, "unknown pool action")
	}
}

func (s *Server) handlePoolPositions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []PoolPosition{})
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	respondJSON(w, http.StatusOK, s.Store.PoolPositions(userID))
}

func (s *Server) handlePoolPositionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	positionID, err := parseID(strings.TrimPrefix(r.URL.Path, "/pools/positions/"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid position id")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	userID := parseUserID(r)
	if userID == 0 {
		userID = s.userIDFromRequest(r)
	}
	position, err := s.Store.ClosePoolPosition(userID, positionID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, position)
}

func (s *Server) handleMarginPools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []MarginPool{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.MarginPools())
}

func (s *Server) handleMarginPoolByID(w http.ResponseWriter, r *http.Request) {
	poolID, segments, err := parsePathID(r.URL.Path, "/margin/pools/")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid pool id")
		return
	}
	if len(segments) == 0 {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondJSON(w, http.StatusOK, MarginPool{})
			return
		}
		pool, ok := s.Store.MarginPool(poolID)
		if !ok {
			respondError(w, http.StatusNotFound, "margin pool not found")
			return
		}
		respondJSON(w, http.StatusOK, pool)
		return
	}
	switch segments[0] {
	case "supply":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		var payload marginPoolRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		userID := payload.UserID
		if userID == 0 {
			userID = s.userIDFromRequest(r)
		}
		result, err := s.Store.SupplyMarginPool(poolID, userID, payload.CashAmount, payload.AssetAmount)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	case "withdraw":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		var payload marginPoolRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		userID := payload.UserID
		if userID == 0 {
			userID = s.userIDFromRequest(r)
		}
		result, err := s.Store.WithdrawMarginPool(poolID, userID, payload.CashAmount, payload.AssetAmount)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	default:
		respondError(w, http.StatusNotFound, "unknown margin action")
	}
}

func (s *Server) handleCurrentSeason(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, Season{})
		return
	}
	seasons := s.Store.Seasons()
	if len(seasons) == 0 {
		respondJSON(w, http.StatusOK, Season{})
		return
	}
	respondJSON(w, http.StatusOK, seasons[0])
}

func (s *Server) handleWorldRegions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []Region{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Regions())
}

func (s *Server) handleWorldCompanies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []Company{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Companies())
}

func (s *Server) handleWorldEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []WorldEvent{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.WorldEvents())
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []LeaderboardEntry{})
		return
	}
	limit := parseLimit(r, 20)
	respondJSON(w, http.StatusOK, s.Store.Leaderboard(limit))
}

func (s *Server) handleIndices(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/indices/")
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 {
		respondError(w, http.StatusBadRequest, "asset id and action required")
		return
	}
	assetID, err := parseID(segments[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}
	action := segments[1]
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	switch action {
	case "create", "redeem":
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		var payload indexActionRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		quantity := payload.Quantity
		if quantity == 0 {
			quantity = 1
		}
		userID := payload.UserID
		if userID == 0 {
			userID = s.userIDFromRequest(r)
		}
		var result IndexActionResult
		var err error
		if action == "create" {
			result, err = s.Store.CreateIndex(userID, assetID, quantity)
		} else {
			result, err = s.Store.RedeemIndex(userID, assetID, quantity)
		}
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	default:
		respondError(w, http.StatusNotFound, "unknown index action")
	}
}

func (s *Server) issueAuthKey(w http.ResponseWriter, r *http.Request) {
	if s.Store == nil || s.APIKeys == nil {
		respondError(w, http.StatusInternalServerError, "auth store unavailable")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	user := s.Store.AddUser(username)
	key, err := generateAPIKeyHex()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}
	if err := s.APIKeys.AddHex(key); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to register API key")
		return
	}
	s.Store.RegisterAPIKey(key, user.ID)
	respondJSON(w, http.StatusOK, authResponse{APIKey: key, User: user})
}

func (s *Server) userIDFromRequest(r *http.Request) int64 {
	if s == nil || s.Store == nil || r == nil {
		return 0
	}
	apiKey := strings.TrimSpace(r.Header.Get(apiKeyHeader))
	if apiKey == "" {
		return 0
	}
	user, ok := s.Store.UserForAPIKey(apiKey)
	if !ok {
		return 0
	}
	return user.ID
}

func generateAPIKeyHex() (string, error) {
	var key auth.APIKey
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	return key.String(), nil
}
