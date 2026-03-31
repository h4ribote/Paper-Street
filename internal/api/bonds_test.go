package api

import (
	"testing"
	"time"
)

func TestBondTheoreticalPriceUsesInterestRate(t *testing.T) {
	store := NewMarketStore()
	store.mu.Lock()
	def, ok := store.perpetualBonds[bondArcadiaAssetID]
	if !ok {
		store.mu.Unlock()
		t.Fatalf("expected Arcadia bond definition")
	}
	store.macroIndicators = []MacroIndicator{
		{Country: def.IssuerCountry, Type: macroTypeGDPGrowth, Value: 200},
		{Country: def.IssuerCountry, Type: macroTypeCPI, Value: 200},
		{Country: def.IssuerCountry, Type: macroTypeInterest, Value: 250},
	}
	info := store.perpetualBondInfoLocked(def)
	store.mu.Unlock()
	if info.TheoreticalPrice != 10_000 {
		t.Fatalf("expected theoretical price 10000, got %d", info.TheoreticalPrice)
	}
}

func TestBondCouponPaymentsRespectEligibility(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	store.mu.Lock()
	def, ok := store.perpetualBonds[bondArcadiaAssetID]
	if !ok {
		store.mu.Unlock()
		t.Fatalf("expected Arcadia bond definition")
	}
	eligibleUser := int64(9001)
	ineligibleUser := int64(9002)
	store.ensureUserLocked(eligibleUser)
	store.ensureUserLocked(ineligibleUser)
	store.positions[eligibleUser][bondArcadiaAssetID] = 10
	store.positions[ineligibleUser][bondArcadiaAssetID] = 5
	store.assetAcquiredAt[eligibleUser] = map[int64]int64{
		bondArcadiaAssetID: now.Add(-bondHoldDuration - time.Hour).UnixMilli(),
	}
	store.assetAcquiredAt[ineligibleUser] = map[int64]int64{
		bondArcadiaAssetID: now.Add(-bondHoldDuration + time.Hour).UnixMilli(),
	}
	store.bondCouponIndex[bondArcadiaAssetID] = bondPeriodIndex(now.Add(-macroWeekPeriod), def.PaymentFrequency)
	currency := currencyForCountry(def.IssuerCountry, defaultCurrency)
	startEligible := store.balances[eligibleUser][currency]
	startIneligible := store.balances[ineligibleUser][currency]
	payments := store.processPerpetualBondCouponsLocked(now)
	store.mu.Unlock()
	if len(payments) != 1 {
		t.Fatalf("expected 1 coupon payment, got %d", len(payments))
	}
	expected := int64(10) * def.BaseCoupon
	if got := payments[0].Amount; got != expected {
		t.Fatalf("expected coupon amount %d, got %d", expected, got)
	}
	if got := payments[0].UserID; got != eligibleUser {
		t.Fatalf("expected eligible user payment, got %d", got)
	}
	if store.balances[eligibleUser][currency] != startEligible+expected {
		t.Fatalf("expected eligible balance to increase by %d", expected)
	}
	if store.balances[ineligibleUser][currency] != startIneligible {
		t.Fatalf("expected ineligible balance to remain %d", startIneligible)
	}
}
