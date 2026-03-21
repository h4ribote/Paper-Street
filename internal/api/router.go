package api

import (
	"net/http"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

func NewRouter(e *engine.Engine) http.Handler {
	srv := &Server{Engine: e}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/orders", srv.handleOrders)
	mux.HandleFunc("/orders/", srv.handleOrderByID)
	mux.HandleFunc("/market/orderbook/", srv.handleOrderBook)
	mux.HandleFunc("/assets", srv.handleAssets)
	return mux
}
