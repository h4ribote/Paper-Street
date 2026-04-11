package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	initialRoleUserIDStart = int64(1000)
)

type roleSeed struct {
	Role      string
	UserID    int64
	Username  string
	Balances  map[string]int64
	Positions map[int64]int64
}

type seededUser struct {
	user      models.User
	balances  map[string]int64
	positions map[int64]int64
	botRole   string
	apiKey    string
}

type companyTier struct {
	cash              int64
	inventoryQuarters int64
	sharesIssued      int64
}

func normalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func cloneCurrencyBalances(balances map[string]int64) map[string]int64 {
	if len(balances) == 0 {
		return nil
	}
	out := make(map[string]int64, len(balances))
	for currency, amount := range balances {
		out[currency] = amount
	}
	return out
}

func cloneAssetPositions(positions map[int64]int64) map[int64]int64 {
	if len(positions) == 0 {
		return nil
	}
	out := make(map[int64]int64, len(positions))
	for assetID, qty := range positions {
		out[assetID] = qty
	}
	return out
}

func (s *MarketStore) seedInitialAllocations() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.initialAllocDone {
		s.mu.Unlock()
		return
	}

	config, err := LoadEconomyConfig()
	if err != nil {
		log.Printf("failed to load economy config, using default: %v", err)
		config = defaultEconomyConfig()
	}

	s.registerInitialCurrenciesLocked()
	companies := s.sortedCompanyStatesLocked()
	seeded := make([]seededUser, 0)
	seeded = append(seeded, s.applyCompanyAllocationsLocked(companies)...)
	seeded = append(seeded, s.applyRoleSeedsLocked(companies, config)...)
	s.initialAllocDone = true
	s.mu.Unlock()
	s.persistSeededUsers(seeded)
	s.persistSeededCompanies(companies)

	if s.queries != nil {
		ctx, cancel := s.dbContext()
		defer cancel()
		if err := s.queries.SetServerStateBool(ctx, "is_initial_allocation_done", true); err != nil {
			log.Printf("failed to mark initial allocation as done in db: %v", err)
		}
	}
}

func (s *MarketStore) registerInitialCurrenciesLocked() {
	currencies := []string{"ARC", "BRB", "DRL", "VND", "VDP", "ZMR", "RVD"}
	for _, currency := range currencies {
		if currency == "" {
			continue
		}
		s.currencies[currency] = struct{}{}
	}
}

func (s *MarketStore) sortedCompanyStatesLocked() []*companyState {
	if len(s.companyStates) == 0 {
		return nil
	}
	companies := make([]*companyState, 0, len(s.companyStates))
	for _, state := range s.companyStates {
		companies = append(companies, state)
	}
	sort.Slice(companies, func(i, j int) bool { return companies[i].Company.ID < companies[j].Company.ID })
	return companies
}

func (s *MarketStore) applyCompanyAllocationsLocked(companies []*companyState) []seededUser {
	if len(companies) == 0 {
		return nil
	}
	tiers := []companyTier{
		{cash: 5_000_000, inventoryQuarters: 2, sharesIssued: 10_000_000},
		{cash: 2_000_000, inventoryQuarters: 2, sharesIssued: 5_000_000},
		{cash: 1_000_000, inventoryQuarters: 3, sharesIssued: 1_000_000},
	}
	assignTier := func(index, total int) companyTier {
		if total <= len(tiers) {
			return tiers[index%len(tiers)]
		}
		switch {
		case index < 8:
			return tiers[0]
		case index < 16:
			return tiers[1]
		default:
			return tiers[2]
		}
	}
	seeded := make([]seededUser, 0, len(companies))
	for idx, state := range companies {
		if state == nil {
			continue
		}
		tier := assignTier(idx, len(companies))
		userID := state.UserID
		if userID == 0 {
			userID = state.Company.ID
			state.UserID = userID
		}
		user, ok := s.User(userID)
		if !ok {
			user = models.User{ID: userID, Username: stringOrDefault(state.Company.Name, "company"), Role: "bot", RankID: 1, Rank: defaultRankName}
		}
		if user.Role == "" || !strings.EqualFold(user.Role, "bot") {
			user.Role = "bot"
		}
		localCurrency := currencyForCountry(state.Country, defaultCurrency)
		if localCurrency == "" {
			localCurrency = defaultCurrency
		}
		s.currencies[localCurrency] = struct{}{}

		inventory := state.MaxProductionCapacity * tier.inventoryQuarters
		state.CurrentInventory = inventory

		state.SharesIssued = tier.sharesIssued
		state.TreasuryShares = tier.sharesIssued / 2
		state.SharesOutstanding = tier.sharesIssued - state.TreasuryShares

		seedEntry := seededUser{
			user:     user,
			balances: map[string]int64{localCurrency: tier.cash},
		}
		if state.OutputAssetID != 0 {
			seedEntry.positions = map[int64]int64{state.OutputAssetID: inventory}
		}
		seeded = append(seeded, seedEntry)
	}
	return seeded
}

