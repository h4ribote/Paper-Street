package api

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/h4ribote/Paper-Street/internal/db"
	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	defaultCurrency    = "ARC"
	defaultCashBalance = int64(10_000)
	defaultAltBalance  = int64(0) // starter balance for non-default currencies
	defaultAssetPrice  = int64(10_000)
	dbOperationTimeout = 2 * time.Second
	userIDSeed         = int64(9_999)
	minSellerProceeds  = int64(1)
	macroQuarterPeriod = 14 * 24 * time.Hour
	macroWeekPeriod    = 7 * 24 * time.Hour
	macroCycleQuarters = int64(8)
	macroTypeGDPGrowth = "GDP_GROWTH"
	macroTypeCPI       = "CPI"
	macroTypeUnemp     = "UNEMPLOYMENT"
	macroTypeInterest  = "INTEREST_RATE"
	macroTypeCCI       = "CONSUMER_CONFIDENCE"
	fxTheoreticalBase  = 1.0
	fxTheoreticalGDP   = 0.2
	fxTheoreticalRate  = 10.0
	fxTheoreticalCPI   = 5.0
	fxTheoreticalScale = int64(100)
	fxArcadiaCountry   = "Arcadia"
	newsSentimentScale = 100.0
	defaultNewsSource  = "Paper Street Wire"
)

func stringsEqualFold(a, b string) bool {
	return strings.TrimSpace(strings.ToUpper(a)) == strings.TrimSpace(strings.ToUpper(b))
}

func stringOrDefault(value, fallback string) string {
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

func safeMultiplyInt64(a, b int64) (int64, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	if a == 0 || b == 0 {
		return 0, true
	}
	if a > math.MaxInt64/b {
		return 0, false
	}
	return a * b, true
}

func safeAddInt64(a, b int64) (int64, bool) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, false
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, false
	}
	return a + b, true
}

func newsSentimentToScore(sentiment float64) int64 {
	return int64(math.Round(sentiment * newsSentimentScale))
}

func newsSentimentFromScore(score int64) float64 {
	if newsSentimentScale == 0 {
		return 0
	}
	return float64(score) / newsSentimentScale
}

