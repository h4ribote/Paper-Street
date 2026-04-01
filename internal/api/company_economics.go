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
	defaultSharesIssued           = int64(1_000_000)
	defaultTreasuryBps            = int64(5_000)
	defaultMaxCapacity            = int64(10_000)
	defaultFixedCostPerUnit       = int64(50)
	defaultCapexIncreaseBps       = int64(2_000)
	defaultCapexLeadQuarters      = int64(2)
	defaultCapexCostPerUnit       = int64(500)
	financingDiscountNormalBps    = int64(400)
	financingDiscountEmergencyBps = int64(1_500)
	buybackLimitBps               = int64(2_500)
	buybackPremiumBps             = int64(500)
	capacityPressureBps           = int64(11_000)
	treasuryRetentionBps          = int64(6_000)
	overvaluationPriceBps         = int64(15_000)
	overvaluationPERatioBps       = int64(50) * bpsDenominator
	undervaluationPriceBps        = int64(8_000)
	excessCashWeeks               = int64(5)
)

type companyState struct {
	Company               Company
	UserID                int64
	Country               string
	OutputAssetID         int64
	MaxProductionCapacity int64
	CurrentInventory      int64
	LastCapexAt           int64
	SharesIssued          int64
	SharesOutstanding     int64
	TreasuryShares        int64
	LastProductionAt      int64
	LastProduction        int64
	LastSales             int64
	LastDemand            int64
	LastSalePrice         int64
	LastB2CRevenue        int64
	LastB2GRevenue        int64
	LastCapexCost         int64
	LastInventoryChange   int64
	UtilizationRate       int64
	InventoryCost         int64
	COGSPerUnit           int64
	CapacityPressureCount int64
	ActiveCapex           *capexProject
	LastFinancingAt       int64
	LastBuybackAt         int64
}

type capexProject struct {
	RemainingQuarters int64
	CapacityIncrease  int64
	Cost              int64
}

type ProductionInput struct {
	AssetID  int64 `json:"asset_id"`
	Quantity int64 `json:"quantity"`
}

type ProductionRecipe struct {
	ID             int64             `json:"id"`
	CompanyID      int64             `json:"company_id"`
	OutputAssetID  int64             `json:"output_asset_id"`
	OutputQuantity int64             `json:"output_quantity"`
	Inputs         []ProductionInput `json:"inputs,omitempty"`
}

type CompanyCapitalStructure struct {
	CompanyID         int64 `json:"company_id"`
	SharesIssued      int64 `json:"shares_issued"`
	SharesOutstanding int64 `json:"shares_outstanding"`
	TreasuryShares    int64 `json:"treasury_shares"`
	MarketPrice       int64 `json:"market_price"`
	MarketCap         int64 `json:"market_cap"`
}

type CompanyFinancingRequest struct {
	TargetAmount int64  `json:"target_amount"`
	Reason       string `json:"reason"`
}

type CompanyFinancingResult struct {
	CompanyID         int64  `json:"company_id"`
	Phase             string `json:"phase"`
	Reason            string `json:"reason"`
	TargetAmount      int64  `json:"target_amount"`
	OfferingPrice     int64  `json:"offering_price"`
	DiscountBps       int64  `json:"discount_bps"`
	SharesSold        int64  `json:"shares_sold"`
	CashRaised        int64  `json:"cash_raised"`
	SharesIssued      int64  `json:"shares_issued"`
	SharesOutstanding int64  `json:"shares_outstanding"`
	TreasuryShares    int64  `json:"treasury_shares"`
	DilutionBps       int64  `json:"dilution_bps"`
	NewsID            int64  `json:"news_id,omitempty"`
}

type CompanyBuybackRequest struct {
	Budget int64 `json:"budget"`
}

type CompanyBuybackResult struct {
	CompanyID         int64 `json:"company_id"`
	Budget            int64 `json:"budget"`
	Price             int64 `json:"price"`
	SharesRepurchased int64 `json:"shares_repurchased"`
	TreasuryShares    int64 `json:"treasury_shares"`
	SharesIssued      int64 `json:"shares_issued"`
	SharesOutstanding int64 `json:"shares_outstanding"`
	RetiredShares     int64 `json:"retired_shares"`
	NewsID            int64 `json:"news_id,omitempty"`
}

type CompanyDemandBreakdown struct {
	B2B   int64 `json:"b2b"`
	B2C   int64 `json:"b2c"`
	B2G   int64 `json:"b2g"`
	Total int64 `json:"total"`
}

type CompanyProductionStatus struct {
	CompanyID         int64 `json:"company_id"`
	OutputAssetID     int64 `json:"output_asset_id"`
	MaxCapacity       int64 `json:"max_capacity"`
	CurrentInventory  int64 `json:"current_inventory"`
	UtilizationRate   int64 `json:"utilization_rate"`
	LastDemand        int64 `json:"last_demand"`
	LastProduction    int64 `json:"last_production"`
	LastSales         int64 `json:"last_sales"`
	LastSalePrice     int64 `json:"last_sale_price"`
	InventoryWeeks    int64 `json:"inventory_weeks"`
	CapexInProgress   bool  `json:"capex_in_progress"`
	CapexCompletionAt int64 `json:"capex_completion_at,omitempty"`
}

