package api

import (
	"log"
	"sort"
	"strings"

	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	initialRoleUserIDStart   = int64(1000)
	marketMakerSharePercent  = int64(20)
	liquiditySharePercent    = int64(30)
	initialCurrencySeedValue = int64(20_000_000)
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
	s.registerInitialCurrenciesLocked()
	companies := s.sortedCompanyStatesLocked()
	seeded := make([]seededUser, 0)
	seeded = append(seeded, s.applyCompanyAllocationsLocked(companies)...)
	seeded = append(seeded, s.applyRoleSeedsLocked(companies)...)
	s.initialAllocDone = true
	s.mu.Unlock()
	s.persistSeededUsers(seeded)
	s.persistSeededCompanies(companies)
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
		user := s.users[userID]
		if user.ID == 0 {
			user = models.User{ID: userID, Username: stringOrDefault(state.Company.Name, "company"), Role: "bot", RankID: 1, Rank: defaultRankName}
		}
		if user.Role == "" || !strings.EqualFold(user.Role, "bot") {
			user.Role = "bot"
		}
		if user.RankID <= 0 {
			user.RankID = 1
			user.Rank = defaultRankName
		} else if rankDef, ok := rankDefinitionByID(user.RankID); ok {
			user.Rank = rankDef.Name
		} else {
			user.RankID = 1
			user.Rank = defaultRankName
		}
		s.users[userID] = user
		if _, ok := s.balances[userID]; !ok {
			s.balances[userID] = make(map[string]int64)
		}
		if _, ok := s.positions[userID]; !ok {
			s.positions[userID] = make(map[int64]int64)
		}
		localCurrency := currencyForCountry(state.Country, defaultCurrency)
		if localCurrency == "" {
			localCurrency = defaultCurrency
		}
		s.currencies[localCurrency] = struct{}{}
		s.balances[userID][localCurrency] = tier.cash
		inventory := state.MaxProductionCapacity * tier.inventoryQuarters
		state.CurrentInventory = inventory
		if state.OutputAssetID != 0 {
			s.positions[userID][state.OutputAssetID] = inventory
		}
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

func (s *MarketStore) applyRoleSeedsLocked(companies []*companyState) []seededUser {
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
		marketMakerPositions[assetID] = issued * marketMakerSharePercent / 100
		liquidityPositions[assetID] = issued * liquiditySharePercent / 100
	}
	seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
		Role:      "market_maker",
		UserID:    nextID(),
		Username:  "Market Maker",
		Balances:  map[string]int64{"ARC": 5_000_000},
		Positions: marketMakerPositions,
	}))
	liquidityBalances := map[string]int64{"ARC": 20_000_000}
	for _, currency := range []string{"BRB", "DRL", "VND", "VDP", "ZMR", "RVD"} {
		liquidityBalances[currency] += initialCurrencySeedValue
		liquidityBalances["ARC"] += 10_000_000
	}
	seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
		Role:      "liquidity_provider",
		UserID:    nextID(),
		Username:  "Liquidity Provider",
		Balances:  liquidityBalances,
		Positions: liquidityPositions,
	}))
	seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
		Role:     "whale_northern",
		UserID:   nextID(),
		Username: "Whale Northern",
		Balances: map[string]int64{"ARC": 5_000_000, "BRB": 5_000_000, "DRL": 5_000_000},
	}))
	seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
		Role:     "whale_oceanic",
		UserID:   nextID(),
		Username: "Whale Oceanic",
		Balances: map[string]int64{"VND": 7_500_000, "VDP": 7_500_000},
	}))
	seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
		Role:     "whale_energy",
		UserID:   nextID(),
		Username: "Whale Energy",
		Balances: map[string]int64{"ZMR": 7_500_000, "RVD": 7_500_000},
	}))
	seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
		Role:     "national_ai_arcadia",
		UserID:   nextID(),
		Username: "National AI Arcadia",
		Balances: map[string]int64{"ARC": 30_000_000},
	}))
	nationalSeeds := []struct {
		role     string
		username string
		currency string
	}{
		{role: "national_ai_boros", username: "National AI Boros", currency: "BRB"},
		{role: "national_ai_el_dorado", username: "National AI El Dorado", currency: "DRL"},
		{role: "national_ai_neo_venice", username: "National AI Neo Venice", currency: "VND"},
		{role: "national_ai_san_verde", username: "National AI San Verde", currency: "VDP"},
		{role: "national_ai_novaya_zemlya", username: "National AI Novaya", currency: "ZMR"},
		{role: "national_ai_pearl_river", username: "National AI Pearl River", currency: "RVD"},
	}
	for _, seed := range nationalSeeds {
		seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
			Role:     seed.role,
			UserID:   nextID(),
			Username: seed.username,
			Balances: map[string]int64{seed.currency: 20_000_000, "ARC": 10_000_000},
		}))
	}
	groupA := map[string]int64{"ARC": 200_000, "BRB": 200_000, "DRL": 200_000}
	groupB := map[string]int64{"VND": 300_000, "VDP": 300_000}
	groupC := map[string]int64{"ZMR": 300_000, "RVD": 300_000}
	for _, entry := range []struct {
		role string
		set  map[string]int64
	}{
		{"momentum_chaser_a", groupA},
		{"momentum_chaser_b", groupB},
		{"momentum_chaser_c", groupC},
		{"dip_buyer_a", groupA},
		{"dip_buyer_b", groupB},
		{"dip_buyer_c", groupC},
		{"reversal_sniper_a", groupA},
		{"reversal_sniper_b", groupB},
		{"reversal_sniper_c", groupC},
		{"grid_trader_a", groupA},
		{"grid_trader_b", groupB},
		{"grid_trader_c", groupC},
		{"news_reactor", groupA},
		{"arbitrageur", groupA},
		{"yield_hunter", groupA},
		{"public_consumer", groupA},
	} {
		username := strings.ReplaceAll(entry.role, "_", " ")
		seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
			Role:     entry.role,
			UserID:   nextID(),
			Username: username,
			Balances: entry.set,
		}))
	}
	for _, state := range companies {
		if state == nil {
			continue
		}
		seeded = append(seeded, s.applyRoleSeedLocked(roleSeed{
			Role:     "corporate_ai_" + strings.ToLower(state.Company.Symbol),
			UserID:   state.UserID,
			Username: "Corporate AI " + state.Company.Symbol,
		}))
	}
	return seeded
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
	user := s.users[userID]
	if user.ID == 0 {
		username := stringOrDefault(seed.Username, role)
		user = models.User{ID: userID, Username: username, Role: "bot", RankID: 1, Rank: defaultRankName}
	}
	if user.Role == "" || !strings.EqualFold(user.Role, "bot") {
		user.Role = "bot"
	}
	if user.RankID <= 0 {
		user.RankID = 1
		user.Rank = defaultRankName
	} else if rankDef, ok := rankDefinitionByID(user.RankID); ok {
		user.Rank = rankDef.Name
	} else {
		user.RankID = 1
		user.Rank = defaultRankName
	}
	if user.Username == "" {
		user.Username = stringOrDefault(seed.Username, role)
	}
	s.users[userID] = user
	if _, ok := s.balances[userID]; !ok {
		s.balances[userID] = make(map[string]int64)
	}
	if _, ok := s.positions[userID]; !ok {
		s.positions[userID] = make(map[int64]int64)
	}
	for currency, amount := range seed.Balances {
		s.currencies[currency] = struct{}{}
		s.balances[userID][currency] = amount
	}
	for assetID, qty := range seed.Positions {
		if assetID == 0 {
			continue
		}
		s.positions[userID][assetID] = qty
	}
	s.roleToUserID[role] = userID
	if _, ok := s.roleToAPIKey[role]; !ok {
		key, err := generateAPIKeyHex()
		if err != nil {
			log.Printf("failed to generate api key for role %s: %v", role, err)
		} else {
			s.roleToAPIKey[role] = key
			s.apiKeyToUser[key] = userID
			s.persistAPIKey(role, key, userID)
		}
	}
	return seededUser{
		user:      user,
		balances:  cloneCurrencyBalances(seed.Balances),
		positions: cloneAssetPositions(seed.Positions),
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