func encodeImpactScope(scope []string) string {
	if len(scope) == 0 {
		return ""
	}
	encoded, err := json.Marshal(scope)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func decodeImpactScope(encoded string) []string {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var scope []string
	if err := json.Unmarshal([]byte(encoded), &scope); err != nil {
		return nil
	}
	return scope
}

type NewsItem struct {
	ID          int64    `json:"id"`
	Headline    string   `json:"headline"`
	Body        string   `json:"body,omitempty"`
	Impact      string   `json:"impact,omitempty"`
	AssetID     int64    `json:"asset_id,omitempty"`
	Category    string   `json:"category,omitempty"`
	Sentiment   float64  `json:"sentiment,omitempty"`
	ImpactScope []string `json:"impact_scope,omitempty"`
	PublishedAt int64    `json:"published_at"`
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

type TheoreticalFXRate struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Rate          int64  `json:"rate"`
	UpdatedAt     int64  `json:"updated_at"`
}

type macroCPIWeights struct {
	Food     float64
	Energy   float64
	Goods    float64
	Services float64
}

type macroProfile struct {
	Country             string
	CPIWeights          macroCPIWeights
	BaseGDP             float64
	PotentialGDP        float64
	BaseCPI             float64
	InflationTarget     float64
	NaturalUnemployment float64
	OkunBeta            float64
	RealRate            float64
	GDPAmplitude        float64
	CPIAmplitude        float64
	GDPSectorWeight     float64
	MarketSensitivity   float64
	PolicyBias          float64
	CCIBase             float64
	CCIAmplitude        float64
	SectorFocus         string
	SeasonalBoost       float64
	SeasonalQuarters    []int
}

type macroPriceIndex struct {
	Food     float64
	Energy   float64
	Goods    float64
	Services float64
	Overall  float64
}

var macroProfiles = []macroProfile{
	{
		Country:             "Arcadia",
		CPIWeights:          macroCPIWeights{Food: 0.10, Energy: 0.20, Goods: 0.30, Services: 0.40},
		BaseGDP:             3.5,
		PotentialGDP:        3.2,
		BaseCPI:             2.0,
		InflationTarget:     2.0,
		NaturalUnemployment: 4.5,
		OkunBeta:            0.6,
		RealRate:            2.0,
		GDPAmplitude:        1.4,
		CPIAmplitude:        0.7,
		GDPSectorWeight:     0.10,
		MarketSensitivity:   0.15,
		PolicyBias:          0.1,
		CCIBase:             102.0,
		CCIAmplitude:        6.0,
		SectorFocus:         "SERVICES",
	},
	{
		Country:             "Boros Federation",
		CPIWeights:          macroCPIWeights{Food: 0.30, Energy: 0.40, Goods: 0.20, Services: 0.10},
		BaseGDP:             2.8,
		PotentialGDP:        2.6,
		BaseCPI:             2.6,
		InflationTarget:     2.0,
		NaturalUnemployment: 5.2,
		OkunBeta:            0.7,
		RealRate:            2.2,
		GDPAmplitude:        1.6,
		CPIAmplitude:        0.9,
		GDPSectorWeight:     0.12,
		MarketSensitivity:   0.10,
		PolicyBias:          0.3,
		CCIBase:             98.0,
		CCIAmplitude:        7.0,
		SectorFocus:         "ENERGY",
	},
	{
		Country:             "El Dorado",
		CPIWeights:          macroCPIWeights{Food: 0.20, Energy: 0.10, Goods: 0.50, Services: 0.20},
		BaseGDP:             4.0,
		PotentialGDP:        3.5,
		BaseCPI:             3.2,
		InflationTarget:     2.5,
		NaturalUnemployment: 6.5,
		OkunBeta:            0.8,
		RealRate:            2.0,
		GDPAmplitude:        2.0,
		CPIAmplitude:        1.2,
		GDPSectorWeight:     0.14,
		MarketSensitivity:   0.08,
		PolicyBias:          0.2,
		CCIBase:             96.0,
		CCIAmplitude:        8.0,
		SectorFocus:         "GOODS",
	},
	{
		Country:             "Neo Venice",
		CPIWeights:          macroCPIWeights{Food: 0.30, Energy: 0.20, Goods: 0.30, Services: 0.20},
		BaseGDP:             3.2,
		PotentialGDP:        3.0,
		BaseCPI:             2.1,
		InflationTarget:     2.0,
		NaturalUnemployment: 3.8,
		OkunBeta:            0.5,
		RealRate:            1.8,
		GDPAmplitude:        1.2,
		CPIAmplitude:        0.6,
		GDPSectorWeight:     0.12,
		MarketSensitivity:   0.20,
		PolicyBias:          -0.4,
		CCIBase:             104.0,
		CCIAmplitude:        6.0,
		SectorFocus:         "SERVICES",
	},
	{
		Country:             "San Verde",
		CPIWeights:          macroCPIWeights{Food: 0.20, Energy: 0.30, Goods: 0.40, Services: 0.10},
		BaseGDP:             2.6,
		PotentialGDP:        2.4,
		BaseCPI:             2.4,
		InflationTarget:     2.0,
		NaturalUnemployment: 5.5,
		OkunBeta:            0.6,
		RealRate:            2.0,
		GDPAmplitude:        1.0,
		CPIAmplitude:        0.8,
		GDPSectorWeight:     0.10,
		MarketSensitivity:   0.06,
		PolicyBias:          0.1,
		CCIBase:             97.0,
		CCIAmplitude:        5.0,
		SectorFocus:         "FOOD",
		SeasonalBoost:       0.8,
		SeasonalQuarters:    []int{2, 4},
	},
	{
		Country:             "Novaya Zemlya",
		CPIWeights:          macroCPIWeights{Food: 0.40, Energy: 0.10, Goods: 0.40, Services: 0.10},
		BaseGDP:             2.2,
		PotentialGDP:        2.1,
		BaseCPI:             2.8,
		InflationTarget:     2.2,
		NaturalUnemployment: 6.0,
		OkunBeta:            0.7,
		RealRate:            2.1,
		GDPAmplitude:        1.3,
		CPIAmplitude:        1.1,
		GDPSectorWeight:     0.12,
		MarketSensitivity:   0.05,
		PolicyBias:          0.2,
		CCIBase:             95.0,
		CCIAmplitude:        6.0,
		SectorFocus:         "ENERGY",
	},
	{
		Country:             "Pearl River Zone",
		CPIWeights:          macroCPIWeights{Food: 0.30, Energy: 0.40, Goods: 0.10, Services: 0.20},
		BaseGDP:             3.1,
		PotentialGDP:        3.0,
		BaseCPI:             2.7,
		InflationTarget:     2.3,
		NaturalUnemployment: 4.0,
		OkunBeta:            0.6,
		RealRate:            2.0,
		GDPAmplitude:        1.5,
		CPIAmplitude:        1.0,
		GDPSectorWeight:     0.13,
		MarketSensitivity:   0.12,
		PolicyBias:          0.1,
		CCIBase:             100.0,
		CCIAmplitude:        7.0,
		SectorFocus:         "SERVICES",
	},
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
	mu                   sync.RWMutex
	assets               map[int64]models.Asset
	basePrices           map[int64]int64
	users                map[int64]models.User
	orders               map[int64]*engine.Order
	executions           []engine.Execution
	balances             map[int64]map[string]int64
	positions            map[int64]map[int64]int64
	assetAcquiredAt      map[int64]map[int64]int64
	apiKeyToUser         map[string]int64
	roleToUserID         map[string]int64
	roleToAPIKey         map[string]string
	lastPrices           map[int64]int64
	prevPrices           map[int64]int64
	volumes              map[int64]int64
	currencies           map[string]struct{}
	pools                map[int64]LiquidityPool
	poolPositions        map[int64]PoolPosition
	marginPools          map[int64]MarginPool
	marginProviders      map[marginProviderKey]MarginProviderPosition
	marginPositions      map[int64]MarginPosition
	marginLiquidations   []MarginLiquidation
	indexes              map[int64]IndexDefinition
	indexHoldings        map[int64]map[int64]int64
	dailyMissions        map[string][]DailyMission
	missionProgress      map[int64]map[string]*DailyMissionProgress
	contracts            map[int64]*Contract
	contractProgress     map[int64]map[int64]int64
	companyStates        map[int64]*companyState
	companyRecipes       map[int64][]ProductionRecipe
	financialReports     map[int64][]CompanyFinancialReport
	companyDividends     map[int64][]CompanyDividendRecord
	pendingCompanyDividends map[int64][]pendingCompanyDividend
	perpetualBonds       map[int64]PerpetualBondDefinition
	bondCouponIndex      map[int64]int64
	nextUserID           int64
	nextExecutionID      int64
	nextNewsID           int64
	nextContractID       int64
	nextRecipeID         int64
	nextPoolPosID        int64
	nextMarginPosID      int64
	nextMarginPositionID int64
	nextLiquidationID    int64
	news                 []NewsItem
	macroIndicators      []MacroIndicator
	theoreticalFXRates   []TheoreticalFXRate
	macroQuarterIndex    int64
	macroWeekIndex       int64
	macroGDPPrevTotals   map[string]float64
	macroGDPTotals       map[string]float64
	macroCPIIndexPrev    map[string]float64
	macroCPIIndexCurrent map[string]float64
	macroGovSpending     map[string]int64
	macroGovQuarterIndex int64
	seasons              []Season
	regions              []Region
	worldEvents          []WorldEvent
	queries              *db.Queries
	currencyIDs          map[string]int64
	needsInitialAlloc    bool
	initialAllocDone     bool
}

// NewMarketStore builds an in-memory store. newMarketStore only errors when DB queries are supplied.
func NewMarketStore() *MarketStore {
	store, _ := newMarketStore(context.Background(), nil)
	return store
}

func NewMarketStoreWithDB(ctx context.Context, queries *db.Queries) (*MarketStore, error) {
	if queries == nil {
		return nil, fmt.Errorf("db queries required")
	}
	return newMarketStore(ctx, queries)
}

func newMarketStore(ctx context.Context, queries *db.Queries) (*MarketStore, error) {
	now := time.Now().UTC()
	store := &MarketStore{
		assets:               make(map[int64]models.Asset),
		basePrices:           make(map[int64]int64),
		users:                make(map[int64]models.User),
		orders:               make(map[int64]*engine.Order),
		balances:             make(map[int64]map[string]int64),
		positions:            make(map[int64]map[int64]int64),
		assetAcquiredAt:      make(map[int64]map[int64]int64),
		apiKeyToUser:         make(map[string]int64),
		roleToUserID:         make(map[string]int64),
		roleToAPIKey:         make(map[string]string),
		lastPrices:           make(map[int64]int64),
		prevPrices:           make(map[int64]int64),
		volumes:              make(map[int64]int64),
		currencies:           map[string]struct{}{defaultCurrency: {}},
		pools:                make(map[int64]LiquidityPool),
		poolPositions:        make(map[int64]PoolPosition),
		marginPools:          make(map[int64]MarginPool),
		marginProviders:      make(map[marginProviderKey]MarginProviderPosition),
		marginPositions:      make(map[int64]MarginPosition),
		marginLiquidations:   make([]MarginLiquidation, 0),
		indexes:              make(map[int64]IndexDefinition),
		indexHoldings:        make(map[int64]map[int64]int64),
		dailyMissions:        make(map[string][]DailyMission),
		missionProgress:      make(map[int64]map[string]*DailyMissionProgress),
		contracts:            make(map[int64]*Contract),
		contractProgress:     make(map[int64]map[int64]int64),
		companyStates:        make(map[int64]*companyState),
		companyRecipes:       make(map[int64][]ProductionRecipe),
		financialReports:     make(map[int64][]CompanyFinancialReport),
		companyDividends:     make(map[int64][]CompanyDividendRecord),
		pendingCompanyDividends: make(map[int64][]pendingCompanyDividend),
		perpetualBonds:       make(map[int64]PerpetualBondDefinition),
		bondCouponIndex:      make(map[int64]int64),
		nextUserID:           userIDSeed,
		nextNewsID:           0,
		macroIndicators:      make([]MacroIndicator, 0),
		theoreticalFXRates:   make([]TheoreticalFXRate, 0),
		macroGDPPrevTotals:   make(map[string]float64),
		macroGDPTotals:       make(map[string]float64),
		macroCPIIndexPrev:    make(map[string]float64),
		macroCPIIndexCurrent: make(map[string]float64),
		macroGovSpending:     make(map[string]int64),
		seasons: []Season{
			{Name: "Season 1: The Great Resurgence", Theme: "RECOVERY", StartAt: now.Add(-7 * 24 * time.Hour).UnixMilli(), EndAt: now.Add(53 * 24 * time.Hour).UnixMilli()},
		},
		regions: []Region{
			{ID: 1, Name: "Northern Alliance", Description: "Advanced markets with high-tech leadership and aging demographics."},
			{ID: 2, Name: "Eastern Coalition", Description: "Industrial powerhouse with state-led growth and export-driven policy."},
			{ID: 3, Name: "Southern Resource Pact", Description: "Resource-rich bloc with high commodity exposure and political risk."},
			{ID: 4, Name: "Oceanic Tech Arch", Description: "Financial hubs and tax havens fueling volatile innovation."},
		},
		worldEvents: []WorldEvent{
			{ID: 1, Name: "Central Bank Briefing", Description: "Liquidity outlook update from global policy makers.", StartsAt: now.Add(2 * time.Hour).UnixMilli(), EndsAt: now.Add(3 * time.Hour).UnixMilli()},
			{ID: 2, Name: "Tech Bubble Burst", Description: "Accounting irregularities spark a sharp selloff in Arcadian tech.", StartsAt: now.Add(-6 * time.Hour).UnixMilli(), EndsAt: now.Add(-4 * time.Hour).UnixMilli()},
			{ID: 3, Name: "Resource War", Description: "El Dorado limits rare metal exports, stoking supply shock fears.", StartsAt: now.Add(6 * time.Hour).UnixMilli(), EndsAt: now.Add(12 * time.Hour).UnixMilli()},
			{ID: 4, Name: "Digital Currency Crisis", Description: "Neo Venice exchange hack triggers liquidity freeze across crypto markets.", StartsAt: now.Add(18 * time.Hour).UnixMilli(), EndsAt: now.Add(24 * time.Hour).UnixMilli()},
			{ID: 5, Name: "Boros Election", Description: "Presidential race pivots defense spending and trade policy outlook.", StartsAt: now.Add(36 * time.Hour).UnixMilli(), EndsAt: now.Add(48 * time.Hour).UnixMilli()},
			{ID: 6, Name: "Arcadia Privacy Act", Description: "New data privacy law threatens ad-tech and analytics revenue.", StartsAt: now.Add(60 * time.Hour).UnixMilli(), EndsAt: now.Add(72 * time.Hour).UnixMilli()},
			{ID: 7, Name: "El Dorado Succession", Description: "Royal succession tensions raise civil unrest risks and currency volatility.", StartsAt: now.Add(84 * time.Hour).UnixMilli(), EndsAt: now.Add(96 * time.Hour).UnixMilli()},
		},
		queries:     queries,
		currencyIDs: make(map[string]int64),
	}
	if queries == nil {
		store.seedAssets()
		store.seedCompanies()
		store.seedProductionRecipes()
		store.seedPerpetualBonds(now)
		store.mu.Lock()
		store.refreshMacroIndicatorsLocked(now)
		store.mu.Unlock()
		store.seedPools()
		store.seedInitialAllocations()
		store.seedMarginPools()
		store.seedIndexes()
		store.seedNews(now)
		store.seedContracts(now)
		return store, nil
	}
	currencyID, err := queries.EnsureDefaultCurrency(ctx, defaultCurrency)
	if err != nil {
		return nil, err
	}
	store.currencyIDs[defaultCurrency] = currencyID
	if err := store.loadFromDB(ctx); err != nil {
		return nil, err
	}
	store.mu.Lock()
	store.refreshMacroIndicatorsLocked(now)
	store.mu.Unlock()
	store.seedPools()
	if store.needsInitialAlloc {
		store.seedInitialAllocations()
	}
	store.seedMarginPools()
	store.seedIndexes()
	store.seedNews(now)
	store.seedContracts(now)
	return store, nil
}

func (s *MarketStore) EnqueueOrder(order *engine.Order) {
	if order == nil {
		return
	}
	s.mu.Lock()
	// In-memory state is authoritative even if DB persistence fails.
	s.ensureUserLocked(order.UserID)
	s.ensureAssetLocked(order.AssetID)
	cloned := cloneOrder(order)
	s.orders[order.ID] = cloned
	user := s.users[order.UserID]
	asset := s.assets[order.AssetID]
	basePrice := s.basePrices[order.AssetID]
	// user/asset are value types; copy while holding the lock for persistence snapshots.
	userSnapshot := user
	assetSnapshot := asset
	s.mu.Unlock()
	// Persistence is best-effort; in-memory state remains authoritative during runtime.
	s.persistOrder(cloned, userSnapshot, assetSnapshot, basePrice)
}

func (s *MarketStore) EnqueueExecution(execution engine.Execution) {
	s.mu.Lock()
	takerOrder := s.orders[execution.TakerOrderID]
	makerOrder := s.orders[execution.MakerOrderID]
	if takerOrder == nil || makerOrder == nil {
		s.mu.Unlock()
		return
	}
	if !s.applyExecutionLocked(execution) {
		s.mu.Unlock()
		return
	}
	if execution.ID == 0 {
		s.nextExecutionID++
		execution.ID = s.nextExecutionID
	}
	taker := cloneOrder(takerOrder)
	maker := cloneOrder(makerOrder)
	s.executions = append(s.executions, execution)
	if last := s.lastPrices[execution.AssetID]; last != 0 {
		s.prevPrices[execution.AssetID] = last
	}
	s.lastPrices[execution.AssetID] = execution.Price
	s.volumes[execution.AssetID] += execution.Quantity
	s.checkMarginLiquidationsLocked(execution.AssetID)
	buyerID, sellerID := s.executionParties(taker, maker)
	buyerCash := s.balances[buyerID][defaultCurrency]
	sellerCash := s.balances[sellerID][defaultCurrency]
	buyerQty := s.positions[buyerID][execution.AssetID]
	sellerQty := s.positions[sellerID][execution.AssetID]
	asset := s.assets[execution.AssetID]
	basePrice := s.basePrices[execution.AssetID]
	buyerUser := s.users[buyerID]
	sellerUser := s.users[sellerID]
	s.mu.Unlock()
	// Persistence is best-effort; in-memory state remains authoritative during runtime.
	s.persistExecution(executionSnapshot{
		Execution: execution,
		Taker:     taker,
		Asset:     asset,
		BasePrice: basePrice,
		Buyer: partySnapshot{
			UserID:   buyerID,
			User:     buyerUser,
			Cash:     buyerCash,
			Quantity: buyerQty,
		},
		Seller: partySnapshot{
			UserID:   sellerID,
			User:     sellerUser,
			Cash:     sellerCash,
			Quantity: sellerQty,
		},
	})
}

func (s *MarketStore) Shutdown(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	return s.queries.Close()
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

func (s *MarketStore) APIKeyForUser(userID int64) (string, bool) {
	if userID == 0 {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, id := range s.apiKeyToUser {
		if id == userID {
			return key, true
		}
	}
	return "", false
}

func (s *MarketStore) AddUser(username string) models.User {
	s.mu.Lock()
	s.nextUserID++
	user := models.User{
		ID:       s.nextUserID,
		Username: stringOrDefault(username, fmt.Sprintf("user-%d", s.nextUserID)),
		Role:     "player",
		Rank:     defaultRankName,
	}
	s.users[user.ID] = user
	s.balances[user.ID] = map[string]int64{defaultCurrency: defaultCashBalance}
	s.positions[user.ID] = make(map[int64]int64)
	s.mu.Unlock()
	s.persistUser(user, defaultCashBalance)
	return user
}

func (s *MarketStore) EnsureUser(userID int64) models.User {
	s.mu.Lock()
	user := s.ensureUserLocked(userID)
	var cashBalance int64
	if user.ID != 0 {
		cashBalance = s.balances[user.ID][defaultCurrency]
	}
	s.mu.Unlock()
	s.persistUser(user, cashBalance)
	return user
}

func (s *MarketStore) EnsureUserWithName(userID int64, username string) models.User {
	s.mu.Lock()
	user := s.ensureUserLocked(userID)
	trimmed := strings.TrimSpace(username)
	if trimmed != "" {
		if user.Username == "" || strings.HasPrefix(user.Username, "user-") {
			user.Username = trimmed
		}
	}
	var cashBalance int64
	if user.ID != 0 {
		cashBalance = s.balances[user.ID][defaultCurrency]
	}
	s.users[user.ID] = user
	s.mu.Unlock()
	s.persistUser(user, cashBalance)
	return user
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

func (s *MarketStore) OrderForAsset(orderID int64, assetID int64) (*engine.Order, bool) {
	if orderID <= 0 || assetID <= 0 {
		return nil, false
	}
	s.mu.RLock()
	order, ok := s.orders[orderID]
	s.mu.RUnlock()
	if !ok || order.AssetID != assetID {
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
	now := time.Now().UTC()
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)
	weekIndex := macroPeriodIndex(now, macroWeekPeriod)
	s.mu.RLock()
	if s.macroQuarterIndex == quarterIndex && s.macroWeekIndex == weekIndex && len(s.macroIndicators) > 0 {
		indicators := make([]MacroIndicator, len(s.macroIndicators))
		copy(indicators, s.macroIndicators)
		s.mu.RUnlock()
		return indicators
	}
	s.mu.RUnlock()

	s.mu.Lock()
	if s.macroQuarterIndex != quarterIndex || s.macroWeekIndex != weekIndex || len(s.macroIndicators) == 0 {
		s.refreshMacroIndicatorsLocked(now)
	}
	indicators := make([]MacroIndicator, len(s.macroIndicators))
	copy(indicators, s.macroIndicators)
	s.mu.Unlock()
	return indicators
}

func (s *MarketStore) TheoreticalFXRates() []TheoreticalFXRate {
	now := time.Now().UTC()
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)
	weekIndex := macroPeriodIndex(now, macroWeekPeriod)
	s.mu.RLock()
	if s.macroQuarterIndex == quarterIndex && s.macroWeekIndex == weekIndex && len(s.theoreticalFXRates) > 0 {
		rates := make([]TheoreticalFXRate, len(s.theoreticalFXRates))
		copy(rates, s.theoreticalFXRates)
		s.mu.RUnlock()
		return rates
	}
	s.mu.RUnlock()

	s.mu.Lock()
	if s.macroQuarterIndex != quarterIndex || s.macroWeekIndex != weekIndex || len(s.theoreticalFXRates) == 0 {
		s.refreshMacroIndicatorsLocked(now)
	}
	rates := make([]TheoreticalFXRate, len(s.theoreticalFXRates))
	copy(rates, s.theoreticalFXRates)
	s.mu.Unlock()
	return rates
}

func (s *MarketStore) refreshMacroIndicatorsLocked(now time.Time) {
	s.macroIndicators = s.buildMacroIndicatorsLocked(now)
	s.refreshTheoreticalFXRatesLocked(now)
	s.refreshPerpetualBondPricingLocked(now)
	s.macroQuarterIndex = macroPeriodIndex(now, macroQuarterPeriod)
	s.macroWeekIndex = macroPeriodIndex(now, macroWeekPeriod)
}

func (s *MarketStore) refreshTheoreticalFXRatesLocked(now time.Time) {
	s.theoreticalFXRates = s.buildTheoreticalFXRatesLocked(now)
}

func (s *MarketStore) buildMacroIndicatorsLocked(now time.Time) []MacroIndicator {
	if len(macroProfiles) == 0 {
		return nil
	}
	quarterStart := macroPeriodStart(now, macroQuarterPeriod)
	weekStart := macroPeriodStart(now, macroWeekPeriod)
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)
	weekIndex := macroPeriodIndex(now, macroWeekPeriod)
	s.ensureMacroQuarterTrackingLocked(quarterIndex)
	priceIndex := s.macroPriceIndexLocked()
	economy := s.macroEconomySnapshotLocked()
	indicators := make([]MacroIndicator, 0, len(macroProfiles)*5)
	for _, profile := range macroProfiles {
		weights := normalizeMacroWeights(profile.CPIWeights)
		weightedIndex := weights.Food*priceIndex.Food + weights.Energy*priceIndex.Energy + weights.Goods*priceIndex.Goods + weights.Services*priceIndex.Services
		snapshot := economy[profile.Country]
		gdpTotal := snapshot.consumption + snapshot.investment + snapshot.government + (snapshot.exports - snapshot.imports)
		s.macroGDPTotals[profile.Country] = gdpTotal
		gdpGrowth := profile.BaseGDP
		if prev := s.macroGDPPrevTotals[profile.Country]; prev > 0 {
			gdpGrowth = (gdpTotal - prev) / prev * 100.0
		}
		gdpGrowth += macroSeasonalBoost(profile, quarterIndex)
		gdpGrowth = macroClamp(gdpGrowth, -5.0, 10.0)

		cpiIndex := weightedIndex * 100.0
		s.macroCPIIndexCurrent[profile.Country] = cpiIndex
		cpi := profile.BaseCPI
		if prevIndex := s.macroCPIIndexPrev[profile.Country]; prevIndex > 0 {
			cpi = (cpiIndex - prevIndex) / prevIndex * 100.0
		}
		cpi = macroClamp(cpi, -1.5, 12.0)

		avgUtilization := 1.0
		if snapshot.companyCount > 0 {
			avgUtilization = snapshot.utilizationSum / float64(snapshot.companyCount)
			avgUtilization = macroClamp(avgUtilization, 0.0, 1.0)
		}
		unemployment := profile.NaturalUnemployment + profile.OkunBeta*(1.0-avgUtilization)
		unemployment = macroClamp(unemployment, 2.0, 20.0)

		gdpGap := gdpGrowth - profile.PotentialGDP
		inflationGap := cpi - profile.InflationTarget
		rate := profile.RealRate + cpi + 0.5*inflationGap + 0.5*gdpGap
		rate = macroClamp(rate, 0.0, 15.0)

		cci := profile.CCIBase
		cci += profile.CCIAmplitude * macroCycleValue(weekIndex, macroCycleQuarters*2)
		cci += gdpGap*4.0 - inflationGap*2.0 - (unemployment-profile.NaturalUnemployment)*3.0
		cci += macroNoise(profile.Country, weekIndex, "cci") * 4.0
		cci = macroClamp(cci, 60.0, 140.0)

		indicators = append(indicators,
			MacroIndicator{Country: profile.Country, Type: macroTypeGDPGrowth, Value: macroPercentToBasis(gdpGrowth), PublishedAt: quarterStart.UnixMilli()},
			MacroIndicator{Country: profile.Country, Type: macroTypeCPI, Value: macroPercentToBasis(cpi), PublishedAt: quarterStart.UnixMilli()},
			MacroIndicator{Country: profile.Country, Type: macroTypeUnemp, Value: macroPercentToBasis(unemployment), PublishedAt: quarterStart.UnixMilli()},
			MacroIndicator{Country: profile.Country, Type: macroTypeInterest, Value: macroPercentToBasis(rate), PublishedAt: weekStart.UnixMilli()},
			MacroIndicator{Country: profile.Country, Type: macroTypeCCI, Value: macroIndexToBasis(cci), PublishedAt: weekStart.UnixMilli()},
		)
	}
	return indicators
}

type macroIndicatorValues struct {
	gdp   int64
	rate  int64
	cpi   int64
	unemp int64
}

type macroEconomySnapshot struct {
	consumption    float64
	investment     float64
	government     float64
	exports        float64
	imports        float64
	utilizationSum float64
	companyCount   int
}

func (s *MarketStore) ensureMacroQuarterTrackingLocked(quarterIndex int64) {
	if s.macroGDPTotals == nil {
		s.macroGDPTotals = make(map[string]float64)
	}
	if s.macroGDPPrevTotals == nil {
		s.macroGDPPrevTotals = make(map[string]float64)
	}
	if s.macroCPIIndexCurrent == nil {
		s.macroCPIIndexCurrent = make(map[string]float64)
	}
	if s.macroCPIIndexPrev == nil {
		s.macroCPIIndexPrev = make(map[string]float64)
	}
	if s.macroGovSpending == nil {
		s.macroGovSpending = make(map[string]int64)
	}
	if s.macroGovQuarterIndex == 0 {
		s.macroGovQuarterIndex = quarterIndex
	}
	if quarterIndex != s.macroQuarterIndex {
		s.macroGDPPrevTotals = s.macroGDPTotals
		s.macroGDPTotals = make(map[string]float64)
		s.macroCPIIndexPrev = s.macroCPIIndexCurrent
		s.macroCPIIndexCurrent = make(map[string]float64)
		s.macroGovSpending = make(map[string]int64)
		s.macroGovQuarterIndex = quarterIndex
		s.macroQuarterIndex = quarterIndex
	}
}

func (s *MarketStore) macroEconomySnapshotLocked() map[string]macroEconomySnapshot {
	snapshots := make(map[string]macroEconomySnapshot)
	for _, state := range s.companyStates {
		if state == nil {
			continue
		}
		country := stringOrDefault(state.Country, fxArcadiaCountry)
		snapshot := snapshots[country]
		snapshot.consumption += float64(state.LastB2CRevenue)
		snapshot.government += float64(state.LastB2GRevenue)
		snapshot.investment += float64(state.LastCapexCost) + float64(state.LastInventoryChange)
		snapshot.utilizationSum += float64(state.UtilizationRate) / float64(bpsDenominator)
		snapshot.companyCount++
		snapshots[country] = snapshot
	}
	for country, amount := range s.macroGovSpending {
		snapshot := snapshots[country]
		snapshot.government += float64(amount)
		snapshots[country] = snapshot
	}
	return snapshots
}

func (s *MarketStore) recordGovernmentSpendingLocked(country string, amount int64, now time.Time) {
	if amount <= 0 {
		return
	}
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)
	if s.macroGovSpending == nil || quarterIndex != s.macroGovQuarterIndex {
		s.macroGovSpending = make(map[string]int64)
		s.macroGovQuarterIndex = quarterIndex
	}
	country = stringOrDefault(country, fxArcadiaCountry)
	s.macroGovSpending[country] += amount
}

func (s *MarketStore) macroIndicatorValuesLocked(country string) (macroIndicatorValues, bool) {
	var values macroIndicatorValues
	hasGDP := false
	hasRate := false
	hasCPI := false
	for _, indicator := range s.macroIndicators {
		if !stringsEqualFold(indicator.Country, country) {
			continue
		}
		switch indicator.Type {
		case macroTypeGDPGrowth:
			values.gdp = indicator.Value
			hasGDP = true
		case macroTypeInterest:
			values.rate = indicator.Value
			hasRate = true
		case macroTypeCPI:
			values.cpi = indicator.Value
			hasCPI = true
		case macroTypeUnemp:
			values.unemp = indicator.Value
		}
	}
	return values, hasGDP && hasRate && hasCPI
}

func computeTheoreticalFXScore(gdpTarget, gdpArc, rateTarget, rateArc, cpiTarget, cpiArc int64) float64 {
	gdpTargetPct := float64(gdpTarget) / 100.0
	gdpArcPct := float64(gdpArc) / 100.0
	gdpFactor := 1.0
	if gdpArcPct != 0 {
		gdpFactor = gdpTargetPct / gdpArcPct
	}
	rateTargetPct := float64(rateTarget) / 10000.0
	rateArcPct := float64(rateArc) / 10000.0
	cpiTargetPct := float64(cpiTarget) / 10000.0
	cpiArcPct := float64(cpiArc) / 10000.0
	return fxTheoreticalBase * (1 + fxTheoreticalGDP*gdpFactor + fxTheoreticalRate*(rateTargetPct-rateArcPct) + fxTheoreticalCPI*(cpiArcPct-cpiTargetPct))
}

func (s *MarketStore) buildTheoreticalFXRatesLocked(now time.Time) []TheoreticalFXRate {
	if len(s.macroIndicators) == 0 {
		return nil
	}
	arcadiaIndicators, ok := s.macroIndicatorValuesLocked(fxArcadiaCountry)
	if !ok {
		return nil
	}
	rates := make([]TheoreticalFXRate, 0, len(macroProfiles))
	for _, profile := range macroProfiles {
		if stringsEqualFold(profile.Country, fxArcadiaCountry) {
			continue
		}
		indicatorValues, ok := s.macroIndicatorValuesLocked(profile.Country)
		if !ok {
			continue
		}
		currency := currencyForCountry(profile.Country, "")
		if currency == "" || stringsEqualFold(currency, fxBaseCurrency) {
			continue
		}
		score := computeTheoreticalFXScore(
			indicatorValues.gdp,
			arcadiaIndicators.gdp,
			indicatorValues.rate,
			arcadiaIndicators.rate,
			indicatorValues.cpi,
			arcadiaIndicators.cpi,
		)
		if score <= 0 {
			continue
		}
		rate := int64(math.Round(score * float64(fxTheoreticalScale)))
		rates = append(rates, TheoreticalFXRate{
			BaseCurrency:  currency,
			QuoteCurrency: fxBaseCurrency,
			Rate:          rate,
			UpdatedAt:     now.UnixMilli(),
		})
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i].BaseCurrency < rates[j].BaseCurrency })
	return rates
}

