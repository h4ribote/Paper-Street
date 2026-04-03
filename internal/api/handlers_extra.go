package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

type authResponse struct {
	APIKey string      `json:"api_key"`
	User   interface{} `json:"user"`
}

type botAuthRequest struct {
	Role          string `json:"role"`
	AdminPassword string `json:"admin_password"`
}

type discordTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type discordUserResponse struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
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

type marginTopUpRequest struct {
	UserID int64 `json:"user_id"`
	Amount int64 `json:"amount"`
}

type indexActionRequest struct {
	UserID   int64 `json:"user_id"`
	Quantity int64 `json:"quantity"`
}

type bondOperationRequest struct {
	Quantity    int64 `json:"quantity"`
	PremiumBps  int64 `json:"premium_bps"`
	DiscountBps int64 `json:"discount_bps"`
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil || s.APIKeys == nil {
		respondError(w, http.StatusInternalServerError, "auth store unavailable")
		return
	}
	var payload botAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	role := strings.TrimSpace(payload.Role)
	if role == "" {
		respondError(w, http.StatusBadRequest, "role required")
		return
	}
	if !s.validAdminPassword(payload.AdminPassword) {
		respondError(w, http.StatusUnauthorized, "invalid admin password")
		return
	}
	key, user, ok := s.Store.APIKeyForRole(role)
	if !ok || key == "" {
		respondError(w, http.StatusNotFound, "role not found")
		return
	}
	if !s.APIKeys.ContainsHex(key) {
		if err := s.APIKeys.AddHex(key); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to register api key")
			return
		}
	}
	respondJSON(w, http.StatusOK, authResponse{APIKey: key, User: user})
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil || s.APIKeys == nil {
		respondError(w, http.StatusInternalServerError, "auth store unavailable")
		return
	}
	if errParam := strings.TrimSpace(r.URL.Query().Get("error")); errParam != "" {
		respondError(w, http.StatusBadRequest, "discord oauth error: "+errParam)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		respondError(w, http.StatusBadRequest, "code required")
		return
	}
	expectedState := strings.TrimSpace(os.Getenv("DISCORD_OAUTH_STATE"))
	if expectedState == "" {
		respondError(w, http.StatusInternalServerError, "discord oauth state not configured")
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		respondError(w, http.StatusBadRequest, "state required")
		return
	}
	if !subtleCompare(expectedState, state) {
		respondError(w, http.StatusBadRequest, "invalid oauth state")
		return
	}
	clientID := strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET"))
	redirectURI := strings.TrimSpace(os.Getenv("DISCORD_REDIRECT_URI"))
	if clientID == "" || clientSecret == "" || redirectURI == "" {
		respondError(w, http.StatusInternalServerError, "discord oauth not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	accessToken, err := exchangeDiscordToken(ctx, code, clientID, clientSecret, redirectURI)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	discordUser, err := fetchDiscordUser(ctx, accessToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(discordUser.ID), 10, 64)
	if err != nil || userID == 0 {
		respondError(w, http.StatusBadGateway, "invalid discord user id")
		return
	}
	displayName := discordDisplayName(discordUser, userID)
	user := s.Store.EnsureUserWithName(userID, displayName)
	apiKey, ok := s.Store.APIKeyForUser(userID)
	if !ok || apiKey == "" {
		apiKey, err = generateAPIKeyHex()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to generate api key")
			return
		}
		s.Store.RegisterAPIKey(apiKey, userID)
		s.Store.persistAPIKey(discordRoleForUser(userID), apiKey, userID)
	}
	if !s.APIKeys.ContainsHex(apiKey) {
		if err := s.APIKeys.AddHex(apiKey); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to register api key")
			return
		}
	}
	respondJSON(w, http.StatusOK, authResponse{APIKey: apiKey, User: user})
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), true)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	user, ok := s.Store.UserByID(userID)
	if !ok || user.ID == 0 {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
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

func (s *Server) handleBonds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []PerpetualBondInfo{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.PerpetualBonds())
}

