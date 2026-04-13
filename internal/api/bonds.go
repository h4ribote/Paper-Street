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
	ctx, cancel := s.dbContext()
	_ = s.queries.UpsertAsset(ctx, asset, s.fallbackBondPrice(asset.ID))
	cancel()

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
		if _, ok := s.Asset(assetID); ok {
			// In DB, base_price IS the base price.
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
		s.updateAssetLocked(models.Asset{ID: bond.AssetID}, price)
	}
	s.processPerpetualBondCouponsLocked(now)
}

func (s *MarketStore) processPerpetualBondCouponsLocked(now time.Time) []BondCouponPayment {
	if len(s.perpetualBonds) == 0 {
		return nil
	}
	// nowMillis := now.UnixMilli()
	// cutoff := nowMillis - int64(bondHoldDuration/time.Millisecond)
	payments := make([]BondCouponPayment, 0)
	for _, bond := range s.perpetualBonds {
		periodIndex := bondPeriodIndex(now, bond.PaymentFrequency)
		if lastIndex, ok := s.bondCouponIndex[bond.AssetID]; ok && lastIndex >= periodIndex {
			continue
		}
		var users []models.User
		if s.queries != nil {
			ctx, cancel := s.dbContext()
			users, _ = s.queries.ListUsers(ctx)
			cancel()
		} else {
			for _, u := range s.testUsers {
				users = append(users, u)
			}
		}
		for _, user := range users {
			qty := s.GetPosition(user.ID, bond.AssetID)
			if qty <= 0 {
				continue
			}
			acquiredAt := s.GetAssetAcquiredAt(user.ID, bond.AssetID)
			if acquiredAt > 0 {
				cutoff := now.UnixMilli() - int64(bondHoldDuration/time.Millisecond)
				if acquiredAt > cutoff {
					continue
				}
			}
			amount, ok := safeMultiplyInt64(qty, bond.BaseCoupon)
			if !ok || amount <= 0 {
				continue
			}
			_ = s.UpdateBalance(user.ID, currencyForCountry(bond.IssuerCountry, defaultCurrency), amount)
			payments = append(payments, BondCouponPayment{
				AssetID:  bond.AssetID,
				UserID:   user.ID,
				Quantity: qty,
				Currency: currencyForCountry(bond.IssuerCountry, defaultCurrency),
				Coupon:   bond.BaseCoupon,
				Amount:   amount,
			})
		}
		s.bondCouponIndex[bond.AssetID] = periodIndex
	}
	return payments
}

// Removed updateAssetAcquiredAtLocked as assetAcquiredAt map was removed.

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

	s.updateAssetLocked(asset, defaultAssetPrice)

	currency := currencyForCountry(def.IssuerCountry, defaultCurrency)
	if currency != "" {
		s.mu.Lock()
		s.currencies[currency] = struct{}{}
		s.mu.Unlock()
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
		if asset, ok := s.Asset(seed.Asset.ID); ok {
			seed.Asset.Type = "BOND"
			if seed.Asset.Symbol == "" {
				seed.Asset.Symbol = asset.Symbol
			}
			if seed.Asset.Name == "" {
				seed.Asset.Name = asset.Name
			}
		}

		if s.queries != nil {
			ctx, cancel := s.dbContext()
			_ = s.queries.UpsertAsset(ctx, seed.Asset, defaultAssetPrice)
			cancel()
		}
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
	_ = s.EnsureUser(userID)
	user, _ := s.User(userID)
	bank := centralBankForCountry(def.IssuerCountry, "")
	if bank != "" {
		user.Username = bank
	} else if user.Username == "" {
		user.Username = fmt.Sprintf("bond-issuer-%d", userID)
	}
	user.Role = "bot"
	ctx, cancel := s.dbContext()
	defer cancel()
	_ = s.queries.UpsertUser(ctx, user, time.Now().UTC())

	cash := s.GetBalance(userID, defaultCurrency)
	if cash < bondDefaultIssuerBuffer {
		_ = s.UpdateBalance(userID, defaultCurrency, bondDefaultIssuerBuffer-cash)
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

	s.publishNewsItemLocked(now, NewsItem{
		Headline: headline,
		AssetID:  asset.ID,
		Category: "CENTRAL_BANK",
		Impact:   "NEUTRAL",
	})
}
