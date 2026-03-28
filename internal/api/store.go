package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	defaultCurrency    = "USD"
	defaultCashBalance = int64(1_000_000_000)
	defaultAssetPrice  = int64(10_000)
	initialUserID      = int64(9_999)
)

func stringsEqualFold(a, b string) bool {
	return strings.TrimSpace(strings.ToUpper(a)) == strings.TrimSpace(strings.ToUpper(b))
}

func stringsOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func cloneOrder(order *engine.Order) *engine.Order {
	if order == nil {
		return nil
	}
	copy := *order
	return &copy
}

type NewsItem struct {
	ID          int64  `json:"id"`
	Headline    string `json:"headline"`
	Body        string `json:"body,omitempty"`
	Impact      string `json:"impact,omitempty"`
	AssetID     int64  `json:"asset_id,omitempty"`
	PublishedAt int64  `json:"published_at"`
}

type Candle struct {
	Timestamp int64 `json:"timestamp"`
	Open      int64 `json:"open"`
	High      int64 `json:"high"`
	Low       int64 `json:"low"`
	Close     int64 `json:"close"`
	Volume    int64 `json:"volume"`
}

type Ticker struct {
	AssetID int64  `json:"asset_id"`
	Symbol  string `json:"symbol"`
	Price   int64  `json:"price"`
	Change  int64  `json:"change"`
	Volume  int64  `json:"volume"`
}

type PortfolioAsset struct {
	Asset    models.Asset `json:"asset"`
	Quantity int64        `json:"quantity"`
}

type PerformancePoint struct {
	Timestamp int64 `json:"timestamp"`
	Equity    int64 `json:"equity"`
	Cash      int64 `json:"cash"`
}

type MacroIndicator struct {
	Country     string `json:"country"`
	Type        string `json:"type"`
	Value       int64  `json:"value"`
	PublishedAt int64  `json:"published_at"`
}

type Season struct {
	Name    string `json:"name"`
	Theme   string `json:"theme"`
	StartAt int64  `json:"start_at"`
	EndAt   int64  `json:"end_at"`
}

type Region struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Company struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Sector string `json:"sector"`
}

type WorldEvent struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	StartsAt    int64  `json:"starts_at"`
	EndsAt      int64  `json:"ends_at"`
}

type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Equity   int64  `json:"equity"`
}

type OrderFilter struct {
	UserID  int64
	Status  engine.OrderStatus
	AssetID int64
}

type AssetFilter struct {
	Type   string
	Sector string
}

type MarketStore struct {
	mu              sync.RWMutex
	assets          map[int64]models.Asset
	basePrices      map[int64]int64
	users           map[int64]models.User
	orders          map[int64]*engine.Order
	executions      []engine.Execution
	balances        map[int64]map[string]int64
	positions       map[int64]map[int64]int64
	apiKeyToUser    map[string]int64
	lastPrices      map[int64]int64
	prevPrices      map[int64]int64
	volumes         map[int64]int64
	nextUserID      int64
	nextExecutionID int64
	nextNewsID      int64
	news            []NewsItem
	macroIndicators []MacroIndicator
	seasons         []Season
	regions         []Region
	worldEvents     []WorldEvent
}

func NewMarketStore() *MarketStore {
	now := time.Now().UTC()
	store := &MarketStore{
		assets:       make(map[int64]models.Asset),
		basePrices:   make(map[int64]int64),
		users:        make(map[int64]models.User),
		orders:       make(map[int64]*engine.Order),
		balances:     make(map[int64]map[string]int64),
		positions:    make(map[int64]map[int64]int64),
		apiKeyToUser: make(map[string]int64),
		lastPrices:   make(map[int64]int64),
		prevPrices:   make(map[int64]int64),
		volumes:      make(map[int64]int64),
		nextUserID:   initialUserID,
		nextNewsID:   0,
		macroIndicators: []MacroIndicator{
			{Country: "Neo Venice", Type: "GDP_GROWTH", Value: 312, PublishedAt: now.Add(-24 * time.Hour).UnixMilli()},
			{Country: "Arcadia", Type: "CPI", Value: 215, PublishedAt: now.Add(-12 * time.Hour).UnixMilli()},
			{Country: "Atlas Republic", Type: "INTEREST_RATE", Value: 175, PublishedAt: now.Add(-6 * time.Hour).UnixMilli()},
		},
		seasons: []Season{
			{Name: "Season 1: The Great Resurgence", Theme: "RECOVERY", StartAt: now.Add(-7 * 24 * time.Hour).UnixMilli(), EndAt: now.Add(53 * 24 * time.Hour).UnixMilli()},
		},
		regions: []Region{
			{ID: 1, Name: "Eurasia", Description: "Manufacturing and tech corridor"},
			{ID: 2, Name: "Aurora Belt", Description: "Energy and commodity frontier"},
		},
		worldEvents: []WorldEvent{
			{ID: 1, Name: "Central Bank Briefing", Description: "Liquidity outlook update", StartsAt: now.Add(2 * time.Hour).UnixMilli(), EndsAt: now.Add(3 * time.Hour).UnixMilli()},
		},
	}
	store.seedAssets()
	store.seedNews(now)
	return store
}

