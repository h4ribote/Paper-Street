package api

import (
	"crypto/rand"
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
	respondJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handlePoolByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/pools/")
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		respondError(w, http.StatusBadRequest, "pool id required")
		return
	}
	poolID, err := parseID(segments[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid pool id")
		return
	}
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		respondJSON(w, http.StatusOK, map[string]int64{"pool_id": poolID})
		return
	}
	switch segments[1] {
	case "positions":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"pool_id": poolID, "status": "position_created"})
	case "swap":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"pool_id": poolID, "status": "swap_executed"})
	default:
		respondError(w, http.StatusNotFound, "unknown pool action")
	}
}

func (s *Server) handlePoolPositions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	respondJSON(w, http.StatusOK, []interface{}{})
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
	respondJSON(w, http.StatusOK, map[string]interface{}{"position_id": positionID, "status": "closed"})
}

func (s *Server) handleMarginPools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	respondJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handleMarginPoolByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/margin/pools/")
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		respondError(w, http.StatusBadRequest, "pool id required")
		return
	}
	poolID, err := parseID(segments[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid pool id")
		return
	}
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		respondJSON(w, http.StatusOK, map[string]int64{"pool_id": poolID})
		return
	}
	switch segments[1] {
	case "supply":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"pool_id": poolID, "status": "supplied"})
	case "withdraw":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"pool_id": poolID, "status": "withdrawn"})
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
		respondJSON(w, http.StatusOK, map[string]interface{}{"asset_id": assetID, "action": action, "status": "ok"})
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
