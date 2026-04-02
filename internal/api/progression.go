package api

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	defaultRankName       = "Shrimp"
	secondsPerDay         = int64(24 * time.Hour / time.Second)
	contractAssetNYX      = int64(102)
	contractAssetAUR      = int64(103)
	contractDeadlineLong  = 72 * time.Hour
	contractDeadlineShort = 48 * time.Hour
	contractPremiumMinBps = int64(500)
	contractPremiumMaxBps = int64(1000)
	contractVolumeStep    = int64(1_000)
	contractVWAPWindow    = 24 * time.Hour
	contractCapexBaseQty  = int64(5_000)
	contractGovBaseQty    = int64(2_000)
	contractCapexXP       = int64(5)
	contractGovXP         = int64(3)
	contractCorpName      = "OmniCorp"
	contractGovName       = "Boros Federation"
	contractCapexMinRank  = "Shark"
	contractGovMinRank    = defaultRankName
)

type RankDefinition struct {
	ID                  int
	Name                string
	RequiredXP          int64
	MakerFeeBps10       int64
	TakerFeeBps10       int64
	InterestDiscountBps int64
	FXFeeDiscountBps    int64
}

var rankDefinitions = []RankDefinition{
	{ID: 1, Name: "Shrimp", RequiredXP: 0, MakerFeeBps10: 40, TakerFeeBps10: 100, InterestDiscountBps: 0, FXFeeDiscountBps: 0},
	{ID: 2, Name: "Fish", RequiredXP: 500, MakerFeeBps10: 20, TakerFeeBps10: 75, InterestDiscountBps: 500, FXFeeDiscountBps: 0},
	{ID: 3, Name: "Shark", RequiredXP: 2_500, MakerFeeBps10: 10, TakerFeeBps10: 50, InterestDiscountBps: 1_000, FXFeeDiscountBps: 0},
	{ID: 4, Name: "Whale", RequiredXP: 15_000, MakerFeeBps10: 0, TakerFeeBps10: 25, InterestDiscountBps: 2_000, FXFeeDiscountBps: 0},
	{ID: 5, Name: "Leviathan", RequiredXP: 50_000, MakerFeeBps10: 0, TakerFeeBps10: 0, InterestDiscountBps: 5_000, FXFeeDiscountBps: 5_000},
}

type UserRankInfo struct {
	UserID              int64  `json:"user_id"`
	RankID              int    `json:"rank_id"`
	Rank                string `json:"rank"`
	XP                  int64  `json:"xp"`
	NextRankXP          int64  `json:"next_rank_xp,omitempty"`
	MakerFeeBps         int64  `json:"maker_fee_bps"`
	TakerFeeBps         int64  `json:"taker_fee_bps"`
	InterestDiscountBps int64  `json:"interest_discount_bps"`
	FXFeeDiscountBps    int64  `json:"fx_fee_discount_bps"`
}