func (s *MarketStore) EnqueueOrder(order *engine.Order) {
	if order == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureUserLocked(order.UserID)
	s.ensureAssetLocked(order.AssetID)
	s.orders[order.ID] = cloneOrder(order)
}

func (s *MarketStore) EnqueueExecution(execution engine.Execution) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execution.ID == 0 {
		s.nextExecutionID++
		execution.ID = s.nextExecutionID
	}
	s.executions = append(s.executions, execution)
	if last := s.lastPrices[execution.AssetID]; last != 0 {
		s.prevPrices[execution.AssetID] = last
	}
	s.lastPrices[execution.AssetID] = execution.Price
	s.volumes[execution.AssetID] += execution.Quantity
	s.applyExecutionLocked(execution)
}

func (s *MarketStore) Shutdown(ctx context.Context) error {
	return nil
}

func (s *MarketStore) RegisterAPIKey(key string, userID int64) {
	if key == "" || userID == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureUserLocked(userID)
	s.apiKeyToUser[key] = userID
}

func (s *MarketStore) UnregisterAPIKey(key string) {
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.apiKeyToUser, key)
}

func (s *MarketStore) UserForAPIKey(key string) (models.User, bool) {
	if key == "" {
		return models.User{}, false
	}
	s.mu.RLock()
	userID, ok := s.apiKeyToUser[key]
	user := s.users[userID]
	s.mu.RUnlock()
	if !ok || userID == 0 {
		return models.User{}, false
	}
	return user, true
}

func (s *MarketStore) AddUser(username string) models.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextUserID++
	user := models.User{
		ID:       s.nextUserID,
		Username: stringsOrDefault(username, fmt.Sprintf("user-%d", s.nextUserID)),
		Role:     "player",
	}
	s.users[user.ID] = user
	s.balances[user.ID] = map[string]int64{defaultCurrency: defaultCashBalance}
	s.positions[user.ID] = make(map[int64]int64)
	return user
}

func (s *MarketStore) EnsureUser(userID int64) models.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensureUserLocked(userID)
}

func (s *MarketStore) User(userID int64) (models.User, bool) {
	if userID == 0 {
		return models.User{}, false
	}
	s.mu.Lock()
	user := s.ensureUserLocked(userID)
	s.mu.Unlock()
	if user.ID == 0 {
		return models.User{}, false
	}
	return user, true
}

func (s *MarketStore) Assets(filter AssetFilter) []models.Asset {
	s.mu.RLock()
	assets := make([]models.Asset, 0, len(s.assets))
	for _, asset := range s.assets {
		if filter.Type != "" && !stringsEqualFold(asset.Type, filter.Type) {
			continue
		}
		if filter.Sector != "" && !stringsEqualFold(asset.Sector, filter.Sector) {
			continue
		}
		assets = append(assets, asset)
	}
	s.mu.RUnlock()
	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	return assets
}

func (s *MarketStore) Asset(assetID int64) (models.Asset, bool) {
	s.mu.RLock()
	asset, ok := s.assets[assetID]
	s.mu.RUnlock()
	return asset, ok
}

func (s *MarketStore) Orders(filter OrderFilter) []engine.Order {
	s.mu.RLock()
	orders := make([]engine.Order, 0, len(s.orders))
	for _, order := range s.orders {
		if filter.UserID != 0 && order.UserID != filter.UserID {
			continue
		}
		if filter.AssetID != 0 && order.AssetID != filter.AssetID {
			continue
		}
		if filter.Status != "" && order.Status != filter.Status {
			continue
		}
		orders = append(orders, *cloneOrder(order))
	}
	s.mu.RUnlock()
	sort.Slice(orders, func(i, j int) bool { return orders[i].UpdatedAt.After(orders[j].UpdatedAt) })
	return orders
}

