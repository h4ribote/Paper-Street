package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	bondPaymentWeekly       = "WEEKLY"
	bondPaymentDaily        = "DAILY"
	bondHoldDuration        = 72 * time.Hour
	bondIssueDiscountBps    = int64(200)
	bondBuybackPremiumBps   = int64(200)
	bondDefaultIssueQty     = int64(1_000_000)
	bondDefaultBuybackQty   = int64(500_000)
	bondDefaultIssuerBuffer = int64(20_000_000_000)
)

const (
	bondArcadiaAssetID   = int64(301)
	bondBorosAssetID     = int64(302)
	bondSanVerdeAssetID  = int64(303)
	bondDefaultSector    = "BOND"
	bondDefaultShortName = "Consol"
)

type PerpetualBondDefinition struct {
	AssetID          int64
	IssuerCountry    string
	BaseCoupon       int64
	PaymentFrequency string
}

type PerpetualBondInfo struct {
	Asset            models.Asset `json:"asset"`
	IssuerCountry    string       `json:"issuer_country"`
	Currency         string       `json:"currency"`
	BaseCoupon       int64        `json:"base_coupon"`
	PaymentFrequency string       `json:"payment_frequency"`
	TargetYieldBps   int64        `json:"target_yield_bps"`
	TheoreticalPrice int64        `json:"theoretical_price"`
}

type BondCouponPayment struct {
	AssetID  int64  `json:"asset_id"`
	UserID   int64  `json:"user_id"`
	Quantity int64  `json:"quantity"`
	Currency string `json:"currency"`
	Coupon   int64  `json:"coupon"`
	Amount   int64  `json:"amount"`
}

type bondSeed struct {
	Asset            models.Asset
	IssuerCountry    string
	BaseCoupon       int64
	PaymentFrequency string
}

var defaultBondSeeds = []bondSeed{
	{
		Asset:            models.Asset{ID: bondArcadiaAssetID, Symbol: "ARCB", Name: "Arcadia Consol", Type: "BOND", Sector: bondDefaultSector},
		IssuerCountry:    "Arcadia",
		BaseCoupon:       250,
		PaymentFrequency: bondPaymentWeekly,
	},
	{
		Asset:            models.Asset{ID: bondBorosAssetID, Symbol: "BRSB", Name: "Boros Consol", Type: "BOND", Sector: bondDefaultSector},
		IssuerCountry:    "Boros Federation",
		BaseCoupon:       500,
		PaymentFrequency: bondPaymentWeekly,
	},
	{
		Asset:            models.Asset{ID: bondSanVerdeAssetID, Symbol: "SVDB", Name: "San Verde Consol", Type: "BOND", Sector: bondDefaultSector},
		IssuerCountry:    "San Verde",
		BaseCoupon:       1000,
		PaymentFrequency: bondPaymentWeekly,
	},
}

func normalizeBondFrequency(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case bondPaymentDaily:
		return bondPaymentDaily
	default:
		return bondPaymentWeekly
	}
}

func bondPeriodIndex(now time.Time, frequency string) int64 {
	switch normalizeBondFrequency(frequency) {
	case bondPaymentDaily:
		return macroPeriodIndex(now, 24*time.Hour)
	default:
		return macroPeriodIndex(now, macroWeekPeriod)
	}
}

func (s *MarketStore) PerpetualBonds() []PerpetualBondInfo {
	now := time.Now().UTC()
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)
	weekIndex := macroPeriodIndex(now, macroWeekPeriod)
	s.mu.Lock()
	if s.macroQuarterIndex != quarterIndex || s.macroWeekIndex != weekIndex || len(s.macroIndicators) == 0 {
		s.refreshMacroIndicatorsLocked(now)
	}
	bonds := make([]PerpetualBondInfo, 0, len(s.perpetualBonds))
	for _, bond := range s.perpetualBonds {
		bonds = append(bonds, s.perpetualBondInfoLocked(bond))
	}
	sort.Slice(bonds, func(i, j int) bool { return bonds[i].Asset.ID < bonds[j].Asset.ID })
	s.mu.Unlock()
	return bonds
}