type DailyMission struct {
	ID          string `json:"id"`
	Grade       string `json:"grade"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DailyMissionStatus struct {
	Mission   DailyMission `json:"mission"`
	Completed bool         `json:"completed"`
}

type DailyMissionReward struct {
	Grade string `json:"grade"`
	XP    int64  `json:"xp"`
	Cash  int64  `json:"cash"`
}

type DailyMissionResponse struct {
	Date     string               `json:"date"`
	Missions []DailyMissionStatus `json:"missions"`
}

type MissionCompletionResult struct {
	Mission      DailyMission        `json:"mission"`
	Completed    bool                `json:"completed"`
	Reward       *DailyMissionReward `json:"reward,omitempty"`
	UserProgress UserRankInfo        `json:"user_progress"`
}

type DailyMissionProgress struct {
	Completed map[string]bool
	Rewarded  map[string]bool
}

type Contract struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	AssetID       int64  `json:"asset_id"`
	TotalRequired int64  `json:"total_required"`
	Delivered     int64  `json:"delivered"`
	PricePerUnit  int64  `json:"price_per_unit"`
	MinRank       string `json:"min_rank"`
	DeadlineAt    int64  `json:"deadline_at"`
	XPPerUnit     int64  `json:"xp_per_unit"`
}

type contractKind string

const (
	contractKindCapex       contractKind = "CAPEX"
	contractKindProcurement contractKind = "PROCUREMENT"
)

type contractTemplate struct {
	title        string
	assetID      int64
	minRank      string
	deadline     time.Duration
	baseRequired int64
	xpPerUnit    int64
}

type ContractStatus struct {
	Contract
	Remaining     int64 `json:"remaining"`
	UserDelivered int64 `json:"user_delivered"`
	Expired       bool  `json:"expired"`
	Fulfilled     bool  `json:"fulfilled"`
}

type ContractDeliveryResult struct {
	Contract     ContractStatus `json:"contract"`
	Quantity     int64          `json:"quantity"`
	CashReward   int64          `json:"cash_reward"`
	XPReward     int64          `json:"xp_reward"`
	UserProgress UserRankInfo   `json:"user_progress"`
}

func rankDefinitionByName(name string) (RankDefinition, bool) {
	trimmed := strings.TrimSpace(name)
	for _, rank := range rankDefinitions {
		if strings.EqualFold(rank.Name, trimmed) {
			return rank, true
		}
	}
	return RankDefinition{}, false
}

func rankDefinitionForXP(xp int64) RankDefinition {
	current := rankDefinitions[0]
	for _, rank := range rankDefinitions {
		if xp >= rank.RequiredXP {
			current = rank
		}
	}
	return current
}

func resolveUserRank(user models.User) RankDefinition {
	rank := rankDefinitionForXP(user.XP)
	if def, ok := rankDefinitionByName(user.Rank); ok {
		rank = def
	}
	return rank
}

func nextRankXP(rank RankDefinition) int64 {
	for _, candidate := range rankDefinitions {
		if candidate.ID == rank.ID+1 {
			return candidate.RequiredXP
		}
	}
	return 0
}

func feeBps10ToBps(bps10 int64) int64 {
	if bps10 <= 0 {
		return 0
	}
	return (bps10 + 5) / 10
}

func calculateFeeBps10(amount, bps10 int64) (int64, error) {
	const bps10Denominator = int64(100_000)
	fee, ok := safeMultiplyInt64(amount, bps10)
	if !ok {
		return 0, errors.New("fee overflow")
	}
	return fee / bps10Denominator, nil
}

func applyDiscountBps(value, discountBps int64) int64 {
	if discountBps <= 0 {
		return value
	}
	if discountBps >= bpsDenominator {
		return 0
	}
	product, ok := safeMultiplyInt64(value, bpsDenominator-discountBps)
	if !ok {
		return value
	}
	return product / bpsDenominator
}

func (s *MarketStore) fxFeeBpsForUserLocked(userID int64, baseFee int64) int64 {
	if userID == 0 {
		return baseFee
	}
	user := s.users[userID]
	rank := resolveUserRank(user)
	return applyDiscountBps(baseFee, rank.FXFeeDiscountBps)
}

func (s *MarketStore) marginRateForUserLocked(userID int64, baseRate int64) int64 {
	if userID == 0 {
		return baseRate
	}
	user := s.users[userID]
	rank := resolveUserRank(user)
	return applyDiscountBps(baseRate, rank.InterestDiscountBps)
}

func (s *MarketStore) userRankInfoLocked(userID int64) UserRankInfo {
	user := s.users[userID]
	rank := resolveUserRank(user)
	return UserRankInfo{
		UserID:              userID,
		RankID:              rank.ID,
		Rank:                rank.Name,
		XP:                  user.XP,
		NextRankXP:          nextRankXP(rank),
		MakerFeeBps:         feeBps10ToBps(rank.MakerFeeBps10),
		TakerFeeBps:         feeBps10ToBps(rank.TakerFeeBps10),
		InterestDiscountBps: rank.InterestDiscountBps,
		FXFeeDiscountBps:    rank.FXFeeDiscountBps,
	}
}

func (s *MarketStore) UserRankInfo(userID int64) (UserRankInfo, bool) {
	if userID == 0 {
		return UserRankInfo{}, false
	}
	s.mu.Lock()
	s.ensureUserLocked(userID)
	info := s.userRankInfoLocked(userID)
	s.mu.Unlock()
	return info, true
}

func (s *MarketStore) AddXP(userID, amount int64) (UserRankInfo, error) {
	if userID == 0 {
		return UserRankInfo{}, errors.New("user_id required")
	}
	if amount <= 0 {
		return UserRankInfo{}, errors.New("xp amount must be positive")
	}
	s.mu.Lock()
	user := s.ensureUserLocked(userID)
	user.XP += amount
	rank := rankDefinitionForXP(user.XP)
	user.Rank = rank.Name
	s.users[userID] = user
	info := s.userRankInfoLocked(userID)
	cash := s.balances[userID][defaultCurrency]
	s.mu.Unlock()
	s.persistUser(user, cash)
	return info, nil
}

func dailyMissionCatalog() map[string][]DailyMission {
	return map[string][]DailyMission{
		"C": {
			{ID: "C1", Grade: "C", Name: "Window Shopper", Description: "異なる3つの銘柄の詳細画面を開く / Open details for 3 different assets"},
			{ID: "C2", Grade: "C", Name: "First Bid", Description: "指値注文を1回出す / Place one limit order"},
			{ID: "C3", Grade: "C", Name: "News Reader", Description: "ニュース詳細を3件開く / Open 3 news details"},
			{ID: "C4", Grade: "C", Name: "Tourist", Description: "異なる2つの地域の銘柄を保有する / Hold assets from 2 different regions"},
			{ID: "C5", Grade: "C", Name: "Penny Pincher", Description: "株価が100 ARC未満の銘柄を購入する / Buy an asset priced under 100 ARC"},
			{ID: "C6", Grade: "C", Name: "Safety First", Description: "国債を購入する / Buy a government bond"},
		},
		"B": {
			{ID: "B1", Grade: "B", Name: "Day Trader", Description: "同一銘柄を1日以内に売却する / Sell the same asset within a day"},
			{ID: "B2", Grade: "B", Name: "Profit Taker", Description: "評価益+5%超で利確する / Take profit over +5% unrealized"},
			{ID: "B3", Grade: "B", Name: "Stop Loss", Description: "逆指値注文を発注する / Place a stop order"},
			{ID: "B4", Grade: "B", Name: "Short Seller", Description: "信用売りポジションを建てる / Open a short position"},
			{ID: "B5", Grade: "B", Name: "Sector Rotation", Description: "上昇セクターを買い下落セクターを売る / Buy top sector, sell bottom sector"},
			{ID: "B6", Grade: "B", Name: "Buy The Dip", Description: "前日比-3%以上下落銘柄を買う / Buy assets down 3%+ day-over-day"},
			{ID: "B7", Grade: "B", Name: "Yield Hunter", Description: "利回り5%以上の銘柄を保有する / Hold assets yielding 5%+"},
			{ID: "B8", Grade: "B", Name: "Tech Lover", Description: "TECHセクター比率50%以上にする / Keep 50%+ in TECH sector"},
		},
		"A": {
			{ID: "A1", Grade: "A", Name: "Market Maker", Description: "買い/売り指値を同時に5分維持する / Keep bid/ask quotes for 5 minutes"},
			{ID: "A2", Grade: "A", Name: "Liquidity King", Description: "FXプールに10,000 ARC以上供給する / Provide 10,000+ ARC to FX pool"},
			{ID: "A3", Grade: "A", Name: "Arbitrageur", Description: "指数と構成銘柄を同日に取引する / Trade index and components same day"},
			{ID: "A4", Grade: "A", Name: "Big Ticket", Description: "50,000 ARC以上の約定を成立させる / Execute 50,000+ ARC in one trade"},
			{ID: "A5", Grade: "A", Name: "News Reactor", Description: "Breakingニュース30秒以内に取引する / Trade within 30s of breaking news"},
			{ID: "A6", Grade: "A", Name: "Contrarian", Description: "センチメントと逆のポジションを建てる / Take the opposite sentiment position"},
			{ID: "A7", Grade: "A", Name: "Leverage Master", Description: "レバレッジ20倍以上で8時間維持する / Hold 20x+ leverage for 8 hours"},
			{ID: "A8", Grade: "A", Name: "The Squeeze", Description: "ショートフィー高騰銘柄でロングする / Go long on high short-fee assets"},
		},
	}
}

func missionRewardForGrade(grade string) DailyMissionReward {
	switch strings.ToUpper(grade) {
	case "C":
		return DailyMissionReward{Grade: "C", XP: 5, Cash: 400}
	case "B":
		return DailyMissionReward{Grade: "B", XP: 25}
	case "A":
		return DailyMissionReward{Grade: "A", XP: 60}
	default:
		return DailyMissionReward{Grade: grade}
	}
}

func (s *MarketStore) DailyMissions(date time.Time) []DailyMission {
	date = date.UTC()
	key := date.Format("2006-01-02")
	s.mu.RLock()
	if missions, ok := s.dailyMissions[key]; ok {
		s.mu.RUnlock()
		return missions
	}
	s.mu.RUnlock()
	catalog := dailyMissionCatalog()
	missions := make([]DailyMission, 0, 6)
	seed := date.Unix() / secondsPerDay
	if seed < 0 {
		seed = -seed
	}
	grades := []string{"C", "B", "A"}
	for _, grade := range grades {
		list := catalog[grade]
		if len(list) == 0 {
			continue
		}
		offset := 0
		if len(list) > 1 {
			offset = int(seed % int64(len(list)))
		}
		first := list[offset]
		second := list[(offset+1)%len(list)]
		missions = append(missions, first, second)
	}
	sort.Slice(missions, func(i, j int) bool { return missions[i].ID < missions[j].ID })
	s.mu.Lock()
	s.dailyMissions[key] = missions
	s.mu.Unlock()
	return missions
}

func (s *MarketStore) DailyMissionStatus(userID int64, date time.Time) ([]DailyMissionStatus, error) {
	if userID == 0 {
		return nil, errors.New("user_id required")
	}
	missions := s.DailyMissions(date)
	key := date.UTC().Format("2006-01-02")
	s.mu.Lock()
	s.ensureUserLocked(userID)
	progress := s.ensureMissionProgressLocked(userID, key)
	statuses := make([]DailyMissionStatus, 0, len(missions))
	for _, mission := range missions {
		statuses = append(statuses, DailyMissionStatus{
			Mission:   mission,
			Completed: progress.Completed[mission.ID],
		})
	}
	s.mu.Unlock()
	return statuses, nil
}

func (s *MarketStore) CompleteDailyMission(userID int64, missionID string, date time.Time) (MissionCompletionResult, error) {
	if userID == 0 {
		return MissionCompletionResult{}, errors.New("user_id required")
	}
	missionID = strings.TrimSpace(missionID)
	if missionID == "" {
		return MissionCompletionResult{}, errors.New("mission_id required")
	}
	date = date.UTC()
	missions := s.DailyMissions(date)
	var mission DailyMission
	found := false
	for _, candidate := range missions {
		if candidate.ID == missionID {
			mission = candidate
			found = true
			break
		}
	}
	if !found {
		return MissionCompletionResult{}, errors.New("mission not available today")
	}
	key := date.Format("2006-01-02")
	var reward *DailyMissionReward
	var info UserRankInfo
	s.mu.Lock()
	s.ensureUserLocked(userID)
	progress := s.ensureMissionProgressLocked(userID, key)
	if !progress.Completed[missionID] {
		progress.Completed[missionID] = true
	}
	grade := strings.ToUpper(mission.Grade)
	if grade != "" && !progress.Rewarded[grade] {
		pairCompleted := true
		for _, candidate := range missions {
			if strings.EqualFold(candidate.Grade, grade) && !progress.Completed[candidate.ID] {
				pairCompleted = false
				break
			}
		}
		if pairCompleted {
			progress.Rewarded[grade] = true
			rewardValue := missionRewardForGrade(grade)
			if rewardValue.XP > 0 {
				user := s.users[userID]
				user.XP += rewardValue.XP
				rank := rankDefinitionForXP(user.XP)
				user.Rank = rank.Name
				s.users[userID] = user
			}
			if rewardValue.Cash > 0 {
				s.balances[userID][defaultCurrency] += rewardValue.Cash
				s.recordGovernmentSpendingLocked(fxArcadiaCountry, rewardValue.Cash, date)
			}
			reward = &rewardValue
		}
	}
	info = s.userRankInfoLocked(userID)
	user := s.users[userID]
	cash := s.balances[userID][defaultCurrency]
	s.mu.Unlock()
	if reward != nil {
		s.persistUser(user, cash)
	}
	return MissionCompletionResult{
		Mission:      mission,
		Completed:    true,
		Reward:       reward,
		UserProgress: info,
	}, nil
}

func (s *MarketStore) ensureMissionProgressLocked(userID int64, dateKey string) *DailyMissionProgress {
	progressByDate, ok := s.missionProgress[userID]
	if !ok {
		progressByDate = make(map[string]*DailyMissionProgress)
		s.missionProgress[userID] = progressByDate
	}
	progress, ok := progressByDate[dateKey]
	if !ok {
		progress = &DailyMissionProgress{
			Completed: make(map[string]bool),
			Rewarded:  make(map[string]bool),
		}
		progressByDate[dateKey] = progress
	}
	return progress
}

func (s *MarketStore) refreshContractsLocked(now time.Time) {
	s.syncContractIDLocked()
	s.pruneContractsLocked(now)
	s.ensureContractKindLocked(now, contractKindCapex)
	s.ensureContractKindLocked(now, contractKindProcurement)
}

func (s *MarketStore) syncContractIDLocked() {
	if s.nextContractID != 0 {
		return
	}
	for id := range s.contracts {
		if id > s.nextContractID {
			s.nextContractID = id
		}
	}
}

func (s *MarketStore) pruneContractsLocked(now time.Time) {
	nowMillis := now.UTC().UnixMilli()
	for id, contract := range s.contracts {
		if contract == nil {
			delete(s.contracts, id)
			go s.deleteContract(id)
			continue
		}
		if contract.Delivered >= contract.TotalRequired {
			delete(s.contracts, id)
			go s.deleteContract(id)
			continue
		}
		if contract.DeadlineAt > 0 && nowMillis > contract.DeadlineAt {
			delete(s.contracts, id)
			go s.deleteContract(id)
		}
	}
}

func (s *MarketStore) ensureContractKindLocked(now time.Time, kind contractKind) {
	for _, contract := range s.contracts {
		if s.contractKindForAssetLocked(contract.AssetID) == kind {
			return
		}
	}
	contract := s.generateContractLocked(now, kind)
	if contract != nil {
		s.contracts[contract.ID] = contract
		contractSnapshot := *contract
		go s.persistContract(&contractSnapshot)
	}
}

func (s *MarketStore) contractKindForAssetLocked(assetID int64) contractKind {
	asset := s.assets[assetID]
	if stringsEqualFold(asset.Sector, "ENERGY") {
		return contractKindProcurement
	}
	if stringsEqualFold(asset.Type, "COMMODITY") {
		return contractKindCapex
	}
	return contractKindCapex
}

func (s *MarketStore) generateContractLocked(now time.Time, kind contractKind) *Contract {
	asset := s.pickContractAssetLocked(kind)
	if asset.ID == 0 {
		return nil
	}
	template := contractTemplateForKind(kind, asset)
	totalRequired := s.scaleContractRequirementLocked(asset.ID, template.baseRequired)
	pricePerUnit := s.calculateContractPriceLocked(asset.ID, kind, now)
	if pricePerUnit <= 0 {
		pricePerUnit = s.basePrices[asset.ID]
	}
	if pricePerUnit <= 0 {
		pricePerUnit = defaultAssetPrice
	}
	s.nextContractID++
	contract := &Contract{
		ID:            s.nextContractID,
		Title:         template.title,
		AssetID:       asset.ID,
		TotalRequired: totalRequired,
		Delivered:     0,
		PricePerUnit:  pricePerUnit,
		MinRank:       template.minRank,
		DeadlineAt:    now.Add(template.deadline).UnixMilli(),
		XPPerUnit:     template.xpPerUnit,
	}
	return contract
}

func (s *MarketStore) pickContractAssetLocked(kind contractKind) models.Asset {
	if kind == contractKindCapex {
		if asset, ok := s.assets[contractAssetAUR]; ok {
			return asset
		}
	}
	if kind == contractKindProcurement {
		if asset, ok := s.assets[contractAssetNYX]; ok {
			return asset
		}
	}
	assets := sortedAssets(s.assets)
	var selected models.Asset
	var bestScore int64
	for _, asset := range assets {
		switch kind {
		case contractKindCapex:
			if !stringsEqualFold(asset.Type, "COMMODITY") {
				continue
			}
		case contractKindProcurement:
			if !stringsEqualFold(asset.Sector, "ENERGY") {
				continue
			}
		}
		volume := s.volumes[asset.ID]
		score := volume
		if score == 0 {
			score = s.basePrices[asset.ID]
		}
		if selected.ID == 0 || score > bestScore {
			selected = asset
			bestScore = score
		}
	}
	if selected.ID != 0 {
		return selected
	}
	if len(assets) > 0 {
		return assets[0]
	}
	return models.Asset{}
}

func contractTemplateForKind(kind contractKind, asset models.Asset) contractTemplate {
	switch kind {
	case contractKindProcurement:
		title := fmt.Sprintf("%s: %s Strategic Reserve", contractGovName, asset.Name)
		if strings.TrimSpace(asset.Name) == "" {
			title = fmt.Sprintf("%s: Strategic Reserve Procurement", contractGovName)
		}
		return contractTemplate{
			title:        title,
			assetID:      asset.ID,
			minRank:      contractGovMinRank,
			deadline:     contractDeadlineShort,
			baseRequired: contractGovBaseQty,
			xpPerUnit:    contractGovXP,
		}
	default:
		title := fmt.Sprintf("%s: %s Capacity Expansion", contractCorpName, asset.Name)
		if strings.TrimSpace(asset.Name) == "" {
			title = fmt.Sprintf("%s: Capacity Expansion", contractCorpName)
		}
		if asset.ID == contractAssetAUR {
			title = fmt.Sprintf("%s: Server Farm Expansion (Alpha)", contractCorpName)
		}
		return contractTemplate{
			title:        title,
			assetID:      asset.ID,
			minRank:      contractCapexMinRank,
			deadline:     contractDeadlineLong,
			baseRequired: contractCapexBaseQty,
			xpPerUnit:    contractCapexXP,
		}
	}
}

func (s *MarketStore) scaleContractRequirementLocked(assetID int64, base int64) int64 {
	if base <= 0 {
		return 0
	}
	volume := s.volumes[assetID]
	multiplier := int64(1)
	if volume > 0 {
		multiplier = volume / contractVolumeStep
		if multiplier < 1 {
			multiplier = 1
		}
		if multiplier > 3 {
			multiplier = 3
		}
	}
	required, ok := safeMultiplyInt64(base, multiplier)
	if !ok || required <= 0 {
		return base
	}
	return required
}

func (s *MarketStore) calculateContractPriceLocked(assetID int64, kind contractKind, now time.Time) int64 {
	base := s.contractVWAPLocked(assetID, now)
	if base == 0 {
		base = s.lastPrices[assetID]
	}
	if base == 0 {
		base = s.basePrices[assetID]
	}
	if base == 0 {
		base = defaultAssetPrice
	}
	premium := s.contractPremiumBpsLocked(assetID, kind)
	scaled, ok := safeMultiplyInt64(base, 10_000+premium)
	if !ok {
		return base
	}
	price := roundUpDiv(scaled, 10_000)
	if price <= 0 {
		return base
	}
	return price
}

func roundUpDiv(value, denom int64) int64 {
	if denom <= 0 {
		return value
	}
	adjusted, ok := safeAddInt64(value, denom-1)
	if !ok {
		return value / denom
	}
	return adjusted / denom
}

func (s *MarketStore) contractPremiumBpsLocked(assetID int64, kind contractKind) int64 {
	premium := contractPremiumMinBps
	volume := s.volumes[assetID]
	if volume > 0 {
		bump := volume / contractVolumeStep
		if bump > 3 {
			bump = 3
		}
		premium += bump * 100
	}
	if s.lastPriceChange(assetID) > 0 {
		premium += 100
	}
	if kind == contractKindCapex {
		premium += 100
	}
	if premium < contractPremiumMinBps {
		return contractPremiumMinBps
	}
	if premium > contractPremiumMaxBps {
		return contractPremiumMaxBps
	}
	return premium
}

func (s *MarketStore) contractVWAPLocked(assetID int64, now time.Time) int64 {
	if assetID == 0 {
		return 0
	}
	cutoff := now.Add(-contractVWAPWindow)
	var totalQty int64
	var totalNotional int64
	for _, exec := range s.executions {
		if exec.AssetID != assetID {
			continue
		}
		if exec.OccurredAtUTC.Before(cutoff) {
			continue
		}
		if exec.Quantity <= 0 || exec.Price <= 0 {
			continue
		}
		notional, ok := safeMultiplyInt64(exec.Price, exec.Quantity)
		if !ok {
			continue
		}
		totalNotional, ok = safeAddInt64(totalNotional, notional)
		if !ok {
			return 0
		}
		totalQty, ok = safeAddInt64(totalQty, exec.Quantity)
		if !ok {
			return 0
		}
	}
	if totalQty == 0 {
		return 0
	}
	return totalNotional / totalQty
}

func (s *MarketStore) Contracts(userID int64) []ContractStatus {
	now := time.Now().UTC()
	s.mu.Lock()
	s.refreshContractsLocked(now)
	contracts := make([]ContractStatus, 0, len(s.contracts))
	for _, contract := range s.contracts {
		contracts = append(contracts, s.contractStatusLocked(contract, userID))
	}
	s.mu.Unlock()
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].ID < contracts[j].ID })
	return contracts
}

func (s *MarketStore) Contract(contractID, userID int64) (ContractStatus, bool) {
	if contractID == 0 {
		return ContractStatus{}, false
	}
	now := time.Now().UTC()
	s.mu.Lock()
	s.refreshContractsLocked(now)
	contract, ok := s.contracts[contractID]
	if !ok {
		s.mu.Unlock()
		return ContractStatus{}, false
	}
	status := s.contractStatusLocked(contract, userID)
	s.mu.Unlock()
	return status, true
}

func (s *MarketStore) DeliverContract(userID, contractID, quantity int64) (ContractDeliveryResult, error) {
	if userID == 0 {
		return ContractDeliveryResult{}, errors.New("user_id required")
	}
	if contractID == 0 {
		return ContractDeliveryResult{}, errors.New("contract_id required")
	}
	if quantity <= 0 {
		return ContractDeliveryResult{}, errors.New("quantity must be positive")
	}
	now := time.Now().UTC().UnixMilli()
	s.mu.Lock()
	contract, ok := s.contracts[contractID]
	if !ok {
		s.mu.Unlock()
		return ContractDeliveryResult{}, errors.New("contract not found")
	}
	if contract.DeadlineAt > 0 && now > contract.DeadlineAt {
		s.mu.Unlock()
		return ContractDeliveryResult{}, errors.New("contract expired")
	}
	if contract.Delivered >= contract.TotalRequired {
		s.mu.Unlock()
		return ContractDeliveryResult{}, errors.New("contract fulfilled")
	}
	s.ensureUserLocked(userID)
	user := s.users[userID]
	rankDef := resolveUserRank(user)
	minRankDef, ok := rankDefinitionByName(contract.MinRank)
	if ok && rankDef.ID < minRankDef.ID {
		s.mu.Unlock()
		return ContractDeliveryResult{}, fmt.Errorf("rank %s required", minRankDef.Name)
	}
	remaining := contract.TotalRequired - contract.Delivered
	if quantity > remaining {
		s.mu.Unlock()
		return ContractDeliveryResult{}, errors.New("quantity exceeds remaining")
	}
	available := s.positions[userID][contract.AssetID]
	if available < quantity {
		s.mu.Unlock()
		return ContractDeliveryResult{}, errors.New("insufficient asset holdings")
	}
	contract.Delivered += quantity
	s.positions[userID][contract.AssetID] -= quantity
	contractProgress := s.ensureContractProgressLocked(userID)
	contractProgress[contractID] += quantity
	cashReward, ok := safeMultiplyInt64(quantity, contract.PricePerUnit)
	if !ok {
		s.mu.Unlock()
		return ContractDeliveryResult{}, errors.New("cash reward overflow")
	}
	xpReward, ok := safeMultiplyInt64(quantity, contract.XPPerUnit)
	if !ok {
		s.mu.Unlock()
		return ContractDeliveryResult{}, errors.New("xp reward overflow")
	}
	s.balances[userID][defaultCurrency] += cashReward
	if cashReward > 0 && s.contractKindForAssetLocked(contract.AssetID) == contractKindProcurement {
		s.recordGovernmentSpendingLocked(contractGovName, cashReward, time.UnixMilli(now))
	}
	if xpReward > 0 {
		user.XP += xpReward
		rank := rankDefinitionForXP(user.XP)
		user.Rank = rank.Name
		s.users[userID] = user
	}
	status := s.contractStatusLocked(contract, userID)
	info := s.userRankInfoLocked(userID)
	cash := s.balances[userID][defaultCurrency]
	assetQty := s.positions[userID][contract.AssetID]
	contractAssetID := contract.AssetID
	contractIDSnapshot := contract.ID
	contractSnapshot := *contract
	s.mu.Unlock()
	s.persistUser(user, cash)
	s.persistAssetBalance(userID, contractAssetID, assetQty)
	s.persistContract(&contractSnapshot)
	s.persistContractDelivery(contractIDSnapshot, userID, quantity, cashReward, xpReward, now)
	return ContractDeliveryResult{
		Contract:     status,
		Quantity:     quantity,
		CashReward:   cashReward,
		XPReward:     xpReward,
		UserProgress: info,
	}, nil
}

func (s *MarketStore) ensureContractProgressLocked(userID int64) map[int64]int64 {
	progress, ok := s.contractProgress[userID]
	if !ok {
		progress = make(map[int64]int64)
		s.contractProgress[userID] = progress
	}
	return progress
}

func (s *MarketStore) contractStatusLocked(contract *Contract, userID int64) ContractStatus {
	status := ContractStatus{}
	if contract == nil {
		return status
	}
	status.Contract = *contract
	status.Remaining = contract.TotalRequired - contract.Delivered
	if status.Remaining < 0 {
		status.Remaining = 0
	}
	if userID != 0 {
		status.UserDelivered = s.contractProgress[userID][contract.ID]
	}
	now := time.Now().UTC().UnixMilli()
	status.Expired = contract.DeadlineAt > 0 && now > contract.DeadlineAt
	status.Fulfilled = contract.Delivered >= contract.TotalRequired
	return status
}

func (s *MarketStore) seedContracts(now time.Time) {
	s.mu.Lock()
	s.refreshContractsLocked(now)
	s.mu.Unlock()
}