func (s *Server) handleBondOperations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/bonds/")
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 1 || strings.TrimSpace(segments[0]) == "" {
		respondError(w, http.StatusBadRequest, "bond id required")
		return
	}
	assetID, err := parseID(segments[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid bond id")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		info, ok := s.Store.PerpetualBond(assetID)
		if !ok {
			respondError(w, http.StatusNotFound, "bond not found")
			return
		}
		respondJSON(w, http.StatusOK, info)
		return
	}
	action := segments[1]
	switch action {
	case "issue", "buyback":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var payload bondOperationRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		quantity := payload.Quantity
		if quantity <= 0 {
			if action == "issue" {
				quantity = bondDefaultIssueQty
			} else {
				quantity = bondDefaultBuybackQty
			}
		}
		premiumBps := payload.PremiumBps
		discountBps := payload.DiscountBps
		if action == "issue" {
			premiumBps = 0
			if discountBps <= 0 {
				discountBps = bondIssueDiscountBps
			}
		} else {
			discountBps = 0
			if premiumBps <= 0 {
				premiumBps = bondBuybackPremiumBps
			}
		}
		s.Store.mu.Lock()
		def, ok := s.Store.perpetualBonds[assetID]
		if !ok {
			s.Store.mu.Unlock()
			respondError(w, http.StatusNotFound, "bond not found")
			return
		}
		issuerID := s.Store.ensureBondIssuerLocked(def)
		price, targetYield := s.Store.bondOperationPriceLocked(def, premiumBps, discountBps)
		s.Store.mu.Unlock()
		if price <= 0 {
			respondError(w, http.StatusBadRequest, "unable to compute bond price")
			return
		}
		side := engine.SideSell
		if action == "buyback" {
			side = engine.SideBuy
		}
		order := &engine.Order{
			UserID:   issuerID,
			AssetID:  assetID,
			Side:     side,
			Type:     engine.OrderTypeLimit,
			Quantity: quantity,
			Price:    price,
			Leverage: 1,
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		result, err := s.Engine.SubmitOrder(ctx, order)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.Store.mu.Lock()
		s.Store.bondOperationNewsLocked(def, action, quantity, price, time.Now().UTC())
		s.Store.mu.Unlock()
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"asset_id":         assetID,
			"action":           action,
			"quantity":         quantity,
			"price":            price,
			"target_yield_bps": targetYield,
			"result":           result,
		})
	case "coupons":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		payments := s.Store.TriggerPerpetualBondCoupons(time.Now().UTC())
		filtered := make([]BondCouponPayment, 0, len(payments))
		for _, payment := range payments {
			if payment.AssetID == assetID {
				filtered = append(filtered, payment)
			}
		}
		respondJSON(w, http.StatusOK, filtered)
	default:
		respondError(w, http.StatusNotFound, "unknown bond action")
	}
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

func (s *Server) handleTheoreticalFXRates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []TheoreticalFXRate{})
		return
	}
	respondJSON(w, http.StatusOK, s.Store.TheoreticalFXRates())
}

func (s *Server) handlePortfolioBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
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
		userID, status, message := s.resolveUserID(r, payload.UserID, true)
		if status != 0 {
			respondError(w, status, message)
			return
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
		userID, status, message := s.resolveUserID(r, payload.UserID, true)
		if status != 0 {
			respondError(w, status, message)
			return
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), true)
	if status != 0 {
		respondError(w, status, message)
		return
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
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	if userID != 0 {
		respondJSON(w, http.StatusOK, s.Store.MarginPoolsForUser(userID))
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
		userID, status, message := s.resolveUserID(r, parseUserID(r), false)
		if status != 0 {
			respondError(w, status, message)
			return
		}
		pool, ok := s.Store.MarginPoolForUser(poolID, userID)
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
		userID, status, message := s.resolveUserID(r, payload.UserID, true)
		if status != 0 {
			respondError(w, status, message)
			return
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
		userID, status, message := s.resolveUserID(r, payload.UserID, true)
		if status != 0 {
			respondError(w, status, message)
			return
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

func (s *Server) handleMarginPositions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []MarginPosition{})
		return
	}
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	respondJSON(w, http.StatusOK, s.Store.MarginPositions(userID))
}

func (s *Server) handleMarginPositionByID(w http.ResponseWriter, r *http.Request) {
	positionID, segments, err := parsePathID(r.URL.Path, "/margin/positions/")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid position id")
		return
	}
	if len(segments) == 0 {
		respondError(w, http.StatusNotFound, "unknown margin position action")
		return
	}
	switch segments[0] {
	case "topup":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		var payload marginTopUpRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		userID, status, message := s.resolveUserID(r, payload.UserID, true)
		if status != 0 {
			respondError(w, status, message)
			return
		}
		position, err := s.Store.AddMargin(userID, positionID, payload.Amount)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, position)
	default:
		respondError(w, http.StatusNotFound, "unknown margin position action")
	}
}