func macroPeriodIndex(now time.Time, period time.Duration) int64 {
	if period <= 0 {
		return 0
	}
	seconds := int64(period / time.Second)
	if seconds <= 0 {
		return 0
	}
	return now.Unix() / seconds
}

func macroPeriodStart(now time.Time, period time.Duration) time.Time {
	if period <= 0 {
		return now.UTC()
	}
	seconds := int64(period / time.Second)
	if seconds <= 0 {
		return now.UTC()
	}
	start := (now.Unix() / seconds) * seconds
	return time.Unix(start, 0).UTC()
}

func macroCycleValue(periodIndex int64, cycleLength int64) float64 {
	if cycleLength <= 0 {
		return 0
	}
	angle := 2 * math.Pi * float64(periodIndex%cycleLength) / float64(cycleLength)
	return math.Sin(angle)
}

func macroNoise(country string, periodIndex int64, salt string) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.ToUpper(strings.TrimSpace(country))))
	_, _ = h.Write([]byte(fmt.Sprintf(":%d:%s", periodIndex, salt)))
	sum := h.Sum64()
	return (float64(sum)/float64(^uint64(0)))*2 - 1
}

func macroClamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func macroPercentToBasis(value float64) int64 {
	return int64(math.Round(value * 100))
}

func macroIndexToBasis(value float64) int64 {
	return int64(math.Round(value * 100))
}