func (s *MarketStore) Order(orderID int64) (*engine.Order, bool) {
	if orderID == 0 {
		return nil, false
	}
	s.mu.RLock()
	order, ok := s.orders[orderID]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return cloneOrder(order), true
}

func (s *MarketStore) Executions(assetID int64, limit int) []engine.Execution {
	s.mu.RLock()
	execs := make([]engine.Execution, 0, len(s.executions))
	for _, exec := range s.executions {
		if assetID != 0 && exec.AssetID != assetID {
			continue
		}
		execs = append(execs, exec)
	}
	s.mu.RUnlock()
	sort.Slice(execs, func(i, j int) bool { return execs[i].OccurredAtUTC.After(execs[j].OccurredAtUTC) })
	if limit > 0 && len(execs) > limit {
		execs = execs[:limit]
	}
	return execs
}

func (s *MarketStore) Balances(userID int64) []models.Balance {
	if userID == 0 {
		return []models.Balance{}
	}
	s.mu.Lock()
	s.ensureUserLocked(userID)
	balances := make([]models.Balance, 0, len(s.balances[userID]))
	for currency, amount := range s.balances[userID] {
		balances = append(balances, models.Balance{Currency: currency, Amount: amount})
	}
	s.mu.Unlock()
	sort.Slice(balances, func(i, j int) bool { return balances[i].Currency < balances[j].Currency })
	return balances
}

func (s *MarketStore) Positions(userID int64) []models.Position {
	if userID == 0 {
		return []models.Position{}
	}
	s.mu.Lock()
	s.ensureUserLocked(userID)
	positions := make([]models.Position, 0, len(s.positions[userID]))
	for assetID, qty := range s.positions[userID] {
		if qty == 0 {
			continue
		}
		positions = append(positions, models.Position{AssetID: assetID, Quantity: qty})
	}
	s.mu.Unlock()
	sort.Slice(positions, func(i, j int) bool { return positions[i].AssetID < positions[j].AssetID })
	return positions
}

func (s *MarketStore) PortfolioAssets(userID int64) []PortfolioAsset {
	if userID == 0 {
		return []PortfolioAsset{}
	}
	s.mu.Lock()
	s.ensureUserLocked(userID)
	assets := make([]PortfolioAsset, 0, len(s.positions[userID]))
	for assetID, qty := range s.positions[userID] {
		if qty == 0 {
			continue
		}
		asset := s.ensureAssetLocked(assetID)
		assets = append(assets, PortfolioAsset{Asset: asset, Quantity: qty})
	}
	s.mu.Unlock()
	sort.Slice(assets, func(i, j int) bool { return assets[i].Asset.ID < assets[j].Asset.ID })
	return assets
}

func (s *MarketStore) TradeHistory(userID int64, limit int) []engine.Execution {
	if userID == 0 {
		return []engine.Execution{}
	}
	s.mu.RLock()
	execs := make([]engine.Execution, 0)
	for _, exec := range s.executions {
		if exec.TakerUserID == userID || exec.MakerUserID == userID {
			execs = append(execs, exec)
		}
	}
	s.mu.RUnlock()
	sort.Slice(execs, func(i, j int) bool { return execs[i].OccurredAtUTC.After(execs[j].OccurredAtUTC) })
	if limit > 0 && len(execs) > limit {
		execs = execs[:limit]
	}
	return execs
}

func (s *MarketStore) Performance(userID int64) []PerformancePoint {
	if userID == 0 {
		return []PerformancePoint{}
	}
	s.mu.Lock()
	s.ensureUserLocked(userID)
	cash, equity := s.evaluatePortfolioLocked(userID)
	s.mu.Unlock()
	now := time.Now().UTC().UnixMilli()
	return []PerformancePoint{{Timestamp: now, Equity: equity, Cash: cash}}
}

func (s *MarketStore) News(limit int) []NewsItem {
	s.mu.RLock()
	news := make([]NewsItem, len(s.news))
	copy(news, s.news)
	s.mu.RUnlock()
	sort.Slice(news, func(i, j int) bool { return news[i].PublishedAt > news[j].PublishedAt })
	if limit > 0 && len(news) > limit {
		news = news[:limit]
	}
	return news
}

