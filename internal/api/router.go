package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

const maxFrontendSearchDepth = 8

func NewRouter(e *engine.Engine, apiKeys *auth.APIKeyCache, store *MarketStore, adminPassword string) http.Handler {
	hub := newWSHub(store, e)
	srv := &Server{Engine: e, APIKeys: apiKeys, Store: store, WSHub: hub, AdminPassword: adminPassword}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/auth/login", srv.handleAuthLogin)
	mux.HandleFunc("/auth/bot", srv.handleAuthLogin)
	mux.HandleFunc("/auth/callback", srv.handleAuthCallback)
	mux.HandleFunc("/api/users/me", srv.handleCurrentUser)
	mux.HandleFunc("/api/user/rank", srv.handleUserRank)
	mux.HandleFunc("/api/orders", srv.handleOrders)
	mux.HandleFunc("/api/orders/", srv.handleOrderByID)
	mux.HandleFunc("/api/market/orderbook/", srv.handleOrderBook)
	mux.HandleFunc("/api/market/trades/", srv.handleTrades)
	mux.HandleFunc("/api/market/candles/", srv.handleCandles)
	mux.HandleFunc("/api/market/ticker", srv.handleTicker)
	mux.HandleFunc("/api/news", srv.handleNews)
	mux.HandleFunc("/api/macro/indicators", srv.handleMacroIndicators)
	mux.HandleFunc("/api/fx/theoretical", srv.handleTheoreticalFXRates)
	mux.HandleFunc("/api/assets", srv.handleAssets)
	mux.HandleFunc("/api/assets/", srv.handleAssetByID)
	mux.HandleFunc("/api/bonds", srv.handleBonds)
	mux.HandleFunc("/api/bonds/", srv.handleBondOperations)
	mux.HandleFunc("/api/missions/daily", srv.handleDailyMissions)
	mux.HandleFunc("/api/missions/", srv.handleMissionByID)
	mux.HandleFunc("/api/user/missions", srv.handleUserMissions)
	mux.HandleFunc("/api/contracts", srv.handleContracts)
	mux.HandleFunc("/api/contracts/", srv.handleContractByID)
	mux.HandleFunc("/api/user/contracts", srv.handleUserContracts)
	mux.HandleFunc("/api/portfolio/balances", srv.handlePortfolioBalances)
	mux.HandleFunc("/api/portfolio/assets", srv.handlePortfolioAssets)
	mux.HandleFunc("/api/portfolio/positions", srv.handlePortfolioPositions)
	mux.HandleFunc("/api/portfolio/history", srv.handlePortfolioHistory)
	mux.HandleFunc("/api/portfolio/performance", srv.handlePortfolioPerformance)
	mux.HandleFunc("/api/pools", srv.handlePools)
	mux.HandleFunc("/api/pools/", srv.handlePoolByID)
	mux.HandleFunc("/api/pools/positions", srv.handlePoolPositions)
	mux.HandleFunc("/api/pools/positions/", srv.handlePoolPositionByID)
	mux.HandleFunc("/api/margin/pools", srv.handleMarginPools)
	mux.HandleFunc("/api/margin/pools/", srv.handleMarginPoolByID)
	mux.HandleFunc("/api/margin/positions", srv.handleMarginPositions)
	mux.HandleFunc("/api/margin/positions/", srv.handleMarginPositionByID)
	mux.HandleFunc("/api/margin/liquidations", srv.handleMarginLiquidations)
	mux.HandleFunc("/api/world/seasons/current", srv.handleCurrentSeason)
	mux.HandleFunc("/api/world/regions", srv.handleWorldRegions)
	mux.HandleFunc("/api/world/companies", srv.handleWorldCompanies)
	mux.HandleFunc("/api/world/events", srv.handleWorldEvents)
	mux.HandleFunc("/api/leaderboard", srv.handleLeaderboard)
	mux.HandleFunc("/api/companies/", srv.handleCompanyOperations)
	mux.HandleFunc("/api/indices/", srv.handleIndices)
	mux.HandleFunc("/ws", srv.handleWebSocket)
	registerFrontendRoutes(mux)
	return srv.withAPIKeyAuth(mux)
}

func registerFrontendRoutes(mux *http.ServeMux) {
	frontendDir, ok := resolveFrontendDir()
	if !ok {
		return
	}
	frontendAbs, err := filepath.Abs(frontendDir)
	if err != nil {
		return
	}

	fs := http.FileServer(http.Dir(frontendAbs))
	mux.Handle("/", fs)
}

func resolveFrontendDir() (string, bool) {
	if dir, ok := findFrontendFrom("."); ok {
		return dir, true
	}
	if wd, err := os.Getwd(); err == nil {
		if dir, ok := findFrontendFrom(wd); ok {
			return dir, true
		}
	}
	if exePath, err := os.Executable(); err == nil {
		if dir, ok := findFrontendFrom(filepath.Dir(exePath)); ok {
			return dir, true
		}
	}
	return "", false
}

func findFrontendFrom(base string) (string, bool) {
	current := base
	for i := 0; i < maxFrontendSearchDepth; i++ {
		candidate := filepath.Join(current, "frontend")
		indexPath := filepath.Join(candidate, "index.html")
		if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", false
}