func (s *MarketStore) applyRoleSeedsLocked(companies []*companyState, config *EconomyConfig) []seededUser {
	var stockAssets []int64
	sharesIssued := make(map[int64]int64)
	for _, state := range companies {
		if state == nil {
			continue
		}
		stockAssets = append(stockAssets, state.Company.ID)
		sharesIssued[state.Company.ID] = state.SharesIssued
	}
	sort.Slice(stockAssets, func(i, j int) bool { return stockAssets[i] < stockAssets[j] })
	seeded := make([]seededUser, 0)
	nextUserID := initialRoleUserIDStart
	nextID := func() int64 {
		id := nextUserID
		nextUserID++
		return id
	}
	marketMakerPositions := make(map[int64]int64)
	liquidityPositions := make(map[int64]int64)
	for _, assetID := range stockAssets {
		issued := sharesIssued[assetID]
		if issued <= 0 {
			continue
		}
		marketMakerPositions[assetID] = issued * config.MarketMakerSharePercent / 100
		liquidityPositions[assetID] = issued * config.LiquiditySharePercent / 100
	}

	// Prepare dynamic configurations based on EconomyConfig
	for role, conf := range config.Roles {
		positions := make(map[int64]int64)
		if role == "market_maker" {
			for k, v := range marketMakerPositions {
				positions[k] = v
			}
		} else if role == "liquidity_provider" {
			for k, v := range liquidityPositions {
				positions[k] = v
			}
			// ensure dynamic basic balances
			if conf.Balances == nil {
				conf.Balances = map[string]int64{"ARC": 20_000_000}
			}
			for _, currency := range []string{"BRB", "DRL", "VND", "VDP", "ZMR", "RVD"} {
				conf.Balances[currency] += config.InitialCurrencySeedValue
				conf.Balances["ARC"] += 10_000_000
			}
		}

		seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
			Role:      role,
			UserID:    nextID(),
			Username:  conf.Username,
			Balances:  conf.Balances,
			Positions: positions,
		}))
	}

	for _, state := range companies {
		if state == nil {
			continue
		}
		seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
			Role:      "corporate_ai_" + strings.ToLower(state.Company.Symbol),
			UserID:    state.UserID,
			Username:  "Corporate AI " + state.Company.Symbol,
			Positions: map[int64]int64{state.Company.ID: state.TreasuryShares},
		}))
	}
	return seeded
}

func generateDeterministicAPIKey(role string) string {
	adminSecret := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if adminSecret == "" {
		adminSecret = "fallback"
	}
	h := hmac.New(sha256.New, []byte(adminSecret))
	h.Write([]byte(role))
	hashHex := hex.EncodeToString(h.Sum(nil))
	return hashHex[:20]
}

func (s *MarketStore) applyRoleSeedLocked(seed roleSeed) seededUser {
	if s == nil {
		return seededUser{}
	}
	role := normalizeRole(seed.Role)
	if role == "" {
		return seededUser{}
	}
	userID := seed.UserID
	if userID == 0 {
		return seededUser{}
	}
	user, _ := s.User(userID)
	if user.ID == 0 {
		user = models.User{ID: userID, Username: stringOrDefault(seed.Username, role), Role: "bot", RankID: 1, Rank: defaultRankName}
	}
	if user.Username == "" {
		user.Username = stringOrDefault(seed.Username, role)
	}
	for currency := range seed.Balances {
		s.currencies[currency] = struct{}{}
	}

	s.roleToUserID[role] = userID
	if _, ok := s.roleToAPIKey[role]; !ok {
		key := generateDeterministicAPIKey(role)
		s.roleToAPIKey[role] = key
		s.apiKeyToUser[key] = userID
	}
	return seededUser{
		user:      user,
		balances:  cloneCurrencyBalances(seed.Balances),
		positions: cloneAssetPositions(seed.Positions),
		botRole:   role,
		apiKey:    s.roleToAPIKey[role],
	}
}

func (s *MarketStore) persistSeededUsers(seeded []seededUser) {
	if s == nil || s.queries == nil || len(seeded) == 0 {
		return
	}
	for _, entry := range seeded {
		user := entry.user
		if user.ID == 0 {
			continue
		}
		defaultAmount := entry.balances[defaultCurrency]
		s.persistUser(user, defaultAmount)
		for currency, amount := range entry.balances {
			s.persistCurrencyBalance(user.ID, currency, amount)
		}
		for assetID, qty := range entry.positions {
			s.persistAssetBalance(user.ID, assetID, qty)
		}
		if entry.botRole != "" && entry.apiKey != "" {
			s.persistAPIKey(entry.botRole, entry.apiKey, user.ID)
		}
	}
}

func (s *MarketStore) persistSeededCompanies(companies []*companyState) {
	if s == nil || s.queries == nil {
		return
	}
	for _, state := range companies {
		if state == nil {
			continue
		}
		s.persistCompanyState(state)
	}
}