func (s *MarketStore) Tickers() []Ticker {
	s.mu.RLock()
	tickers := make([]Ticker, 0, len(s.assets))
	for _, asset := range s.assets {
		lastPrice := s.lastPrices[asset.ID]
		basePrice := s.basePrices[asset.ID]
		price := lastPrice
		if price == 0 {
			price = basePrice
		}
		change := s.lastPriceChange(asset.ID)
		volume := s.volumeForAsset(asset.ID)
		tickers = append(tickers, Ticker{
			AssetID: asset.ID,
			Symbol:  asset.Symbol,
			Price:   price,
			Change:  change,
			Volume:  volume,
		})
	}
	s.mu.RUnlock()
	sort.Slice(tickers, func(i, j int) bool { return tickers[i].AssetID < tickers[j].AssetID })
	return tickers
}

func (s *MarketStore) Candles(assetID int64, timeframe time.Duration, start, end time.Time, limit int) []Candle {
	if assetID == 0 || timeframe <= 0 {
		return []Candle{}
	}
	s.mu.RLock()
	execs := make([]engine.Execution, 0)
	for _, exec := range s.executions {
		if exec.AssetID != assetID {
			continue
		}
		execs = append(execs, exec)
	}
	s.mu.RUnlock()
	if len(execs) == 0 {
		return []Candle{}
	}
	buckets := make(map[time.Time]*Candle)
	for _, exec := range execs {
		timestamp := exec.OccurredAtUTC.Truncate(timeframe)
		if !start.IsZero() && timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && !timestamp.Before(end) {
			continue
		}
		candle := buckets[timestamp]
		if candle == nil {
			buckets[timestamp] = &Candle{
				Timestamp: timestamp.UnixMilli(),
				Open:      exec.Price,
				High:      exec.Price,
				Low:       exec.Price,
				Close:     exec.Price,
				Volume:    exec.Quantity,
			}
			continue
		}
		if exec.Price > candle.High {
			candle.High = exec.Price
		}
		if exec.Price < candle.Low {
			candle.Low = exec.Price
		}
		candle.Close = exec.Price
		candle.Volume += exec.Quantity
	}
	times := make([]time.Time, 0, len(buckets))
	for ts := range buckets {
		times = append(times, ts)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	candles := make([]Candle, 0, len(times))
	for _, ts := range times {
		candles = append(candles, *buckets[ts])
	}
	if limit > 0 && len(candles) > limit {
		candles = candles[len(candles)-limit:]
	}
	return candles
}

func (s *MarketStore) MacroIndicators() []MacroIndicator {
	s.mu.RLock()
	indicators := make([]MacroIndicator, len(s.macroIndicators))
	copy(indicators, s.macroIndicators)
	s.mu.RUnlock()
	return indicators
}

func (s *MarketStore) Seasons() []Season {
	s.mu.RLock()
	seasons := make([]Season, len(s.seasons))
	copy(seasons, s.seasons)
	s.mu.RUnlock()
	return seasons
}

func (s *MarketStore) Regions() []Region {
	s.mu.RLock()
	regions := make([]Region, len(s.regions))
	copy(regions, s.regions)
	s.mu.RUnlock()
	return regions
}

func (s *MarketStore) Companies() []Company {
	s.mu.RLock()
	companies := make([]Company, 0, len(s.assets))
	for _, asset := range s.assets {
		companies = append(companies, Company{
			ID:     asset.ID,
			Name:   asset.Name,
			Symbol: asset.Symbol,
			Sector: asset.Sector,
		})
	}
	s.mu.RUnlock()
	sort.Slice(companies, func(i, j int) bool { return companies[i].ID < companies[j].ID })
	return companies
}

func (s *MarketStore) WorldEvents() []WorldEvent {
	s.mu.RLock()
	events := make([]WorldEvent, len(s.worldEvents))
	copy(events, s.worldEvents)
	s.mu.RUnlock()
	return events
}

func (s *MarketStore) Leaderboard(limit int) []LeaderboardEntry {
	s.mu.RLock()
	entries := make([]LeaderboardEntry, 0, len(s.users))
	for userID, user := range s.users {
		cash, equity := s.evaluatePortfolioLocked(userID)
		entries = append(entries, LeaderboardEntry{
			UserID:   userID,
			Username: user.Username,
			Equity:   cash + equity,
		})
	}
	s.mu.RUnlock()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Equity > entries[j].Equity })
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries
}