func (s *MarketStore) PerpetualBond(assetID int64) (PerpetualBondInfo, bool) {
	now := time.Now().UTC()
	quarterIndex := macroPeriodIndex(now, macroQuarterPeriod)
	weekIndex := macroPeriodIndex(now, macroWeekPeriod)
	s.mu.Lock()
	if s.macroQuarterIndex != quarterIndex || s.macroWeekIndex != weekIndex || len(s.macroIndicators) == 0 {
		s.refreshMacroIndicatorsLocked(now)
	}
	bond, ok := s.perpetualBonds[assetID]
	if !ok {
		s.mu.Unlock()
		return PerpetualBondInfo{}, false
	}
	info := s.perpetualBondInfoLocked(bond)
	s.mu.Unlock()
	return info, true
}

func (s *MarketStore) TriggerPerpetualBondCoupons(now time.Time) []BondCouponPayment {
	s.mu.Lock()
	payments := s.processPerpetualBondCouponsLocked(now)
	s.mu.Unlock()
	return payments
}

func (s *MarketStore) perpetualBondInfoLocked(def PerpetualBondDefinition) PerpetualBondInfo {
	asset := s.ensureAssetLocked(def.AssetID)
	asset.Type = "BOND"
	if asset.Sector == "" {
		asset.Sector = bondDefaultSector
	}
	s.assets[asset.ID] = asset
	currency := currencyForCountry(def.IssuerCountry, defaultCurrency)
	targetYield := s.bondTargetYieldBpsLocked(def)
	return PerpetualBondInfo{
		Asset:            asset,
		IssuerCountry:    def.IssuerCountry,
		Currency:         currency,
		BaseCoupon:       def.BaseCoupon,
		PaymentFrequency: normalizeBondFrequency(def.PaymentFrequency),
		TargetYieldBps:   targetYield,
		TheoreticalPrice: s.bondTheoreticalPriceLocked(def),
	}
}

func (s *MarketStore) bondTargetYieldBpsLocked(def PerpetualBondDefinition) int64 {
	values, ok := s.macroIndicatorValuesLocked(def.IssuerCountry)
	if !ok {
		return 0
	}
	return values.rate
}

func (s *MarketStore) bondTheoreticalPriceLocked(def PerpetualBondDefinition) int64 {
	if def.BaseCoupon <= 0 {
		return s.fallbackBondPrice(def.AssetID)
	}
	rate := s.bondTargetYieldBpsLocked(def)
	if rate <= 0 {
		return s.fallbackBondPrice(def.AssetID)
	}
	product, ok := safeMultiplyInt64(def.BaseCoupon, bpsDenominator)
	if !ok || product <= 0 {
		return s.fallbackBondPrice(def.AssetID)
	}
	price := product / rate
	if price <= 0 {
		return s.fallbackBondPrice(def.AssetID)
	}
	return price
}

func (s *MarketStore) fallbackBondPrice(assetID int64) int64 {
	if assetID != 0 {
		if base := s.basePrices[assetID]; base > 0 {
			return base
		}
	}
	return defaultAssetPrice
}

func (s *MarketStore) refreshPerpetualBondPricingLocked(now time.Time) {
	if len(s.perpetualBonds) == 0 {
		return
	}
	for _, bond := range s.perpetualBonds {
		price := s.bondTheoreticalPriceLocked(bond)
		if price <= 0 {
			price = defaultAssetPrice
		}
		s.basePrices[bond.AssetID] = price
	}
	s.processPerpetualBondCouponsLocked(now)
}