type CompanySupplyChain struct {
	CompanyID int64              `json:"company_id"`
	Recipes   []ProductionRecipe `json:"recipes"`
}

type CompanyFinancialReport struct {
	CompanyID       int64  `json:"company_id"`
	FiscalYear      int    `json:"fiscal_year"`
	FiscalQuarter   int    `json:"fiscal_quarter"`
	Revenue         int64  `json:"revenue"`
	NetIncome       int64  `json:"net_income"`
	EPS             int64  `json:"eps"`
	Capex           int64  `json:"capex"`
	UtilizationRate int64  `json:"utilization_rate"`
	InventoryLevel  int64  `json:"inventory_level"`
	Guidance        string `json:"guidance"`
	PublishedAt     int64  `json:"published_at"`
}

type CompanySimulationResult struct {
	CompanyID  int64                  `json:"company_id"`
	Demand     CompanyDemandBreakdown `json:"demand"`
	Production int64                  `json:"production"`
	Sales      int64                  `json:"sales"`
	Revenue    int64                  `json:"revenue"`
	NetIncome  int64                  `json:"net_income"`
	Report     CompanyFinancialReport `json:"report"`
}

type demandProfile struct {
	BaseMultiplier float64
	GDPWeight      float64
	UnempWeight    float64
	CPIWeight      float64
}

var sectorDemandProfiles = map[string]demandProfile{
	"TECH":     {BaseMultiplier: 1.1, GDPWeight: 0.9, UnempWeight: 0.4, CPIWeight: 0.2},
	"ENERGY":   {BaseMultiplier: 1.0, GDPWeight: 0.5, UnempWeight: 0.2, CPIWeight: 0.1},
	"METAL":    {BaseMultiplier: 0.9, GDPWeight: 0.6, UnempWeight: 0.2, CPIWeight: 0.1},
	"FOOD":     {BaseMultiplier: 1.0, GDPWeight: 0.3, UnempWeight: 0.1, CPIWeight: 0.1},
	"SERVICES": {BaseMultiplier: 1.0, GDPWeight: 0.7, UnempWeight: 0.3, CPIWeight: 0.2},
}

func (s *MarketStore) CompanyCapitalStructure(companyID int64) (CompanyCapitalStructure, bool) {
	if companyID == 0 {
		return CompanyCapitalStructure{}, false
	}
	s.mu.RLock()
	state := s.companyStates[companyID]
	if state == nil {
		s.mu.RUnlock()
		return CompanyCapitalStructure{}, false
	}
	price := s.marketPriceLocked(state.Company.ID)
	sharesIssued := state.SharesIssued
	sharesOutstanding := state.SharesOutstanding
	treasury := state.TreasuryShares
	s.mu.RUnlock()
	marketCap, _ := safeMultiplyInt64(price, sharesOutstanding)
	return CompanyCapitalStructure{
		CompanyID:         companyID,
		SharesIssued:      sharesIssued,
		SharesOutstanding: sharesOutstanding,
		TreasuryShares:    treasury,
		MarketPrice:       price,
		MarketCap:         marketCap,
	}, true
}

func (s *MarketStore) CompanySupplyChain(companyID int64) (CompanySupplyChain, bool) {
	if companyID == 0 {
		return CompanySupplyChain{}, false
	}
	s.mu.RLock()
	recipes := s.companyRecipes[companyID]
	s.mu.RUnlock()
	if recipes == nil {
		recipes = []ProductionRecipe{}
	}
	return CompanySupplyChain{CompanyID: companyID, Recipes: recipes}, true
}

func (s *MarketStore) CompanyProductionStatus(companyID int64) (CompanyProductionStatus, bool) {
	if companyID == 0 {
		return CompanyProductionStatus{}, false
	}
	now := time.Now().UTC()
	s.mu.Lock()
	state := s.companyStates[companyID]
	if state == nil {
		s.mu.Unlock()
		return CompanyProductionStatus{}, false
	}
	s.maybeRunCompanyCycleLocked(state, now)
	status := s.buildProductionStatusLocked(state, now)
	s.mu.Unlock()
	return status, true
}

func (s *MarketStore) CompanyFinancialReports(companyID int64, limit int) []CompanyFinancialReport {
	if companyID == 0 {
		return nil
	}
	s.mu.RLock()
	reports := append([]CompanyFinancialReport(nil), s.financialReports[companyID]...)
	s.mu.RUnlock()
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].FiscalYear == reports[j].FiscalYear {
			return reports[i].FiscalQuarter > reports[j].FiscalQuarter
		}
		return reports[i].FiscalYear > reports[j].FiscalYear
	})
	if limit > 0 && len(reports) > limit {
		reports = reports[:limit]
	}
	return reports
}

