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