func (s *MarketStore) processPerpetualBondCouponsLocked(now time.Time) []BondCouponPayment {
	if len(s.perpetualBonds) == 0 {
		return nil
	}
	nowMillis := now.UnixMilli()
	cutoff := nowMillis - int64(bondHoldDuration/time.Millisecond)
	payments := make([]BondCouponPayment, 0)
	for _, bond := range s.perpetualBonds {
		periodIndex := bondPeriodIndex(now, bond.PaymentFrequency)
		if lastIndex, ok := s.bondCouponIndex[bond.AssetID]; ok && lastIndex >= periodIndex {
			continue
		}
		for userID, holdings := range s.positions {
			qty := holdings[bond.AssetID]
			if qty <= 0 {
				continue
			}
			acquiredAt := s.assetAcquiredAt[userID][bond.AssetID]
			if acquiredAt == 0 || acquiredAt > cutoff {
				continue
			}
			amount, ok := safeMultiplyInt64(qty, bond.BaseCoupon)
			if !ok || amount <= 0 {
				continue
			}
			s.ensureUserLocked(userID)
			currency := currencyForCountry(bond.IssuerCountry, defaultCurrency)
			if currency == "" {
				currency = defaultCurrency
			}
			if _, ok := s.balances[userID][currency]; !ok {
				s.balances[userID][currency] = 0
			}
			s.balances[userID][currency] += amount
			payments = append(payments, BondCouponPayment{
				AssetID:  bond.AssetID,
				UserID:   userID,
				Quantity: qty,
				Currency: currency,
				Coupon:   bond.BaseCoupon,
				Amount:   amount,
			})
		}
		s.bondCouponIndex[bond.AssetID] = periodIndex
	}
	return payments
}

func (s *MarketStore) updateAssetAcquiredAtLocked(userID, assetID, oldQty, deltaQty, acquiredAt int64) {
	if userID == 0 || assetID == 0 {
		return
	}
	if _, ok := s.assetAcquiredAt[userID]; !ok {
		s.assetAcquiredAt[userID] = make(map[int64]int64)
	}
	newQty := oldQty + deltaQty
	if newQty <= 0 {
		delete(s.assetAcquiredAt[userID], assetID)
		return
	}
	if deltaQty <= 0 {
		if oldQty <= 0 {
			s.assetAcquiredAt[userID][assetID] = acquiredAt
		}
		return
	}
	if oldQty <= 0 {
		s.assetAcquiredAt[userID][assetID] = acquiredAt
		return
	}
	oldAvg := s.assetAcquiredAt[userID][assetID]
	if oldAvg <= 0 {
		s.assetAcquiredAt[userID][assetID] = acquiredAt
		return
	}
	weightedOld, ok := safeMultiplyInt64(oldAvg, oldQty)
	if !ok {
		s.assetAcquiredAt[userID][assetID] = acquiredAt
		return
	}
	weightedNew, ok := safeMultiplyInt64(acquiredAt, deltaQty)
	if !ok {
		s.assetAcquiredAt[userID][assetID] = acquiredAt
		return
	}
	total, ok := safeAddInt64(weightedOld, weightedNew)
	if !ok || total <= 0 {
		s.assetAcquiredAt[userID][assetID] = acquiredAt
		return
	}
	s.assetAcquiredAt[userID][assetID] = total / newQty
}

