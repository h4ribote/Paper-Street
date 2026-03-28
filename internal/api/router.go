package api

import (
	"net/http"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func NewRouter(e *engine.Engine, apiKeys *auth.APIKeyCache, store *MarketStore) http.Handler {
	srv := &Server{Engine: e, APIKeys: apiKeys, Store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/auth/login", srv.handleAuthLogin)
	mux.HandleFunc("/auth/callback", srv.handleAuthCallback)
	mux.HandleFunc("/auth/refresh", srv.handleAuthRefresh)
	mux.HandleFunc("/auth/logout", srv.handleAuthLogout)
	mux.HandleFunc("/users/me", srv.handleCurrentUser)
	mux.HandleFunc("/orders", srv.handleOrders)
	mux.HandleFunc("/orders/", srv.handleOrderByID)
	mux.HandleFunc("/market/orderbook/", srv.handleOrderBook)
	mux.HandleFunc("/market/trades/", srv.handleTrades)
	mux.HandleFunc("/market/candles/", srv.handleCandles)
	mux.HandleFunc("/market/ticker", srv.handleTicker)
	mux.HandleFunc("/news", srv.handleNews)
	mux.HandleFunc("/macro/indicators", srv.handleMacroIndicators)
	mux.HandleFunc("/assets", srv.handleAssets)
	mux.HandleFunc("/assets/", srv.handleAssetByID)
	mux.HandleFunc("/portfolio/balances", srv.handlePortfolioBalances)
	mux.HandleFunc("/portfolio/assets", srv.handlePortfolioAssets)
	mux.HandleFunc("/portfolio/positions", srv.handlePortfolioPositions)
	mux.HandleFunc("/portfolio/history", srv.handlePortfolioHistory)
	mux.HandleFunc("/portfolio/performance", srv.handlePortfolioPerformance)
	mux.HandleFunc("/pools", srv.handlePools)
	mux.HandleFunc("/pools/", srv.handlePoolByID)
	mux.HandleFunc("/pools/positions", srv.handlePoolPositions)
	mux.HandleFunc("/pools/positions/", srv.handlePoolPositionByID)
	mux.HandleFunc("/margin/pools", srv.handleMarginPools)
	mux.HandleFunc("/margin/pools/", srv.handleMarginPoolByID)
	mux.HandleFunc("/world/seasons/current", srv.handleCurrentSeason)
	mux.HandleFunc("/world/regions", srv.handleWorldRegions)
	mux.HandleFunc("/world/companies", srv.handleWorldCompanies)
	mux.HandleFunc("/world/events", srv.handleWorldEvents)
	mux.HandleFunc("/leaderboard", srv.handleLeaderboard)
	mux.HandleFunc("/indices/", srv.handleIndices)
	return srv.withAPIKeyAuth(mux)
}