func (s *MarketStore) SimulateCompanyQuarter(companyID int64, now time.Time) (CompanySimulationResult, error) {
	if companyID == 0 {
		return CompanySimulationResult{}, errors.New("company id required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	state := s.companyStates[companyID]
	if state == nil {
		s.mu.Unlock()
		return CompanySimulationResult{}, errors.New("company not found")
	}
	result := s.runCompanyQuarterLocked(state, now)
	report := result.Report
	s.mu.Unlock()
	s.persistCompanyState(state)
	s.persistFinancialReport(report)
	return result, nil
}

func (s *MarketStore) InitiateEquityFinancing(companyID int64, req CompanyFinancingRequest) (CompanyFinancingResult, error) {
	if companyID == 0 {
		return CompanyFinancingResult{}, errors.New("company id required")
	}
	s.mu.Lock()
	state := s.companyStates[companyID]
	if state == nil {
		s.mu.Unlock()
		return CompanyFinancingResult{}, errors.New("company not found")
	}
	result, err := s.initiateEquityFinancingLocked(state, req)
	s.mu.Unlock()
	if err != nil {
		return CompanyFinancingResult{}, err
	}
	s.persistCompanyState(state)
	return result, nil
}

func (s *MarketStore) AuthorizeShareBuyback(companyID int64, req CompanyBuybackRequest) (CompanyBuybackResult, error) {
	if companyID == 0 {
		return CompanyBuybackResult{}, errors.New("company id required")
	}
	s.mu.Lock()
	state := s.companyStates[companyID]
	if state == nil {
		s.mu.Unlock()
		return CompanyBuybackResult{}, errors.New("company not found")
	}
	result, err := s.authorizeShareBuybackLocked(state, req)
	s.mu.Unlock()
	if err != nil {
		return CompanyBuybackResult{}, err
	}
	s.persistCompanyState(state)
	return result, nil
}

func (s *MarketStore) seedCompanies() {
	commodityID := s.firstCommodityAssetID()
	for _, asset := range s.assets {
		if !stringsEqualFold(asset.Type, "STOCK") {
			continue
		}
		state := s.ensureCompanyStateLocked(asset, commodityID)
		if state == nil {
			continue
		}
	}
}

func (s *MarketStore) seedProductionRecipes() {
	if len(s.companyStates) == 0 {
		return
	}
	for companyID, state := range s.companyStates {
		if state.OutputAssetID == 0 {
			continue
		}
		if len(s.companyRecipes[companyID]) > 0 {
			continue
		}
		s.nextRecipeID++
		s.companyRecipes[companyID] = []ProductionRecipe{
			{
				ID:             s.nextRecipeID,
				CompanyID:      companyID,
				OutputAssetID:  state.OutputAssetID,
				OutputQuantity: 1,
			},
		}
	}
}

func (s *MarketStore) ensureCompanyStateLocked(asset models.Asset, commodityID int64) *companyState {
	state := s.companyStates[asset.ID]
	if state != nil {
		return state
	}
	userID := asset.ID
	user := s.ensureUserLocked(userID)
	if user.Role != "bot" {
		user.Role = "bot"
		s.users[userID] = user
	}
	issued := defaultSharesIssued
	treasury := issued * defaultTreasuryBps / bpsDenominator
	outstanding := issued - treasury
	state = &companyState{
		Company: Company{
			ID:     asset.ID,
			Name:   asset.Name,
			Symbol: asset.Symbol,
			Sector: asset.Sector,
		},
		UserID:                userID,
		Country:               s.defaultCountryForSector(asset.Sector),
		OutputAssetID:         commodityID,
		MaxProductionCapacity: defaultMaxCapacity,
		SharesIssued:          issued,
		SharesOutstanding:     outstanding,
		TreasuryShares:        treasury,
		LastSalePrice:         s.marketPriceLocked(asset.ID),
	}
	s.companyStates[asset.ID] = state
	return state
}

func (s *MarketStore) defaultCountryForSector(sector string) string {
	switch strings.ToUpper(strings.TrimSpace(sector)) {
	case "ENERGY":
		return "Boros Federation"
	case "METAL", "FOOD":
		return "El Dorado"
	default:
		return fxArcadiaCountry
	}
}

func (s *MarketStore) firstCommodityAssetID() int64 {
	var first int64
	for assetID, asset := range s.assets {
		if stringsEqualFold(asset.Type, "COMMODITY") {
			if first == 0 || assetID < first {
				first = assetID
			}
		}
	}
	return first
}

func (s *MarketStore) buildProductionStatusLocked(state *companyState, now time.Time) CompanyProductionStatus {
	inventory := state.CurrentInventory
	if state.OutputAssetID != 0 {
		if qty, ok := s.positions[state.UserID][state.OutputAssetID]; ok {
			inventory = qty
		}
	}
	weeklySales := int64(1)
	if state.LastDemand > 0 {
		weeklySales = maxInt64(1, state.LastDemand/2)
	}
	inventoryWeeks := inventory / weeklySales
	capexCompletion := int64(0)
	if state.ActiveCapex != nil {
		capexCompletion = now.Add(time.Duration(state.ActiveCapex.RemainingQuarters) * macroQuarterPeriod).UnixMilli()
	}
	return CompanyProductionStatus{
		CompanyID:         state.Company.ID,
		OutputAssetID:     state.OutputAssetID,
		MaxCapacity:       state.MaxProductionCapacity,
		CurrentInventory:  inventory,
		UtilizationRate:   state.UtilizationRate,
		LastDemand:        state.LastDemand,
		LastProduction:    state.LastProduction,
		LastSales:         state.LastSales,
		LastSalePrice:     state.LastSalePrice,
		InventoryWeeks:    inventoryWeeks,
		CapexInProgress:   state.ActiveCapex != nil,
		CapexCompletionAt: capexCompletion,
	}
}

func (s *MarketStore) maybeRunCompanyCycleLocked(state *companyState, now time.Time) {
	if state == nil {
		return
	}
	if state.LastProductionAt == 0 {
		return
	}
	elapsed := now.Sub(time.UnixMilli(state.LastProductionAt))
	if elapsed < macroQuarterPeriod {
		return
	}
	_ = s.runCompanyQuarterLocked(state, now)
}

func (s *MarketStore) runCompanyQuarterLocked(state *companyState, now time.Time) CompanySimulationResult {
	result := CompanySimulationResult{CompanyID: state.Company.ID}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	startInventory := state.CurrentInventory
	demand := s.calculateCompanyDemandLocked(state)
	result.Demand = demand

	production := s.runCompanyProductionLocked(state, demand.Total, now)
	sales, revenue, netIncome := s.runCompanySalesLocked(state, demand.Total, production)
	capexCost := s.handleCapexLocked(state, demand.Total, now)
	updateMacroQuarterTracking(state, revenue, demand, capexCost, startInventory)
	if capexCost > 0 {
		netIncome -= capexCost
	}
	result.Production = production
	result.Sales = sales
	result.Revenue = revenue
	result.NetIncome = netIncome

	report := s.buildFinancialReportLocked(state, netIncome, revenue, capexCost, now)
	result.Report = report

	s.evaluateFinancingLocked(state, demand, now)
	return result
}

func splitDemandRevenue(revenue int64, demand CompanyDemandBreakdown) (int64, int64) {
	if revenue <= 0 || demand.Total <= 0 {
		return 0, 0
	}
	if demand.B2C < 0 || demand.B2G < 0 {
		return 0, 0
	}
	denominator := demand.Total
	componentTotal := demand.B2C + demand.B2G
	if componentTotal > demand.Total {
		denominator = componentTotal
		if denominator <= 0 {
			return 0, 0
		}
	}
	b2c := revenue * demand.B2C / denominator
	b2g := revenue * demand.B2G / denominator
	return b2c, b2g
}

func updateMacroQuarterTracking(state *companyState, revenue int64, demand CompanyDemandBreakdown, capexCost int64, startInventory int64) {
	if state == nil {
		return
	}
	state.LastCapexCost = capexCost
	state.LastB2CRevenue, state.LastB2GRevenue = splitDemandRevenue(revenue, demand)
	state.LastInventoryChange = inventoryChangeValue(startInventory, state.CurrentInventory, state.COGSPerUnit, state.LastSalePrice)
}

func inventoryChangeValue(startInventory, endInventory, unitCost, fallbackPrice int64) int64 {
	delta := endInventory - startInventory
	if delta == 0 {
		return 0
	}
	unit := unitCost
	if unit <= 0 {
		unit = fallbackPrice
	}
	if unit <= 0 {
		unit = defaultAssetPrice
	}
	return delta * unit
}

func (s *MarketStore) calculateCompanyDemandLocked(state *companyState) CompanyDemandBreakdown {
	b2b := s.calculateB2BDemandLocked(state)
	b2c := s.calculateB2CDemandLocked(state)
	b2g := s.calculateB2GDemandLocked(state)
	total := b2b + b2c + b2g
	if total < 0 {
		total = 0
	}
	state.LastDemand = total
	return CompanyDemandBreakdown{B2B: b2b, B2C: b2c, B2G: b2g, Total: total}
}

func (s *MarketStore) calculateB2BDemandLocked(state *companyState) int64 {
	if state.OutputAssetID == 0 {
		return 0
	}
	var total int64
	for companyID, recipes := range s.companyRecipes {
		if companyID == state.Company.ID {
			continue
		}
		downstream := s.companyStates[companyID]
		if downstream == nil {
			continue
		}
		for _, recipe := range recipes {
			for _, input := range recipe.Inputs {
				if input.AssetID != state.OutputAssetID {
					continue
				}
				required, ok := safeMultiplyInt64(downstream.MaxProductionCapacity, input.Quantity)
				if !ok {
					continue
				}
				total += required
			}
		}
	}
	return total
}

func (s *MarketStore) calculateB2CDemandLocked(state *companyState) int64 {
	if state.MaxProductionCapacity == 0 {
		return 0
	}
	profile := sectorDemandProfiles[strings.ToUpper(strings.TrimSpace(state.Company.Sector))]
	if profile.BaseMultiplier == 0 {
		profile = demandProfile{BaseMultiplier: 1.0, GDPWeight: 0.5, UnempWeight: 0.2, CPIWeight: 0.1}
	}
	base := float64(state.MaxProductionCapacity) * profile.BaseMultiplier
	values, ok := s.macroIndicatorValuesLocked(state.Country)
	if !ok {
		return int64(base)
	}
	gdp := float64(values.gdp) / 100.0
	unemp := float64(values.unemp) / 100.0
	cpi := float64(values.cpi) / 100.0
	gdpFactor := 1.0 + (gdp/100.0)*profile.GDPWeight
	unempFactor := 1.0 - (unemp/100.0)*profile.UnempWeight
	cpiFactor := 1.0 - (cpi/100.0)*profile.CPIWeight
	demand := base * gdpFactor * unempFactor * cpiFactor
	if demand < 0 {
		return 0
	}
	return int64(demand)
}

func (s *MarketStore) calculateB2GDemandLocked(state *companyState) int64 {
	if state.OutputAssetID == 0 {
		return 0
	}
	var total int64
	for _, contract := range s.contracts {
		if contract == nil || contract.AssetID != state.OutputAssetID {
			continue
		}
		remaining := contract.TotalRequired - contract.Delivered
		if remaining < 0 {
			continue
		}
		total += remaining
	}
	return total
}

func (s *MarketStore) runCompanyProductionLocked(state *companyState, demandTotal int64, now time.Time) int64 {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	state.LastProductionAt = now.UnixMilli()
	recipes := s.companyRecipes[state.Company.ID]
	if len(recipes) == 0 {
		state.LastProduction = 0
		state.UtilizationRate = 0
		return 0
	}
	targetOutput := state.MaxProductionCapacity
	production := targetOutput
	production = s.procureInputsLocked(state, production, recipes)
	if production < 0 {
		production = 0
	}
	production = minInt64(production, targetOutput)
	if production > targetOutput {
		production = targetOutput
	}
	for _, recipe := range recipes {
		for _, input := range recipe.Inputs {
			consumed := production * input.Quantity
			s.positions[state.UserID][input.AssetID] -= consumed
		}
	}
	if state.OutputAssetID != 0 {
		s.positions[state.UserID][state.OutputAssetID] += production
		state.CurrentInventory = s.positions[state.UserID][state.OutputAssetID]
	}
	state.LastProduction = production
	if state.MaxProductionCapacity > 0 {
		state.UtilizationRate = production * bpsDenominator / state.MaxProductionCapacity
	}
	return production
}

func (s *MarketStore) procureInputsLocked(state *companyState, production int64, recipes []ProductionRecipe) int64 {
	if production <= 0 {
		return 0
	}
	cash := s.balances[state.UserID][defaultCurrency]
	production = s.affordableProductionLocked(state, production, recipes, cash)
	if production <= 0 {
		return 0
	}
	for _, recipe := range recipes {
		for _, input := range recipe.Inputs {
			if input.Quantity <= 0 {
				continue
			}
			price := s.marketPriceLocked(input.AssetID)
			if price <= 0 {
				continue
			}
			required, ok := safeMultiplyInt64(production, input.Quantity)
			if !ok {
				continue
			}
			available := s.positions[state.UserID][input.AssetID]
			if available >= required {
				continue
			}
			shortfall := required - available
			cost, ok := safeMultiplyInt64(shortfall, price)
			if !ok || cost <= 0 {
				continue
			}
			if cost > cash {
				shortfall = cash / price
				cost = shortfall * price
			}
			if shortfall <= 0 || cost <= 0 {
				continue
			}
			cash -= cost
			s.positions[state.UserID][input.AssetID] += shortfall
		}
	}
	s.balances[state.UserID][defaultCurrency] = cash
	return production
}

func (s *MarketStore) affordableProductionLocked(state *companyState, target int64, recipes []ProductionRecipe, cash int64) int64 {
	if target <= 0 {
		return 0
	}
	low := int64(0)
	high := target
	for low < high {
		mid := (low + high + 1) / 2
		if s.canAffordProductionLocked(state, mid, recipes, cash) {
			low = mid
			continue
		}
		high = mid - 1
	}
	return low
}

func (s *MarketStore) canAffordProductionLocked(state *companyState, production int64, recipes []ProductionRecipe, cash int64) bool {
	if production <= 0 {
		return true
	}
	requiredCash := int64(0)
	for _, recipe := range recipes {
		for _, input := range recipe.Inputs {
			if input.Quantity <= 0 {
				continue
			}
			required, ok := safeMultiplyInt64(production, input.Quantity)
			if !ok {
				return false
			}
			available := s.positions[state.UserID][input.AssetID]
			if available >= required {
				continue
			}
			price := s.marketPriceLocked(input.AssetID)
			if price <= 0 {
				return false
			}
			shortfall := required - available
			cost, ok := safeMultiplyInt64(shortfall, price)
			if !ok || cost <= 0 {
				return false
			}
			if cost > cash || requiredCash > cash-cost {
				return false
			}
			requiredCash += cost
		}
	}
	return true
}

func (s *MarketStore) runCompanySalesLocked(state *companyState, demandTotal, production int64) (int64, int64, int64) {
	inventory := state.CurrentInventory
	if state.OutputAssetID != 0 {
		inventory = s.positions[state.UserID][state.OutputAssetID]
	}
	sales := minInt64(inventory, demandTotal)
	if sales < 0 {
		sales = 0
	}
	price := s.marketPriceLocked(state.OutputAssetID)
	if price <= 0 {
		price = defaultAssetPrice
	}
	weeklySales := maxInt64(1, demandTotal/2)
	inventoryWeeks := inventory / weeklySales
	discountBps := int64(0)
	premiumBps := int64(0)
	if inventoryWeeks > 4 {
		discountBps = minInt64(1_500, (inventoryWeeks-4)*200)
	}
	if inventory == 0 && demandTotal > state.MaxProductionCapacity {
		premiumBps = buybackPremiumBps
	}
	adjustedPrice := applyBps(price, premiumBps, discountBps)
	if adjustedPrice <= 0 {
		adjustedPrice = price
	}
	revenue := adjustedPrice * sales
	s.balances[state.UserID][defaultCurrency] += revenue
	inventory -= sales
	if state.OutputAssetID != 0 {
		s.positions[state.UserID][state.OutputAssetID] = inventory
	}
	state.CurrentInventory = inventory
	state.LastSales = sales
	state.LastSalePrice = adjustedPrice
	state.COGSPerUnit = s.updateCOGSPerUnitLocked(state, production)
	cogs := state.COGSPerUnit * sales
	fixedCost := state.MaxProductionCapacity * defaultFixedCostPerUnit
	if fixedCost > 0 {
		cash := s.balances[state.UserID][defaultCurrency]
		if fixedCost > cash {
			fixedCost = cash
		}
		s.balances[state.UserID][defaultCurrency] = cash - fixedCost
	}
	netIncome := revenue - cogs - fixedCost
	return sales, revenue, netIncome
}

func (s *MarketStore) handleCapexLocked(state *companyState, demandTotal int64, now time.Time) int64 {
	if state.ActiveCapex != nil {
		state.ActiveCapex.RemainingQuarters--
		if state.ActiveCapex.RemainingQuarters <= 0 {
			state.MaxProductionCapacity += state.ActiveCapex.CapacityIncrease
			state.LastCapexAt = now.UnixMilli()
			state.ActiveCapex = nil
		}
	}
	if state.MaxProductionCapacity == 0 {
		return 0
	}
	if demandTotal > state.MaxProductionCapacity*capacityPressureBps/bpsDenominator {
		state.CapacityPressureCount++
	} else {
		state.CapacityPressureCount = 0
	}
	if state.ActiveCapex != nil || state.CapacityPressureCount < 2 {
		return 0
	}
	increase := state.MaxProductionCapacity * defaultCapexIncreaseBps / bpsDenominator
	if increase <= 0 {
		increase = 1
	}
	cost := increase * defaultCapexCostPerUnit
	cash := s.balances[state.UserID][defaultCurrency]
	if cost > cash {
		cost = cash
	}
	s.balances[state.UserID][defaultCurrency] = cash - cost
	state.ActiveCapex = &capexProject{RemainingQuarters: defaultCapexLeadQuarters, CapacityIncrease: increase, Cost: cost}
	state.CapacityPressureCount = 0
	state.LastCapexAt = now.UnixMilli()
	return cost
}

func (s *MarketStore) updateCOGSPerUnitLocked(state *companyState, production int64) int64 {
	unitCost := s.inputUnitCostLocked(state)
	return unitCost
}

func (s *MarketStore) inputUnitCostLocked(state *companyState) int64 {
	recipes := s.companyRecipes[state.Company.ID]
	if len(recipes) == 0 {
		return state.LastSalePrice / 2
	}
	var total int64
	for _, recipe := range recipes {
		for _, input := range recipe.Inputs {
			price := s.marketPriceLocked(input.AssetID)
			if price <= 0 {
				continue
			}
			total += price * input.Quantity
		}
	}
	if total == 0 {
		price := s.marketPriceLocked(state.OutputAssetID)
		if price == 0 {
			price = defaultAssetPrice
		}
		return price / 2
	}
	return total
}

func (s *MarketStore) buildFinancialReportLocked(state *companyState, netIncome, revenue, capex int64, now time.Time) CompanyFinancialReport {
	year, quarter := fiscalPeriod(now)
	eps := int64(0)
	if state.SharesOutstanding > 0 {
		eps = netIncome / state.SharesOutstanding
	}
	report := CompanyFinancialReport{
		CompanyID:       state.Company.ID,
		FiscalYear:      year,
		FiscalQuarter:   quarter,
		Revenue:         revenue,
		NetIncome:       netIncome,
		EPS:             eps,
		Capex:           capex,
		UtilizationRate: state.UtilizationRate,
		InventoryLevel:  state.CurrentInventory,
		Guidance:        "",
		PublishedAt:     now.UnixMilli(),
	}
	s.storeFinancialReportLocked(state.Company.ID, report)
	return report
}

func (s *MarketStore) storeFinancialReportLocked(companyID int64, report CompanyFinancialReport) {
	reports := s.financialReports[companyID]
	for i, existing := range reports {
		if existing.FiscalYear == report.FiscalYear && existing.FiscalQuarter == report.FiscalQuarter {
			reports[i] = report
			s.financialReports[companyID] = reports
			return
		}
	}
	s.financialReports[companyID] = append(reports, report)
}

func (s *MarketStore) evaluateFinancingLocked(state *companyState, demand CompanyDemandBreakdown, now time.Time) {
	if state == nil {
		return
	}
	weeklyCost := state.MaxProductionCapacity * defaultFixedCostPerUnit / 2
	cash := s.balances[state.UserID][defaultCurrency]
	price := s.marketPriceLocked(state.Company.ID)
	ma := s.movingAveragePriceLocked(state.Company.ID, 200*24*time.Hour)
	pe := int64(0)
	if state.SharesOutstanding > 0 {
		lastReport := s.latestFinancialReportLocked(state.Company.ID)
		if lastReport.EPS > 0 {
			pe = price * bpsDenominator / lastReport.EPS
		}
	}
	if cash < weeklyCost*2 {
		_, _ = s.initiateEquityFinancingLocked(state, CompanyFinancingRequest{Reason: "SAFETY_MARGIN"})
		return
	}
	if ma > 0 && price > ma*overvaluationPriceBps/bpsDenominator && pe > overvaluationPERatioBps {
		_, _ = s.initiateEquityFinancingLocked(state, CompanyFinancingRequest{Reason: "OVERVALUATION"})
		return
	}
	if cash > weeklyCost*excessCashWeeks && (ma == 0 || price < ma*undervaluationPriceBps/bpsDenominator) {
		_, _ = s.authorizeShareBuybackLocked(state, CompanyBuybackRequest{})
	}
}

func (s *MarketStore) initiateEquityFinancingLocked(state *companyState, req CompanyFinancingRequest) (CompanyFinancingResult, error) {
	cash := s.balances[state.UserID][defaultCurrency]
	weeklyCost := state.MaxProductionCapacity * defaultFixedCostPerUnit / 2
	target := req.TargetAmount
	if target <= 0 {
		shortfall := weeklyCost*2 - cash
		if shortfall < 0 {
			shortfall = 0
		}
		target = shortfall + weeklyCost
	}
	if target <= 0 {
		return CompanyFinancingResult{}, errors.New("no financing required")
	}
	price := s.marketPriceLocked(state.Company.ID)
	if price <= 0 {
		return CompanyFinancingResult{}, errors.New("market price unavailable")
	}
	discountBps := financingDiscountNormalBps
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "CAPITAL_NEED"
	}
	if stringsEqualFold(reason, "SAFETY_MARGIN") || stringsEqualFold(reason, "CAPEX") {
		discountBps = financingDiscountEmergencyBps
	}
	offeringPrice := applyDiscountBps(price, discountBps)
	if offeringPrice <= 0 {
		return CompanyFinancingResult{}, errors.New("invalid offering price")
	}
	sharesNeeded := ceilDiv(target, offeringPrice)
	phase := "SECONDARY"
	soldFromTreasury := minInt64(sharesNeeded, state.TreasuryShares)
	newShares := sharesNeeded - soldFromTreasury
	if newShares > 0 {
		phase = "PRIMARY"
		state.SharesIssued += newShares
	}
	state.TreasuryShares -= soldFromTreasury
	state.SharesOutstanding += sharesNeeded
	cashRaised := sharesNeeded * offeringPrice
	s.balances[state.UserID][defaultCurrency] += cashRaised
	dilutionBps := int64(0)
	if state.SharesOutstanding > 0 {
		dilutionBps = sharesNeeded * bpsDenominator / state.SharesOutstanding
	}
	newsID := s.addNewsLocked(fmt.Sprintf("[FINANCE] %s announces plan to raise %d via %s offering.", state.Company.Name, cashRaised, phase), state.Company.ID, "FINANCE")
	state.LastFinancingAt = time.Now().UTC().UnixMilli()
	return CompanyFinancingResult{
		CompanyID:         state.Company.ID,
		Phase:             phase,
		Reason:            reason,
		TargetAmount:      target,
		OfferingPrice:     offeringPrice,
		DiscountBps:       discountBps,
		SharesSold:        sharesNeeded,
		CashRaised:        cashRaised,
		SharesIssued:      state.SharesIssued,
		SharesOutstanding: state.SharesOutstanding,
		TreasuryShares:    state.TreasuryShares,
		DilutionBps:       dilutionBps,
		NewsID:            newsID,
	}, nil
}

func (s *MarketStore) authorizeShareBuybackLocked(state *companyState, req CompanyBuybackRequest) (CompanyBuybackResult, error) {
	cash := s.balances[state.UserID][defaultCurrency]
	weeklyCost := state.MaxProductionCapacity * defaultFixedCostPerUnit / 2
	budget := req.Budget
	if budget <= 0 {
		excess := cash - weeklyCost*5
		if excess <= 0 {
			return CompanyBuybackResult{}, errors.New("no excess cash for buyback")
		}
		budget = excess / 2
	}
	if budget <= 0 {
		return CompanyBuybackResult{}, errors.New("buyback budget too small")
	}
	price := s.vwapPriceLocked(state.Company.ID, 24*time.Hour)
	if price <= 0 {
		price = s.marketPriceLocked(state.Company.ID)
	}
	if price <= 0 {
		return CompanyBuybackResult{}, errors.New("market price unavailable")
	}
	limitPrice := applyBps(price, buybackPremiumBps, 0)
	if price > limitPrice {
		price = limitPrice
	}
	maxShares := budget / price
	volumeLimit := s.averageVolumeLocked(state.Company.ID, 5*24*time.Hour)
	if volumeLimit > 0 {
		volumeLimit = volumeLimit * buybackLimitBps / bpsDenominator
		if maxShares > volumeLimit {
			maxShares = volumeLimit
		}
	}
	if maxShares <= 0 {
		return CompanyBuybackResult{}, errors.New("buyback limit reached")
	}
	if maxShares > state.SharesOutstanding {
		maxShares = state.SharesOutstanding
	}
	cost := maxShares * price
	if cost > cash {
		maxShares = cash / price
		cost = maxShares * price
	}
	if maxShares <= 0 {
		return CompanyBuybackResult{}, errors.New("insufficient cash")
	}
	s.balances[state.UserID][defaultCurrency] -= cost
	state.SharesOutstanding -= maxShares
	state.TreasuryShares += maxShares
	retired := int64(0)
	treasuryLimit := state.SharesIssued * treasuryRetentionBps / bpsDenominator
	if state.TreasuryShares > treasuryLimit {
		retired = state.TreasuryShares - treasuryLimit
		state.TreasuryShares -= retired
		state.SharesIssued -= retired
	}
	newsID := s.addNewsLocked(fmt.Sprintf("[BUYBACK] %s authorizes %d share repurchase program.", state.Company.Name, maxShares), state.Company.ID, "BUYBACK")
	state.LastBuybackAt = time.Now().UTC().UnixMilli()
	return CompanyBuybackResult{
		CompanyID:         state.Company.ID,
		Budget:            budget,
		Price:             price,
		SharesRepurchased: maxShares,
		TreasuryShares:    state.TreasuryShares,
		SharesIssued:      state.SharesIssued,
		SharesOutstanding: state.SharesOutstanding,
		RetiredShares:     retired,
		NewsID:            newsID,
	}, nil
}

func (s *MarketStore) addNewsLocked(headline string, assetID int64, category string) int64 {
	s.nextNewsID++
	item := NewsItem{
		ID:          s.nextNewsID,
		Headline:    headline,
		AssetID:     assetID,
		Category:    category,
		PublishedAt: time.Now().UTC().UnixMilli(),
	}
	s.news = append([]NewsItem{item}, s.news...)
	s.persistNewsItem(item)
	return item.ID
}

func (s *MarketStore) marketPriceLocked(assetID int64) int64 {
	if assetID == 0 {
		return 0
	}
	price := s.lastPrices[assetID]
	if price == 0 {
		price = s.basePrices[assetID]
	}
	if price == 0 {
		price = defaultAssetPrice
	}
	return price
}

func (s *MarketStore) movingAveragePriceLocked(assetID int64, window time.Duration) int64 {
	if assetID == 0 || window <= 0 {
		return s.marketPriceLocked(assetID)
	}
	cutoff := time.Now().UTC().Add(-window)
	var sum int64
	var count int64
	for _, exec := range s.executions {
		if exec.AssetID != assetID {
			continue
		}
		if exec.OccurredAtUTC.Before(cutoff) {
			continue
		}
		sum += exec.Price
		count++
	}
	if count == 0 {
		return s.marketPriceLocked(assetID)
	}
	return sum / count
}

func (s *MarketStore) vwapPriceLocked(assetID int64, window time.Duration) int64 {
	if assetID == 0 {
		return 0
	}
	cutoff := time.Now().UTC().Add(-window)
	var totalValue int64
	var totalQty int64
	for _, exec := range s.executions {
		if exec.AssetID != assetID {
			continue
		}
		if exec.OccurredAtUTC.Before(cutoff) {
			continue
		}
		totalValue += exec.Price * exec.Quantity
		totalQty += exec.Quantity
	}
	if totalQty == 0 {
		return s.marketPriceLocked(assetID)
	}
	return totalValue / totalQty
}

func (s *MarketStore) averageVolumeLocked(assetID int64, window time.Duration) int64 {
	if assetID == 0 {
		return 0
	}
	cutoff := time.Now().UTC().Add(-window)
	var total int64
	for _, exec := range s.executions {
		if exec.AssetID != assetID {
			continue
		}
		if exec.OccurredAtUTC.Before(cutoff) {
			continue
		}
		total += exec.Quantity
	}
	if total == 0 {
		total = s.volumes[assetID]
	}
	if total == 0 {
		return 0
	}
	days := int64(window / (24 * time.Hour))
	if days <= 0 {
		days = 1
	}
	return total / days
}

func (s *MarketStore) latestFinancialReportLocked(companyID int64) CompanyFinancialReport {
	reports := s.financialReports[companyID]
	if len(reports) == 0 {
		return CompanyFinancialReport{}
	}
	latest := reports[0]
	for _, report := range reports[1:] {
		if report.FiscalYear > latest.FiscalYear {
			latest = report
			continue
		}
		if report.FiscalYear == latest.FiscalYear && report.FiscalQuarter > latest.FiscalQuarter {
			latest = report
		}
	}
	return latest
}

func applyBps(value, premiumBps, discountBps int64) int64 {
	if value <= 0 {
		return value
	}
	adjusted := value
	if premiumBps > 0 {
		adjusted = adjusted * (bpsDenominator + premiumBps) / bpsDenominator
	}
	if discountBps > 0 {
		adjusted = adjusted * (bpsDenominator - discountBps) / bpsDenominator
	}
	return adjusted
}

func fiscalPeriod(now time.Time) (int, int) {
	year := now.Year()
	quarter := int((now.Month()-1)/3) + 1
	return year, quarter
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func ceilDiv(value, divisor int64) int64 {
	if divisor == 0 {
		return 0
	}
	if value <= 0 {
		return 0
	}
	return (value + divisor - 1) / divisor
}