func macroSectorIndex(focus string, index macroPriceIndex) float64 {
	switch strings.ToUpper(strings.TrimSpace(focus)) {
	case "FOOD":
		return index.Food
	case "ENERGY":
		return index.Energy
	case "GOODS":
		return index.Goods
	case "SERVICES":
		return index.Services
	default:
		return index.Overall
	}
}

func macroSeasonalBoost(profile macroProfile, quarterIndex int64) float64 {
	if profile.SeasonalBoost == 0 || len(profile.SeasonalQuarters) == 0 {
		return 0
	}
	quarterInYear := int(quarterIndex%4) + 1
	for _, quarter := range profile.SeasonalQuarters {
		if quarter == quarterInYear {
			return profile.SeasonalBoost
		}
	}
	return 0
}

func normalizeMacroWeights(weights macroCPIWeights) macroCPIWeights {
	total := weights.Food + weights.Energy + weights.Goods + weights.Services
	if total <= 0 {
		return macroCPIWeights{Food: 0.30, Energy: 0.20, Goods: 0.30, Services: 0.20}
	}
	return macroCPIWeights{
		Food:     weights.Food / total,
		Energy:   weights.Energy / total,
		Goods:    weights.Goods / total,
		Services: weights.Services / total,
	}
}

func (s *MarketStore) macroPriceIndexLocked() macroPriceIndex {
	overall := macroIndexAccumulator{}
	food := macroIndexAccumulator{}
	energy := macroIndexAccumulator{}
	goods := macroIndexAccumulator{}
	services := macroIndexAccumulator{}
	for assetID, asset := range s.assets {
		base := s.basePrices[assetID]
		if base <= 0 {
			continue
		}
		price := s.lastPrices[assetID]
		if price <= 0 {
			price = base
		}
		overall.add(price, base)
		switch strings.ToUpper(strings.TrimSpace(asset.Sector)) {
		case "FOOD", "AGRI":
			food.add(price, base)
		case "ENERGY":
			energy.add(price, base)
		case "METAL", "CONS", "DEF", "BASIC", "INDUSTRIAL":
			goods.add(price, base)
		case "TECH", "FIN", "LOG", "BIO", "SERVICES":
			services.add(price, base)
		default:
			goods.add(price, base)
		}
	}
	overallIndex := overall.ratioOrDefault(1.0)
	return macroPriceIndex{
		Food:     food.ratioOrDefault(overallIndex),
		Energy:   energy.ratioOrDefault(overallIndex),
		Goods:    goods.ratioOrDefault(overallIndex),
		Services: services.ratioOrDefault(overallIndex),
		Overall:  overallIndex,
	}
}