func (s *Server) handleMarginLiquidations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []MarginLiquidation{})
		return
	}
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	respondJSON(w, http.StatusOK, s.Store.MarginLiquidations(userID))
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

	// GET /indices/ — list all index definitions with current NAV.
	if r.Method == http.MethodGet && (path == "" || path == "/") {
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		respondJSON(w, http.StatusOK, s.Store.Indexes())
		return
	}

	if len(segments) < 1 || segments[0] == "" {
		respondError(w, http.StatusBadRequest, "asset id required")
		return
	}
	assetID, err := parseID(segments[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	// GET /indices/{assetID} — return index definition and current NAV.
	if r.Method == http.MethodGet && len(segments) == 1 {
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		info, ok := s.Store.Index(assetID)
		if !ok {
			respondError(w, http.StatusNotFound, "index not found")
			return
		}
		respondJSON(w, http.StatusOK, info)
		return
	}

	if len(segments) < 2 {
		respondError(w, http.StatusBadRequest, "asset id and action required")
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
		// Empty body defaults quantity to 1, matching the API spec.
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		quantity := payload.Quantity
		if quantity == 0 {
			quantity = 1
		}
		userID, status, message := s.resolveUserID(r, payload.UserID, true)
		if status != 0 {
			respondError(w, status, message)
			return
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

func (s *Server) validAdminPassword(password string) bool {
	if s == nil {
		return false
	}
	expected := strings.TrimSpace(s.AdminPassword)
	if expected == "" {
		return false
	}
	actual := strings.TrimSpace(password)
	if actual == "" {
		return false
	}
	return subtleCompare(expected, actual)
}

func subtleCompare(expected, actual string) bool {
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
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

func discordAPIBaseURL() string {
	base := strings.TrimSpace(os.Getenv("DISCORD_API_BASE_URL"))
	if base == "" {
		return "https://discord.com/api"
	}
	return strings.TrimRight(base, "/")
}

func exchangeDiscordToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (string, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordAPIBaseURL()+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := readLimitedBody(resp.Body)
		if message == "" {
			message = resp.Status
		}
		return "", fmt.Errorf("discord token exchange failed: %s", message)
	}
	var payload discordTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", errors.New("discord access token missing")
	}
	return strings.TrimSpace(payload.AccessToken), nil
}

func fetchDiscordUser(ctx context.Context, accessToken string) (discordUserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBaseURL()+"/users/@me", nil)
	if err != nil {
		return discordUserResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return discordUserResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := readLimitedBody(resp.Body)
		if message == "" {
			message = resp.Status
		}
		return discordUserResponse{}, fmt.Errorf("discord user fetch failed: %s", message)
	}
	var payload discordUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return discordUserResponse{}, err
	}
	return payload, nil
}

func readLimitedBody(reader io.Reader) string {
	const maxBody = 1024
	body, err := io.ReadAll(io.LimitReader(reader, maxBody))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func discordDisplayName(user discordUserResponse, userID int64) string {
	displayName := strings.TrimSpace(user.GlobalName)
	if displayName == "" {
		displayName = strings.TrimSpace(user.Username)
	}
	if displayName == "" && userID != 0 {
		displayName = fmt.Sprintf("user-%d", userID)
	}
	return displayName
}
