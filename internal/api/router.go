package api

import (
	"net/http"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func NewRouter(e *engine.Engine, apiKeys *auth.APIKeyCache) http.Handler {
	srv := &Server{Engine: e, APIKeys: apiKeys}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/orders", srv.handleOrders)
	mux.HandleFunc("/orders/", srv.handleOrderByID)
	mux.HandleFunc("/market/orderbook/", srv.handleOrderBook)
	mux.HandleFunc("/assets", srv.handleAssets)
	return srv.withAPIKeyAuth(mux)
}