type macroIndexAccumulator struct {
	sumPrice float64
	sumBase  float64
}

func (acc *macroIndexAccumulator) add(price, base int64) {
	acc.sumPrice += float64(price)
	acc.sumBase += float64(base)
}

func (acc *macroIndexAccumulator) ratioOrDefault(fallback float64) float64 {
	if acc.sumBase == 0 {
		return fallback
	}
	return acc.sumPrice / acc.sumBase
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
	companies := make([]Company, 0, len(s.companyStates))
	if len(s.companyStates) == 0 {
		for _, asset := range s.assets {
			companies = append(companies, Company{
				ID:     asset.ID,
				Name:   asset.Name,
				Symbol: asset.Symbol,
				Sector: asset.Sector,
			})
		}
	} else {
		for _, state := range s.companyStates {
			companies = append(companies, state.Company)
		}
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
	s.mu.RLock()
	hasNews := len(s.news) > 0
	s.mu.RUnlock()
	if hasNews {
		return
	}
	headlines := s.generatePatternNews(now)
	if len(headlines) == 0 {
		headlines = []NewsItem{
			{Headline: "Omni Dynamics announces a breakthrough in quantum logistics.", Impact: "POSITIVE", AssetID: 101},
			{Headline: "Nyx Energy faces supply chain disruption in the Aurora Belt.", Impact: "NEGATIVE", AssetID: 102},
			{Headline: "Aurora Metals signs a new long-term export agreement.", Impact: "POSITIVE", AssetID: 103},
		}
	}
	for idx := range headlines {
		headlines[idx].PublishedAt = now.Add(-time.Duration(idx+1) * time.Hour).UnixMilli()
		s.publishNewsItem(now, headlines[idx])
	}
}

func (s *MarketStore) ensureUserLocked(userID int64) models.User {
	if userID == 0 {
		return models.User{}
	}
	user, ok := s.users[userID]
	if !ok {
		user = models.User{ID: userID, Username: fmt.Sprintf("user-%d", userID), Role: "player", Rank: defaultRankName}
		s.users[userID] = user
	}
	if user.Rank == "" {
		user.Rank = defaultRankName
	}
	isBot := strings.EqualFold(user.Role, "bot")
	cashBalance := defaultCashBalance
	altBalance := defaultAltBalance
	if isBot {
		cashBalance = 0
		altBalance = 0
	}
	if _, ok := s.balances[userID]; !ok {
		s.balances[userID] = map[string]int64{defaultCurrency: cashBalance}
	}
	for currency := range s.currencies {
		if _, ok := s.balances[userID][currency]; !ok {
			if currency == defaultCurrency {
				s.balances[userID][currency] = cashBalance
			} else {
				s.balances[userID][currency] = altBalance
			}
		}
	}
	if _, ok := s.positions[userID]; !ok {
		s.positions[userID] = make(map[int64]int64)
	}
	s.users[userID] = user
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

func (s *MarketStore) applyExecutionLocked(exec engine.Execution) bool {
	taker := s.orders[exec.TakerOrderID]
	maker := s.orders[exec.MakerOrderID]
	if taker == nil || maker == nil {
		return false
	}
	var buyerID, sellerID int64
	buyerOrder := taker
	sellerOrder := maker
	if taker.Side == engine.SideBuy {
		buyerID = taker.UserID
		sellerID = maker.UserID
	} else {
		buyerID = maker.UserID
		sellerID = taker.UserID
		buyerOrder = maker
		sellerOrder = taker
	}
	if buyerID == 0 || sellerID == 0 {
		return false
	}
	s.ensureUserLocked(buyerID)
	s.ensureUserLocked(sellerID)
	s.ensureAssetLocked(exec.AssetID)
	oldBuyerQty := s.positions[buyerID][exec.AssetID]
	oldSellerQty := s.positions[sellerID][exec.AssetID]
	cashDelta, ok := safeMultiplyInt64(exec.Price, exec.Quantity)
	if !ok {
		return false
	}
	takerUser := s.users[taker.UserID]
	makerUser := s.users[maker.UserID]
	takerRank := resolveUserRank(takerUser)
	makerRank := resolveUserRank(makerUser)
	takerFeeBps10 := takerRank.TakerFeeBps10
	makerFeeBps10 := makerRank.MakerFeeBps10
	takerFee, err := calculateFeeBps10(cashDelta, takerFeeBps10)
	if err != nil {
		return false
	}
	makerFee, err := calculateFeeBps10(cashDelta, makerFeeBps10)
	if err != nil {
		return false
	}
	buyerFee := makerFee
	sellerFee := takerFee
	if taker.Side == engine.SideBuy {
		buyerFee = takerFee
		sellerFee = makerFee
	}
	if sellerFee >= cashDelta {
		if cashDelta <= minSellerProceeds {
			sellerFee = 0
		} else {
			sellerFee = cashDelta - minSellerProceeds
		}
	}
	if buyerOrder.Leverage > marginLeverageMax || sellerOrder.Leverage > marginLeverageMax {
		log.Printf("rejecting execution with leverage above %dx (buyer=%d seller=%d)", marginLeverageMax, buyerOrder.Leverage, sellerOrder.Leverage)
		return false
	}
	buyerLeverage := normalizeLeverage(buyerOrder.Leverage)
	sellerLeverage := normalizeLeverage(sellerOrder.Leverage)
	buyerIsMargin := buyerLeverage > 1
	sellerIsMargin := sellerLeverage > 1
	var buyerMarginUsed int64
	var sellerMarginUsed int64
	if buyerIsMargin {
		buyerMarginUsed, err = requiredMargin(cashDelta, buyerLeverage)
		if err != nil {
			return false
		}
	}
	if sellerIsMargin {
		sellerMarginUsed, err = requiredMargin(cashDelta, sellerLeverage)
		if err != nil {
			return false
		}
	}
	buyerBorrowed := int64(0)
	if buyerIsMargin && cashDelta > buyerMarginUsed {
		buyerBorrowed = cashDelta - buyerMarginUsed
	}
	sellerBorrowed := int64(0)
	if sellerIsMargin {
		sellerBorrowed = exec.Quantity
	}
	if buyerIsMargin {
		if err := s.canBorrowMarginLocked(exec.AssetID, engine.SideBuy, buyerBorrowed); err != nil {
			return false
		}
	}
	if sellerIsMargin {
		if err := s.canBorrowMarginLocked(exec.AssetID, engine.SideSell, sellerBorrowed); err != nil {
			return false
		}
	}
	if buyerIsMargin {
		requiredCash := buyerMarginUsed + buyerFee
		if s.balances[buyerID][defaultCurrency] < requiredCash {
			return false
		}
	} else {
		totalCost := cashDelta + buyerFee
		if s.balances[buyerID][defaultCurrency] < totalCost {
			return false
		}
	}
	if sellerIsMargin && s.balances[sellerID][defaultCurrency] < sellerMarginUsed {
		return false
	}
	if buyerIsMargin {
		s.balances[buyerID][defaultCurrency] -= buyerMarginUsed + buyerFee
	} else {
		s.balances[buyerID][defaultCurrency] -= cashDelta + buyerFee
	}
	if cashDelta > sellerFee {
		s.balances[sellerID][defaultCurrency] += cashDelta - sellerFee
	}
	if sellerIsMargin {
		s.balances[sellerID][defaultCurrency] -= sellerMarginUsed
	}
	if !buyerIsMargin {
		s.positions[buyerID][exec.AssetID] += exec.Quantity
	}
	if !sellerIsMargin {
		s.positions[sellerID][exec.AssetID] -= exec.Quantity
	}
	execTime := exec.OccurredAtUTC
	if execTime.IsZero() {
		execTime = time.Now().UTC()
	}
	nowMillis := execTime.UnixMilli()
	if !buyerIsMargin {
		s.updateAssetAcquiredAtLocked(buyerID, exec.AssetID, oldBuyerQty, exec.Quantity, nowMillis)
	}
	if !sellerIsMargin {
		s.updateAssetAcquiredAtLocked(sellerID, exec.AssetID, oldSellerQty, -exec.Quantity, nowMillis)
	}
	if buyerIsMargin {
		if err := s.applyMarginBorrowLocked(exec.AssetID, engine.SideBuy, buyerBorrowed); err != nil {
			return false
		}
		s.openMarginPositionLocked(buyerID, exec.AssetID, engine.SideBuy, exec.Quantity, exec.Price, buyerMarginUsed, buyerLeverage, buyerBorrowed, nowMillis)
	}
	if sellerIsMargin {
		if err := s.applyMarginBorrowLocked(exec.AssetID, engine.SideSell, sellerBorrowed); err != nil {
			return false
		}
		s.openMarginPositionLocked(sellerID, exec.AssetID, engine.SideSell, exec.Quantity, exec.Price, sellerMarginUsed, sellerLeverage, sellerBorrowed, nowMillis)
	}
	return true
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
		absQty := qty
		if absQty < 0 {
			absQty = -absQty
		}
		value, ok := safeMultiplyInt64(price, absQty)
		if !ok {
			continue
		}
		if qty < 0 {
			equity -= value
		} else {
			equity += value
		}
	}
	return cash, equity
}

func (s *MarketStore) loadFromDB(ctx context.Context) error {
	assets, err := s.queries.ListAssets(ctx)
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		s.seedAssets()
		for _, asset := range s.assets {
			basePrice := s.basePrices[asset.ID]
			if basePrice == 0 {
				basePrice = defaultAssetPrice
			}
			if err := s.queries.UpsertAsset(ctx, asset, basePrice); err != nil {
				return err
			}
		}
	} else {
		for _, snapshot := range assets {
			s.assets[snapshot.Asset.ID] = snapshot.Asset
			basePrice := snapshot.BasePrice
			if basePrice == 0 {
				basePrice = defaultAssetPrice
			}
			s.basePrices[snapshot.Asset.ID] = basePrice
		}
	}
	if err := s.loadPerpetualBondsFromDB(ctx); err != nil {
		return err
	}

	if err := s.loadCompaniesFromDB(ctx); err != nil {
		return err
	}
	if len(s.companyStates) == 0 {
		s.seedCompanies()
	}
	if err := s.loadProductionRecipesFromDB(ctx); err != nil {
		return err
	}
	if len(s.companyRecipes) == 0 {
		s.seedProductionRecipes()
	}
	if err := s.loadFinancialReportsFromDB(ctx); err != nil {
		return err
	}
	if err := s.loadCompanyDividendsFromDB(ctx); err != nil {
		return err
	}
	if err := s.loadNewsFromDB(ctx); err != nil {
		return err
	}

	users, err := s.queries.ListUsers(ctx)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		s.needsInitialAlloc = true
	}
	for _, user := range users {
		user = normalizeUser(user, user.ID)
		if user.XP == 0 {
			if rankDef, ok := rankDefinitionByName(user.Rank); ok {
				user.XP = rankDef.RequiredXP
			}
		}
		s.users[user.ID] = user
		if user.ID > s.nextUserID {
			s.nextUserID = user.ID
		}
	}
	if err := s.loadAPIKeysFromDB(ctx); err != nil {
		return err
	}

	currencyBalances, err := s.queries.ListCurrencyBalances(ctx)
	if err != nil {
		return err
	}
	for _, balance := range currencyBalances {
		if _, ok := s.balances[balance.UserID]; !ok {
			s.balances[balance.UserID] = make(map[string]int64)
		}
		s.balances[balance.UserID][balance.Currency] = balance.Amount
		s.currencies[balance.Currency] = struct{}{}
	}
	s.currencies[defaultCurrency] = struct{}{}

	assetBalances, err := s.queries.ListAssetBalances(ctx)
	if err != nil {
		return err
	}
	for _, balance := range assetBalances {
		if _, ok := s.positions[balance.UserID]; !ok {
			s.positions[balance.UserID] = make(map[int64]int64)
		}
		s.positions[balance.UserID][balance.AssetID] = balance.Quantity
	}

	orders, err := s.queries.ListOrders(ctx)
	if err != nil {
		return err
	}
	orderUsers := make(map[int64]int64)
	for _, order := range orders {
		s.orders[order.ID] = cloneOrder(order)
		orderUsers[order.ID] = order.UserID
		if order.UserID > s.nextUserID {
			s.nextUserID = order.UserID
		}
	}

	executionRecords, err := s.queries.ListExecutions(ctx)
	if err != nil {
		return err
	}
	for _, record := range executionRecords {
		execution := engine.Execution{
			ID:            record.ID,
			AssetID:       record.AssetID,
			Price:         record.Price,
			Quantity:      record.Quantity,
			OccurredAtUTC: record.ExecutedAt,
		}
		if record.IsTakerBuyer {
			execution.TakerOrderID = record.BuyOrderID
			execution.MakerOrderID = record.SellOrderID
		} else {
			execution.TakerOrderID = record.SellOrderID
			execution.MakerOrderID = record.BuyOrderID
		}
		execution.TakerUserID = orderUsers[execution.TakerOrderID]
		execution.MakerUserID = orderUsers[execution.MakerOrderID]
		if execution.ID > s.nextExecutionID {
			s.nextExecutionID = execution.ID
		}
		s.executions = append(s.executions, execution)
	}

	for userID := range s.users {
		s.ensureUserLocked(userID)
	}
	s.refreshMarketStatsLocked()
	return nil
}

