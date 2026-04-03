package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

type Server struct {
	Engine           *engine.Engine
	APIKeys          *auth.APIKeyCache
	Store            *MarketStore
	WSHub            *wsHub
	AdminPassword    string
	marketCooldownMu sync.Mutex
	marketCooldown   map[marketCooldownKey]time.Time
}

type orderRequest struct {
	AssetID     int64  `json:"asset_id"`
	UserID      int64  `json:"user_id"`
	Side        string `json:"side"`
	Type        string `json:"type"`
	TimeInForce string `json:"time_in_force"`
	Quantity    int64  `json:"quantity"`
	Price       int64  `json:"price"`
	StopPrice   int64  `json:"stop_price"`
	Leverage    int64  `json:"leverage"`
}

type errorResponse struct {
	Error string `json:"error"`
}

const (
	defaultOrderBookDepth = 20
	maxOrderBookDepth     = 100
	marketOrderCooldown   = 5 * time.Second
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateOrder(w, r)
	case http.MethodGet:
		statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
		userID, authStatus, message := s.resolveUserID(r, parseUserID(r), false)
		if authStatus != 0 {
			respondError(w, authStatus, message)
			return
		}
		assetID := parseQueryInt64(r, "asset_id")
		var status engine.OrderStatus
		if statusFilter != "" {
			status = engine.OrderStatus(strings.ToUpper(statusFilter))
			switch status {
			case engine.OrderStatusOpen, engine.OrderStatusPartial, engine.OrderStatusFilled, engine.OrderStatusCancelled, engine.OrderStatusRejected:
			default:
				respondError(w, http.StatusBadRequest, "invalid status filter")
				return
			}
		}
		if s.Store == nil {
			respondJSON(w, http.StatusOK, []engine.Order{})
			return
		}
		limit := parseLimit(r, 0)
		offset := parseOffset(r, 0)
		orders := s.Store.Orders(OrderFilter{UserID: userID, Status: status, AssetID: assetID})
		if offset > 0 {
			if offset >= len(orders) {
				respondJSON(w, http.StatusOK, []engine.Order{})
				return
			}
			orders = orders[offset:]
		}
		if limit > 0 && len(orders) > limit {
			orders = orders[:limit]
		}
		respondJSON(w, http.StatusOK, orders)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(strings.TrimPrefix(r.URL.Path, "/orders/"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		assetID := parseQueryInt64(r, "asset_id")
		if assetID == 0 {
			respondError(w, http.StatusBadRequest, "asset_id required")
			return
		}
		order, ok := s.Engine.FindOrder(assetID, id)
		if !ok && s.Store != nil {
			order, ok = s.Store.OrderForAsset(id, assetID)
		}
		if !ok {
			respondError(w, http.StatusNotFound, "order not found or does not belong to asset")
			return
		}
		respondJSON(w, http.StatusOK, order)
	case http.MethodDelete:
		assetID := parseQueryInt64(r, "asset_id")
		if assetID == 0 {
			respondError(w, http.StatusBadRequest, "asset_id required")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		result, err := s.Engine.CancelOrder(ctx, assetID, id)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var payload orderRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	userID, status, message := s.resolveUserID(r, payload.UserID, true)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	order, err := payload.toOrder(userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	previousTimestamp, hadPreviousEntry, err := s.checkAndSetMarketCooldown(order)
	if err != nil {
		respondError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	result, err := s.Engine.SubmitOrder(ctx, order)
	if err != nil {
		s.restoreMarketCooldown(order, previousTimestamp, hadPreviousEntry)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleOrderBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, err := parseID(strings.TrimPrefix(r.URL.Path, "/market/orderbook/"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}
	depth := defaultOrderBookDepth
	if value := r.URL.Query().Get("depth"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			depth = parsed
		}
	}
	if depth > maxOrderBookDepth {
		depth = maxOrderBookDepth
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	// Ensure the order book exists before snapshotting.
	_ = s.Engine.OrderBook(id)
	snapshot, err := s.Engine.Snapshot(ctx, id, depth)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []models.Asset{})
		return
	}
	filter := AssetFilter{
		Type:   r.URL.Query().Get("type"),
		Sector: r.URL.Query().Get("sector"),
	}
	respondJSON(w, http.StatusOK, s.Store.Assets(filter))
}

func (o orderRequest) toOrder(defaultUserID int64) (*engine.Order, error) {
	side := strings.ToUpper(o.Side)
	orderSide := engine.Side(side)
	if orderSide != engine.SideBuy && orderSide != engine.SideSell {
		return nil, errors.New("invalid side")
	}
	orderType := engine.OrderType(strings.ToUpper(o.Type))
	switch orderType {
	case engine.OrderTypeLimit, engine.OrderTypeMarket, engine.OrderTypeStop, engine.OrderTypeStopLimit:
	default:
		return nil, errors.New("invalid order type")
	}
	if o.AssetID == 0 {
		return nil, errors.New("asset_id required")
	}
	if o.Quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}
	userID := o.UserID
	if userID == 0 {
		userID = defaultUserID
	}
	if userID == 0 {
		return nil, errors.New("user_id required in request or via authentication")
	}
	if (orderType == engine.OrderTypeLimit || orderType == engine.OrderTypeStopLimit) && o.Price <= 0 {
		return nil, errors.New("price required for limit orders")
	}
	if (orderType == engine.OrderTypeStop || orderType == engine.OrderTypeStopLimit) && o.StopPrice <= 0 {
		return nil, errors.New("stop_price required for stop orders")
	}
	leverage := o.Leverage
	if leverage == 0 {
		leverage = 1
	}
	if leverage < 1 {
		return nil, errors.New("leverage must be at least 1")
	}
	if leverage > marginLeverageMax {
		return nil, errors.New("leverage must be at most 5")
	}
	timeInForce := strings.ToUpper(strings.TrimSpace(o.TimeInForce))
	if timeInForce == "" {
		timeInForce = string(engine.TimeInForceGTC)
	}
	switch engine.TimeInForce(timeInForce) {
	case engine.TimeInForceGTC, engine.TimeInForceIOC, engine.TimeInForceFOK:
	default:
		return nil, errors.New("invalid time in force")
	}
	return &engine.Order{
		AssetID:     o.AssetID,
		UserID:      userID,
		Side:        orderSide,
		Type:        orderType,
		TimeInForce: engine.TimeInForce(timeInForce),
		Quantity:    o.Quantity,
		Price:       o.Price,
		StopPrice:   o.StopPrice,
		Leverage:    leverage,
	}, nil
}

type marketCooldownKey struct {
	userID int64
	side   engine.Side
	assetID int64
}

func (s *Server) checkAndSetMarketCooldown(order *engine.Order) (time.Time, bool, error) {
	if s == nil || order == nil || order.Type != engine.OrderTypeMarket {
		return time.Time{}, false, nil
	}
	now := time.Now()
	key := marketCooldownKey{userID: order.UserID, side: order.Side, assetID: order.AssetID}
	s.marketCooldownMu.Lock()
	defer s.marketCooldownMu.Unlock()
	if s.marketCooldown == nil {
		s.marketCooldown = make(map[marketCooldownKey]time.Time)
	}
	last, ok := s.marketCooldown[key]
	if ok && last.Add(marketOrderCooldown).After(now) {
		remaining := last.Add(marketOrderCooldown).Sub(now)
		remainingSeconds := int(math.Ceil(remaining.Seconds()))
		return time.Time{}, false, fmt.Errorf("market order cooldown active, retry in %d seconds", remainingSeconds)
	}
	s.marketCooldown[key] = now
	return last, ok, nil
}

func (s *Server) restoreMarketCooldown(order *engine.Order, previous time.Time, hadPrevious bool) {
	if s == nil || order == nil || order.Type != engine.OrderTypeMarket {
		return
	}
	key := marketCooldownKey{userID: order.UserID, side: order.Side, assetID: order.AssetID}
	s.marketCooldownMu.Lock()
	defer s.marketCooldownMu.Unlock()
	if !hadPrevious {
		delete(s.marketCooldown, key)
		return
	}
	s.marketCooldown[key] = previous
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		log.Printf("response encode error: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("JSON encoding failed for response payload")); err != nil {
			log.Printf("response encode fallback error: %v", err)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Printf("response write error: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, errorResponse{Error: message})
}