func (s *MarketStore) registerPerpetualBondLocked(def PerpetualBondDefinition, now time.Time) {
	def.PaymentFrequency = normalizeBondFrequency(def.PaymentFrequency)
	if def.IssuerCountry == "" {
		def.IssuerCountry = fxArcadiaCountry
	}
	s.perpetualBonds[def.AssetID] = def
	asset := s.ensureAssetLocked(def.AssetID)
	if asset.Type == "" || asset.Type == "STOCK" {
		asset.Type = "BOND"
	}
	if asset.Sector == "" {
		asset.Sector = bondDefaultSector
	}
	if asset.Name == "" {
		asset.Name = fmt.Sprintf("%s %s", def.IssuerCountry, bondDefaultShortName)
	}
	if asset.Symbol == "" || strings.HasPrefix(asset.Symbol, "ASSET-") {
		asset.Symbol = fmt.Sprintf("BOND-%d", def.AssetID)
	}
	s.assets[asset.ID] = asset
	if _, ok := s.basePrices[def.AssetID]; !ok {
		s.basePrices[def.AssetID] = defaultAssetPrice
	}
	currency := currencyForCountry(def.IssuerCountry, defaultCurrency)
	if currency != "" {
		s.currencies[currency] = struct{}{}
	}
	if _, ok := s.bondCouponIndex[def.AssetID]; !ok {
		s.bondCouponIndex[def.AssetID] = bondPeriodIndex(now, def.PaymentFrequency)
	}
}

func (s *MarketStore) seedPerpetualBonds(now time.Time) {
	for _, seed := range defaultBondSeeds {
		if existing, ok := s.perpetualBonds[seed.Asset.ID]; ok {
			s.registerPerpetualBondLocked(existing, now)
			continue
		}
		if asset, ok := s.assets[seed.Asset.ID]; ok {
			seed.Asset.Type = "BOND"
			if seed.Asset.Symbol == "" {
				seed.Asset.Symbol = asset.Symbol
			}
			if seed.Asset.Name == "" {
				seed.Asset.Name = asset.Name
			}
		}
		s.assets[seed.Asset.ID] = seed.Asset
		def := PerpetualBondDefinition{
			AssetID:          seed.Asset.ID,
			IssuerCountry:    seed.IssuerCountry,
			BaseCoupon:       seed.BaseCoupon,
			PaymentFrequency: seed.PaymentFrequency,
		}
		s.registerPerpetualBondLocked(def, now)
	}
}

func (s *MarketStore) ensureBondIssuerLocked(def PerpetualBondDefinition) int64 {
	userID := def.AssetID
	user := s.ensureUserLocked(userID)
	bank := centralBankForCountry(def.IssuerCountry, "")
	if bank != "" {
		user.Username = bank
	} else if user.Username == "" {
		user.Username = fmt.Sprintf("bond-issuer-%d", userID)
	}
	user.Role = "bot"
	s.users[userID] = user
	cash := s.balances[userID][defaultCurrency]
	if cash < bondDefaultIssuerBuffer {
		s.balances[userID][defaultCurrency] = bondDefaultIssuerBuffer
	}
	return userID
}

func (s *MarketStore) bondOperationPriceLocked(def PerpetualBondDefinition, premiumBps, discountBps int64) (int64, int64) {
	price := s.bondTheoreticalPriceLocked(def)
	if price <= 0 {
		price = defaultAssetPrice
	}
	adjusted := applyBps(price, premiumBps, discountBps)
	return adjusted, s.bondTargetYieldBpsLocked(def)
}

func (s *MarketStore) bondOperationNewsLocked(def PerpetualBondDefinition, action string, quantity, price int64, now time.Time) {
	asset := s.ensureAssetLocked(def.AssetID)
	if asset.Symbol == "" {
		return
	}
	bank := centralBankForCountry(def.IssuerCountry, "Central Bank")
	var headline string
	switch action {
	case "issue":
		headline = fmt.Sprintf("[MACRO] %s to issue %d units of %s consol bonds at %d.", bank, quantity, asset.Symbol, price)
	case "buyback":
		headline = fmt.Sprintf("[MACRO] %s to buy back %d units of %s consol bonds at %d.", bank, quantity, asset.Symbol, price)
	default:
		return
	}
	s.nextNewsID++
	s.news = append(s.news, NewsItem{
		ID:          s.nextNewsID,
		Headline:    headline,
		Impact:      "NEUTRAL",
		AssetID:     asset.ID,
		Category:    "CENTRAL_BANK",
		PublishedAt: now.UnixMilli(),
	})
}