func (s *MarketStore) loadNewsFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	records, err := s.queries.ListNewsFeed(ctx)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	items := make([]NewsItem, 0, len(records))
	var maxID int64
	for _, record := range records {
		item := newsRecordToItem(record)
		items = append(items, item)
		if record.ID > maxID {
			maxID = record.ID
		}
	}
	s.news = items
	if maxID > s.nextNewsID {
		s.nextNewsID = maxID
	}
	return nil
}

func (s *MarketStore) loadAPIKeysFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	records, err := s.queries.ListAPIKeys(ctx)
	if err != nil {
		return err
	}
	for _, record := range records {
		key := strings.TrimSpace(record.Key)
		if key == "" || record.UserID == 0 {
			continue
		}
		s.apiKeyToUser[key] = record.UserID
		role := normalizeRole(record.Role)
		if role != "" && !isDiscordRole(role) {
			s.roleToUserID[role] = record.UserID
			s.roleToAPIKey[role] = key
		}
	}
	return nil
}

func (s *MarketStore) loadPerpetualBondsFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	records, err := s.queries.ListPerpetualBonds(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if len(records) == 0 {
		s.seedPerpetualBonds(now)
		for _, bond := range s.perpetualBonds {
			if asset, ok := s.assets[bond.AssetID]; ok {
				basePrice := s.basePrices[bond.AssetID]
				if basePrice == 0 {
					basePrice = defaultAssetPrice
				}
				if err := s.queries.UpsertAsset(ctx, asset, basePrice); err != nil {
					return err
				}
			}
			record := db.PerpetualBondRecord{
				AssetID:          bond.AssetID,
				IssuerCountry:    bond.IssuerCountry,
				BaseCoupon:       bond.BaseCoupon,
				PaymentFrequency: bond.PaymentFrequency,
			}
			if err := s.queries.UpsertPerpetualBond(ctx, record); err != nil {
				return err
			}
		}
		return nil
	}
	for _, record := range records {
		def := PerpetualBondDefinition{
			AssetID:          record.AssetID,
			IssuerCountry:    record.IssuerCountry,
			BaseCoupon:       record.BaseCoupon,
			PaymentFrequency: record.PaymentFrequency,
		}
		s.registerPerpetualBondLocked(def, now)
	}
	return nil
}

