package api

import (
	"testing"
	"time"
)

func TestMacroIndicatorsIncludeRequiredTypes(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

	store.mu.Lock()
	store.refreshMacroIndicatorsLocked(now)
	indicators := append([]MacroIndicator(nil), store.macroIndicators...)
	store.mu.Unlock()

	expectedTypes := []string{
		macroTypeGDPGrowth,
		macroTypeCPI,
		macroTypeUnemp,
		macroTypeInterest,
		macroTypeCCI,
	}
	found := make(map[string]map[string]bool)
	for _, indicator := range indicators {
		if _, ok := found[indicator.Country]; !ok {
			found[indicator.Country] = make(map[string]bool)
		}
		found[indicator.Country][indicator.Type] = true
	}
	for _, profile := range macroProfiles {
		for _, indicatorType := range expectedTypes {
			if !found[profile.Country][indicatorType] {
				t.Fatalf("expected indicator type %s for %s", indicatorType, profile.Country)
			}
		}
	}
}

func TestMacroIndicatorsPublishedAtPeriodStart(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 10, 15, 30, 0, 0, time.UTC)
	quarterStart := macroPeriodStart(now, macroQuarterPeriod).UnixMilli()
	weekStart := macroPeriodStart(now, macroWeekPeriod).UnixMilli()

	store.mu.Lock()
	store.refreshMacroIndicatorsLocked(now)
	indicators := append([]MacroIndicator(nil), store.macroIndicators...)
	store.mu.Unlock()

	var gdpPublished, cciPublished int64
	for _, indicator := range indicators {
		if indicator.Country != "Arcadia" {
			continue
		}
		switch indicator.Type {
		case macroTypeGDPGrowth:
			gdpPublished = indicator.PublishedAt
		case macroTypeCCI:
			cciPublished = indicator.PublishedAt
		}
	}
	if gdpPublished == 0 || cciPublished == 0 {
		t.Fatalf("expected Arcadia GDP and CCI indicators to be present")
	}
	if gdpPublished != quarterStart {
		t.Fatalf("expected GDP published at %d, got %d", quarterStart, gdpPublished)
	}
	if cciPublished != weekStart {
		t.Fatalf("expected CCI published at %d, got %d", weekStart, cciPublished)
	}
}

func TestMacroNewsVariablesIncludeUnemployment(t *testing.T) {
	indicator := MacroIndicator{
		Country: "Arcadia",
		Type:    macroTypeUnemp,
		Value:   425,
	}
	vars := macroNewsVariables(indicator, time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	if vars["unemployment"] != "4.25" {
		t.Fatalf("expected unemployment to be formatted, got %q", vars["unemployment"])
	}
}

func TestMacroGDPUsesEconomicTotals(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)

	store.mu.Lock()
	store.macroQuarterIndex = quarterIndex
	store.macroGDPPrevTotals["Arcadia"] = 1_000
	for _, state := range store.companyStates {
		state.Country = "Arcadia"
		state.LastB2CRevenue = 0
		state.LastB2GRevenue = 0
		state.LastCapexCost = 0
		state.LastInventoryChange = 0
	}
	for _, state := range store.companyStates {
		state.LastB2CRevenue = 600
		state.LastB2GRevenue = 200
		state.LastCapexCost = 150
		state.LastInventoryChange = 100
		break
	}
	store.refreshMacroIndicatorsLocked(now)
	indicators := append([]MacroIndicator(nil), store.macroIndicators...)
	store.mu.Unlock()

	var gdpIndicator MacroIndicator
	for _, indicator := range indicators {
		if indicator.Country == "Arcadia" && indicator.Type == macroTypeGDPGrowth {
			gdpIndicator = indicator
			break
		}
	}
	if gdpIndicator.Value != 500 {
		t.Fatalf("expected GDP growth basis 500, got %d", gdpIndicator.Value)
	}
}

func TestMacroCPIUsesPriceIndex(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)

	store.mu.Lock()
	store.macroQuarterIndex = quarterIndex
	store.macroCPIIndexPrev["Arcadia"] = 100
	for _, assetID := range []int64{101, 102, 103} {
		base := store.marketPriceLocked(assetID)
		if base > 0 {
			store.lastPrices[assetID] = base * 110 / 100
		}
	}
	store.refreshMacroIndicatorsLocked(now)
	indicators := append([]MacroIndicator(nil), store.macroIndicators...)
	store.mu.Unlock()

	var cpiIndicator MacroIndicator
	for _, indicator := range indicators {
		if indicator.Country == "Arcadia" && indicator.Type == macroTypeCPI {
			cpiIndicator = indicator
			break
		}
	}
	if cpiIndicator.Value != 1000 {
		t.Fatalf("expected CPI inflation basis 1000, got %d", cpiIndicator.Value)
	}
}

func TestMacroUnemploymentUsesUtilization(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)

	store.mu.Lock()
	store.macroQuarterIndex = quarterIndex
	for _, state := range store.companyStates {
		state.Country = "Arcadia"
		state.UtilizationRate = 8_000
	}
	store.refreshMacroIndicatorsLocked(now)
	indicators := append([]MacroIndicator(nil), store.macroIndicators...)
	store.mu.Unlock()

	var unempIndicator MacroIndicator
	for _, indicator := range indicators {
		if indicator.Country == "Arcadia" && indicator.Type == macroTypeUnemp {
			unempIndicator = indicator
			break
		}
	}
	if unempIndicator.Value != 462 {
		t.Fatalf("expected unemployment basis 462, got %d", unempIndicator.Value)
	}
}