func (s *MarketStore) seedAssets() {
	seeds := []struct {
		asset     models.Asset
		basePrice int64
	}{
		{asset: models.Asset{ID: 101, Symbol: "OMNI", Name: "Omni Dynamics", Type: "STOCK", Sector: "TECH"}, basePrice: 15250},
		{asset: models.Asset{ID: 102, Symbol: "NYX", Name: "Nyx Energy", Type: "STOCK", Sector: "ENERGY"}, basePrice: 9825},
		{asset: models.Asset{ID: 103, Symbol: "AUR", Name: "Aurora Metals", Type: "COMMODITY", Sector: "METAL"}, basePrice: 18750},
	}
	for _, seed := range seeds {
		s.assets[seed.asset.ID] = seed.asset
		s.basePrices[seed.asset.ID] = seed.basePrice
	}
}

func (s *MarketStore) seedNews(now time.Time) {
	headlines := []NewsItem{
		{Headline: "Omni Dynamics announces a breakthrough in quantum logistics.", Impact: "POSITIVE", AssetID: 101},
		{Headline: "Nyx Energy faces supply chain disruption in the Aurora Belt.", Impact: "NEGATIVE", AssetID: 102},
		{Headline: "Aurora Metals signs a new long-term export agreement.", Impact: "POSITIVE", AssetID: 103},
	}
	for _, item := range headlines {
		s.nextNewsID++
		item.ID = s.nextNewsID
		item.PublishedAt = now.Add(-time.Duration(s.nextNewsID) * time.Hour).UnixMilli()
		s.news = append(s.news, item)
	}
}

func (s *MarketStore) ensureUserLocked(userID int64) models.User {
	if userID == 0 {
		return models.User{}
	}
	user, ok := s.users[userID]
	if !ok {
		user = models.User{ID: userID, Username: fmt.Sprintf("user-%d", userID), Role: "player"}
		s.users[userID] = user
	}
	if _, ok := s.balances[userID]; !ok {
		s.balances[userID] = map[string]int64{defaultCurrency: defaultCashBalance}
	}
	if _, ok := s.positions[userID]; !ok {
		s.positions[userID] = make(map[int64]int64)
	}
	return user
}

func (s *MarketStore) ensureAssetLocked(assetID int64) models.Asset {
	if assetID == 0 {
		return models.Asset{}
	}
	asset, ok := s.assets[assetID]
	if ok {
		return asset
	}
	asset = models.Asset{
		ID:     assetID,
		Symbol: fmt.Sprintf("ASSET-%d", assetID),
		Name:   fmt.Sprintf("Asset %d", assetID),
		Type:   "STOCK",
		Sector: "GENERAL",
	}
	s.assets[assetID] = asset
	if _, ok := s.basePrices[assetID]; !ok {
		s.basePrices[assetID] = defaultAssetPrice
	}
	return asset
}

func (s *MarketStore) applyExecutionLocked(exec engine.Execution) {
	taker := s.orders[exec.TakerOrderID]
	maker := s.orders[exec.MakerOrderID]
	if taker == nil || maker == nil {
		return
	}
	var buyerID, sellerID int64
	if taker.Side == engine.SideBuy {
		buyerID = taker.UserID
		sellerID = maker.UserID
	} else {
		buyerID = maker.UserID
		sellerID = taker.UserID
	}
	if buyerID == 0 || sellerID == 0 {
		return
	}
	s.ensureUserLocked(buyerID)
	s.ensureUserLocked(sellerID)
	s.ensureAssetLocked(exec.AssetID)
	cashDelta := exec.Price * exec.Quantity
	if s.balances[buyerID][defaultCurrency] < cashDelta {
		return
	}
	s.balances[buyerID][defaultCurrency] -= cashDelta
	s.balances[sellerID][defaultCurrency] += cashDelta
	s.positions[buyerID][exec.AssetID] += exec.Quantity
	s.positions[sellerID][exec.AssetID] -= exec.Quantity
}

func (s *MarketStore) lastPriceChange(assetID int64) int64 {
	lastPrice := s.lastPrices[assetID]
	prevPrice := s.prevPrices[assetID]
	if prevPrice == 0 {
		return 0
	}
	return lastPrice - prevPrice
}

func (s *MarketStore) volumeForAsset(assetID int64) int64 {
	return s.volumes[assetID]
}

func (s *MarketStore) evaluatePortfolioLocked(userID int64) (cash int64, equity int64) {
	cash = s.balances[userID][defaultCurrency]
	for assetID, qty := range s.positions[userID] {
		if qty == 0 {
			continue
		}
		price := s.lastPrices[assetID]
		if price == 0 {
			price = s.basePrices[assetID]
		}
		equity += price * qty
	}
	return cash, equity
}
