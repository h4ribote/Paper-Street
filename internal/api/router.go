package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func NewRouter(e *engine.Engine, apiKeys *auth.APIKeyCache, store *MarketStore, adminPassword string) http.Handler {
	hub := newWSHub(store, e)
	srv := &Server{Engine: e, APIKeys: apiKeys, Store: store, WSHub: hub, AdminPassword: adminPassword}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/auth/login", srv.handleAuthLogin)
	mux.HandleFunc("/auth/bot", srv.handleAuthLogin)
	mux.HandleFunc("/auth/callback", srv.handleAuthCallback)
	mux.HandleFunc("/users/me", srv.handleCurrentUser)
	mux.HandleFunc("/user/rank", srv.handleUserRank)
	mux.HandleFunc("/orders", srv.handleOrders)
	mux.HandleFunc("/orders/", srv.handleOrderByID)
	mux.HandleFunc("/market/orderbook/", srv.handleOrderBook)
	mux.HandleFunc("/market/trades/", srv.handleTrades)
	mux.HandleFunc("/market/candles/", srv.handleCandles)
	mux.HandleFunc("/market/ticker", srv.handleTicker)
	mux.HandleFunc("/news", srv.handleNews)
	mux.HandleFunc("/macro/indicators", srv.handleMacroIndicators)
	mux.HandleFunc("/fx/theoretical", srv.handleTheoreticalFXRates)
	mux.HandleFunc("/assets", srv.handleAssets)
	mux.HandleFunc("/assets/", srv.handleAssetByID)
	mux.HandleFunc("/bonds", srv.handleBonds)
	mux.HandleFunc("/bonds/", srv.handleBondOperations)
	mux.HandleFunc("/missions/daily", srv.handleDailyMissions)
	mux.HandleFunc("/missions/", srv.handleMissionByID)
	mux.HandleFunc("/user/missions", srv.handleUserMissions)
	mux.HandleFunc("/contracts", srv.handleContracts)
	mux.HandleFunc("/contracts/", srv.handleContractByID)
	mux.HandleFunc("/user/contracts", srv.handleUserContracts)
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
	mux.HandleFunc("/margin/positions", srv.handleMarginPositions)
	mux.HandleFunc("/margin/positions/", srv.handleMarginPositionByID)
	mux.HandleFunc("/margin/liquidations", srv.handleMarginLiquidations)
	mux.HandleFunc("/world/seasons/current", srv.handleCurrentSeason)
	mux.HandleFunc("/world/regions", srv.handleWorldRegions)
	mux.HandleFunc("/world/companies", srv.handleWorldCompanies)
	mux.HandleFunc("/world/events", srv.handleWorldEvents)
	mux.HandleFunc("/leaderboard", srv.handleLeaderboard)
	mux.HandleFunc("/companies/", srv.handleCompanyOperations)
	mux.HandleFunc("/indices/", srv.handleIndices)
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
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, filepath.Join(frontendAbs, "index.html"))
			return
		}
		cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanPath == "." || strings.HasPrefix(cleanPath, "..") {
			http.NotFound(w, r)
			return
		}
		targetPath := filepath.Join(frontendAbs, cleanPath)
		if !isWithinDir(frontendAbs, targetPath) {
			http.NotFound(w, r)
			return
		}
		info, statErr := os.Stat(targetPath)
		if statErr != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, targetPath)
	})
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
	for i := 0; i < 8; i++ {
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

func isWithinDir(dir, target string) bool {
	dirClean := filepath.Clean(dir)
	targetClean := filepath.Clean(target)
	if dirClean == targetClean {
		return true
	}
	return strings.HasPrefix(targetClean, dirClean+string(os.PathSeparator))
}