func (s *MarketStore) refreshMarketStatsLocked() {
	s.lastPrices = make(map[int64]int64)
	s.prevPrices = make(map[int64]int64)
	s.volumes = make(map[int64]int64)
	if len(s.executions) == 0 {
		return
	}
	execs := make([]engine.Execution, len(s.executions))
	copy(execs, s.executions)
	sort.Slice(execs, func(i, j int) bool { return execs[i].OccurredAtUTC.Before(execs[j].OccurredAtUTC) })
	for _, exec := range execs {
		if last, ok := s.lastPrices[exec.AssetID]; ok && last != 0 {
			s.prevPrices[exec.AssetID] = last
		}
		s.lastPrices[exec.AssetID] = exec.Price
		s.volumes[exec.AssetID] += exec.Quantity
	}
}

func (s *MarketStore) executionParties(taker *engine.Order, maker *engine.Order) (int64, int64) {
	if taker == nil || maker == nil {
		return 0, 0
	}
	if taker.Side == engine.SideBuy {
		return taker.UserID, maker.UserID
	}
	return maker.UserID, taker.UserID
}

func (s *MarketStore) dbContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), dbOperationTimeout)
}

func (s *MarketStore) persistUser(user models.User, cashBalance int64) {
	if s.queries == nil || user.ID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertUser(ctx, user, time.Now().UTC()); err != nil {
		log.Printf("db upsert user %d: %v", user.ID, err)
		return
	}
	currencyID := s.currencyIDs[defaultCurrency]
	if currencyID == 0 {
		return
	}
	if err := s.queries.SetCurrencyBalance(ctx, user.ID, currencyID, cashBalance); err != nil {
		log.Printf("db set currency balance %d: %v", user.ID, err)
	}
}

