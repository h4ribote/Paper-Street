package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

type Server struct {
	Engine  *engine.Engine
	APIKeys *auth.APIKeyCache
	Store   *MarketStore
}

type orderRequest struct {
	AssetID   int64  `json:"asset_id"`
	UserID    int64  `json:"user_id"`
	Side      string `json:"side"`
	Type      string `json:"type"`
	Quantity  int64  `json:"quantity"`
	Price     int64  `json:"price"`
	StopPrice int64  `json:"stop_price"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateOrder(w, r)
	case http.MethodGet:
		statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
		userID := parseUserID(r)
		if userID == 0 {
			userID = s.userIDFromRequest(r)
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
		orders := s.Store.Orders(OrderFilter{UserID: userID, Status: status, AssetID: assetID})
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
		order, ok := s.Engine.FindOrder(id)
		if !ok && s.Store != nil {
			order, ok = s.Store.Order(id)
		}
		if !ok {
			respondError(w, http.StatusNotFound, "order not found")
			return
		}
		respondJSON(w, http.StatusOK, order)
	case http.MethodDelete:
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		result, err := s.Engine.CancelOrderByID(ctx, id)
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
	order, err := payload.toOrder(s.userIDFromRequest(r))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	result, err := s.Engine.SubmitOrder(ctx, order)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleOrderBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(strings.TrimPrefix(r.URL.Path, "/market/orderbook/"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}
	depth := 20
	if value := r.URL.Query().Get("depth"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			depth = parsed
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	book := s.Engine.OrderBook(id)
	snapshot, err := book.Snapshot(ctx, depth)
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
	return &engine.Order{
		AssetID:   o.AssetID,
		UserID:    userID,
		Side:      orderSide,
		Type:      orderType,
		Quantity:  o.Quantity,
		Price:     o.Price,
		StopPrice: o.StopPrice,
	}, nil
}

func parseID(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("id required")
	}
	if strings.Contains(value, "/") {
		return 0, errors.New("invalid id path")
	}
	return strconv.ParseInt(value, 10, 64)
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