func (s *MarketStore) persistOrder(order *engine.Order, user models.User, asset models.Asset, basePrice int64) {
	if s.queries == nil || order == nil {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	user = normalizeUser(user, order.UserID)
	if err := s.queries.UpsertUser(ctx, user, time.Now().UTC()); err != nil {
		log.Printf("db upsert user %d: %v", user.ID, err)
		return
	}
	asset = normalizeAsset(asset, order.AssetID)
	basePrice = normalizeBasePrice(basePrice)
	if err := s.queries.UpsertAsset(ctx, asset, basePrice); err != nil {
		log.Printf("db upsert asset %d: %v", asset.ID, err)
		return
	}
	if err := s.queries.UpsertOrder(ctx, order); err != nil {
		log.Printf("db upsert order %d: %v", order.ID, err)
	}
}

func (s *MarketStore) persistNewsItem(item NewsItem) {
	if s.queries == nil || item.ID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	record := newsItemToRecord(item)
	if err := s.queries.UpsertNewsFeed(ctx, record); err != nil {
		log.Printf("db upsert news %d: %v", item.ID, err)
	}
}

func newsItemToRecord(item NewsItem) db.NewsRecord {
	return db.NewsRecord{
		ID:             item.ID,
		Headline:       item.Headline,
		Body:           item.Body,
		PublishedAt:    item.PublishedAt,
		Source:         defaultNewsSource,
		SentimentScore: newsSentimentToScore(item.Sentiment),
		AssetID:        item.AssetID,
		Category:       item.Category,
		Impact:         item.Impact,
		ImpactScope:    encodeImpactScope(item.ImpactScope),
	}
}

func newsRecordToItem(record db.NewsRecord) NewsItem {
	item := NewsItem{
		ID:          record.ID,
		Headline:    record.Headline,
		Body:        record.Body,
		AssetID:     record.AssetID,
		Category:    record.Category,
		Sentiment:   newsSentimentFromScore(record.SentimentScore),
		ImpactScope: decodeImpactScope(record.ImpactScope),
		PublishedAt: record.PublishedAt,
	}
	item.Impact = strings.TrimSpace(record.Impact)
	if item.Impact == "" && item.Sentiment != 0 {
		item.Impact = sentimentImpact(item.Sentiment)
	}
	return item
}

type partySnapshot struct {
	UserID   int64
	User     models.User
	Cash     int64
	Quantity int64
}

type executionSnapshot struct {
	Execution engine.Execution
	Taker     *engine.Order
	Asset     models.Asset
	BasePrice int64
	Buyer     partySnapshot
	Seller    partySnapshot
}

func (s *MarketStore) persistExecution(snapshot executionSnapshot) {
	if s.queries == nil || snapshot.Taker == nil || snapshot.Buyer.UserID == 0 || snapshot.Seller.UserID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	buyer := normalizeUser(snapshot.Buyer.User, snapshot.Buyer.UserID)
	seller := normalizeUser(snapshot.Seller.User, snapshot.Seller.UserID)
	if err := s.queries.UpsertUser(ctx, buyer, time.Now().UTC()); err != nil {
		log.Printf("db upsert user %d: %v", buyer.ID, err)
		return
	}
	if err := s.queries.UpsertUser(ctx, seller, time.Now().UTC()); err != nil {
		log.Printf("db upsert user %d: %v", seller.ID, err)
		return
	}
	asset := normalizeAsset(snapshot.Asset, snapshot.Execution.AssetID)
	basePrice := normalizeBasePrice(snapshot.BasePrice)
	if err := s.queries.UpsertAsset(ctx, asset, basePrice); err != nil {
		log.Printf("db upsert asset %d: %v", asset.ID, err)
		return
	}
	if err := s.queries.InsertExecution(ctx, snapshot.Execution, snapshot.Taker.Side); err != nil {
		log.Printf("db insert execution %d: %v", snapshot.Execution.ID, err)
		return
	}
	currencyID := s.currencyIDs[defaultCurrency]
	if currencyID != 0 {
		if err := s.queries.SetCurrencyBalance(ctx, snapshot.Buyer.UserID, currencyID, snapshot.Buyer.Cash); err != nil {
			log.Printf("db set currency balance %d: %v", snapshot.Buyer.UserID, err)
		}
		if err := s.queries.SetCurrencyBalance(ctx, snapshot.Seller.UserID, currencyID, snapshot.Seller.Cash); err != nil {
			log.Printf("db set currency balance %d: %v", snapshot.Seller.UserID, err)
		}
	}
	if err := s.queries.SetAssetBalance(ctx, snapshot.Buyer.UserID, snapshot.Execution.AssetID, snapshot.Buyer.Quantity); err != nil {
		log.Printf("db set asset balance %d: %v", snapshot.Buyer.UserID, err)
	}
	if err := s.queries.SetAssetBalance(ctx, snapshot.Seller.UserID, snapshot.Execution.AssetID, snapshot.Seller.Quantity); err != nil {
		log.Printf("db set asset balance %d: %v", snapshot.Seller.UserID, err)
	}
}

func normalizeUser(user models.User, fallbackID int64) models.User {
	if user.ID == 0 {
		user.ID = fallbackID
	}
	if user.Username == "" && user.ID != 0 {
		user.Username = fmt.Sprintf("user-%d", user.ID)
	}
	if user.Role == "" {
		user.Role = "player"
	}
	if user.Rank == "" {
		user.Rank = defaultRankName
	}
	return user
}

func normalizeAsset(asset models.Asset, fallbackID int64) models.Asset {
	if asset.ID == 0 {
		asset.ID = fallbackID
	}
	if asset.ID != 0 && asset.Symbol == "" {
		asset.Symbol = fmt.Sprintf("ASSET-%d", asset.ID)
	}
	if asset.ID != 0 && asset.Name == "" {
		asset.Name = fmt.Sprintf("Asset %d", asset.ID)
	}
	if asset.Type == "" {
		asset.Type = "STOCK"
	}
	if asset.Sector == "" {
		asset.Sector = "GENERAL"
	}
	return asset
}

func normalizeBasePrice(basePrice int64) int64 {
	if basePrice == 0 {
		return defaultAssetPrice
	}
	return basePrice
}
