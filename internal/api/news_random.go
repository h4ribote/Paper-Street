package api

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/models"
)

var (
	newsCities      = []string{"Neo Venice", "Arcadia City", "Boros Prime", "Aurora Harbor", "San Verde Bay"}
	newsCommodities = []string{"rare earths", "fusion fuel", "quantum chips", "alloy steel", "agri staples"}
	newsConcerns    = []string{"inflation fears", "supply bottlenecks", "demand softness", "geopolitical risk"}
	newsEvents      = []string{"Aurora Summit", "Arcadia Tech Expo", "Boros Energy Forum", "Neo Venice FinTech Week"}
	newsGroups      = []string{"Arcadian Trade Bloc", "Aurora Logistics Guild", "Boros Energy Council", "Neo Venice Syndicate"}
	newsReasons     = []string{"strong demand", "policy tailwinds", "breakthrough innovation", "renewed investor optimism"}
	newsTech        = []string{"quantum processors", "fusion grids", "AI logistics", "nanoforge systems"}
	newsTrends      = []string{"green transition", "supply chain reshoring", "AI adoption wave", "clean energy boom"}
	newsNames       = []string{"Alex Rivera", "Samira Chen", "Luca Moretti", "Priya Nandakumar"}
)

const randomChoiceAttempts = 5

func (s *MarketStore) randomNewsItem(now time.Time, rng *rand.Rand) (NewsItem, bool) {
	library, err := loadNewsPatterns()
	if err != nil || library == nil {
		return NewsItem{}, false
	}
	rng = ensureRand(rng)
	keys := make([]string, 0, len(library.Categories))
	for key := range library.Categories {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return NewsItem{}, false
	}
	category := library.Categories[keys[rng.Intn(len(keys))]]
	if len(category.Patterns) == 0 {
		return NewsItem{}, false
	}
	pattern := category.Patterns[rng.Intn(len(category.Patterns))]

	s.mu.RLock()
	assets := make([]models.Asset, 0, len(s.assets))
	for _, asset := range s.assets {
		assets = append(assets, asset)
	}
	basePrices := make(map[int64]int64, len(s.basePrices))
	for id, price := range s.basePrices {
		basePrices[id] = price
	}
	indicators := make([]MacroIndicator, len(s.macroIndicators))
	copy(indicators, s.macroIndicators)
	s.mu.RUnlock()

	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	sort.Slice(indicators, func(i, j int) bool {
		if indicators[i].Country == indicators[j].Country {
			return indicators[i].Type < indicators[j].Type
		}
		return indicators[i].Country < indicators[j].Country
	})

	asset := randomAsset(rng, assets)
	indicator := randomMacroIndicator(rng, indicators)
	vars, assetID := s.newsVariablesForPattern(strings.ToUpper(category.ID), pattern, asset, indicator, basePrices, now, rng)
	sentiment := randomSentiment(pattern, rng)
	return buildNewsItemWithSentiment(category.ID, assetID, pattern, vars, sentiment), true
}

func buildNewsItemWithSentiment(category string, assetID int64, pattern newsPattern, variables map[string]string, sentiment float64) NewsItem {
	headlineTemplate, bodyTemplate := selectNewsTemplates(pattern)
	headline := fillNewsTemplate(headlineTemplate, variables)
	body := fillNewsTemplate(bodyTemplate, variables)
	return NewsItem{
		Headline:    headline,
		Body:        body,
		Impact:      sentimentImpact(sentiment),
		AssetID:     assetID,
		Category:    category,
		Sentiment:   sentiment,
		ImpactScope: fillImpactScope(pattern.ImpactScope, variables),
	}
}

func randomSentiment(pattern newsPattern, rng *rand.Rand) float64 {
	if len(pattern.SentimentRange) < 2 {
		return 0
	}
	min := pattern.SentimentRange[0]
	max := pattern.SentimentRange[1]
	if min > max {
		min, max = max, min
	}
	rng = ensureRand(rng)
	return min + rng.Float64()*(max-min)
}

func (s *MarketStore) newsVariablesForPattern(categoryID string, pattern newsPattern, asset models.Asset, indicator MacroIndicator, basePrices map[int64]int64, now time.Time, rng *rand.Rand) (map[string]string, int64) {
	var vars map[string]string
	assetID := int64(0)
	switch categoryID {
	case "EARNINGS":
		if asset.ID != 0 {
			vars = assetNewsVariables(asset, basePrices[asset.ID], now)
			assetID = asset.ID
		}
	case "MACRO", "CENTRAL_BANK":
		if indicator.Type != "" {
			vars = macroNewsVariables(indicator, now)
		}
	}
	if vars == nil {
		if asset.ID != 0 {
			vars = assetNewsVariables(asset, basePrices[asset.ID], now)
		} else {
			vars = defaultNewsVariables(now)
		}
	}
	vars = fillNewsRandomExtras(vars, asset, indicator, basePrices, rng)
	if asset.ID != 0 && assetID == 0 {
		if categoryID != "MACRO" && categoryID != "CENTRAL_BANK" &&
			(patternHasVariable(pattern, "ticker") || patternHasVariable(pattern, "company_name")) {
			assetID = asset.ID
		}
	}
	return vars, assetID
}

func fillNewsRandomExtras(vars map[string]string, asset models.Asset, indicator MacroIndicator, basePrices map[int64]int64, rng *rand.Rand) map[string]string {
	if vars == nil {
		vars = map[string]string{}
	}
	rng = ensureRand(rng)
	if asset.ID != 0 {
		if vars["company_name"] == "" {
			vars["company_name"] = asset.Name
		}
		if vars["ticker"] == "" {
			vars["ticker"] = asset.Symbol
		}
		if vars["sector"] == "" {
			vars["sector"] = asset.Sector
		}
		if vars["company_lead"] == "" {
			vars["company_lead"] = asset.Name
		}
	}
	countries := macroCountryNames()
	if vars["country"] == "" {
		if indicator.Country != "" {
			vars["country"] = indicator.Country
		} else {
			vars["country"] = randomChoiceString(rng, countries)
		}
	}
	if vars["country_a"] == "" {
		vars["country_a"] = randomChoiceString(rng, countries)
	}
	if vars["country_b"] == "" {
		vars["country_b"] = randomChoiceDifferent(rng, countries, vars["country_a"])
	}
	if vars["currency"] == "" {
		vars["currency"] = currencyForCountry(vars["country"], defaultCurrency)
	}
	if vars["currency_a"] == "" {
		vars["currency_a"] = currencyForCountry(vars["country_a"], vars["currency"])
	}
	if vars["currency_b"] == "" {
		vars["currency_b"] = currencyForCountry(vars["country_b"], vars["currency"])
	}
	if vars["city"] == "" {
		if vars["location"] != "" {
			vars["city"] = vars["location"]
		} else {
			vars["city"] = randomChoiceString(rng, newsCities)
		}
	}
	if vars["commodity"] == "" {
		vars["commodity"] = randomChoiceString(rng, newsCommodities)
	}
	if vars["reason"] == "" {
		vars["reason"] = randomChoiceString(rng, newsReasons)
	}
	if vars["trend_name"] == "" {
		vars["trend_name"] = randomChoiceString(rng, newsTrends)
	}
	if vars["technology"] == "" {
		vars["technology"] = randomChoiceString(rng, newsTech)
	}
	if vars["concern"] == "" {
		vars["concern"] = randomChoiceString(rng, newsConcerns)
	}
	if vars["event_name"] == "" {
		vars["event_name"] = randomChoiceString(rng, newsEvents)
	}
	if vars["group_name"] == "" {
		vars["group_name"] = randomChoiceString(rng, newsGroups)
	}
	if vars["percent"] == "" {
		vars["percent"] = fmt.Sprintf("%.1f", 1+rng.Float64()*9)
	}
	if vars["price_target"] == "" {
		basePrice := basePrices[asset.ID]
		if basePrice <= 0 {
			basePrice = defaultAssetPrice
		}
		price := float64(basePrice) / 100
		vars["price_target"] = fmt.Sprintf("%.0f", price*(1+rng.Float64()*0.2))
	}
	if vars["name"] == "" {
		if vars["person_name"] != "" {
			vars["name"] = vars["person_name"]
		} else {
			vars["name"] = randomChoiceString(rng, newsNames)
		}
	}
	return vars
}

func randomAsset(rng *rand.Rand, assets []models.Asset) models.Asset {
	if len(assets) == 0 {
		return models.Asset{}
	}
	rng = ensureRand(rng)
	return assets[rng.Intn(len(assets))]
}

func randomMacroIndicator(rng *rand.Rand, indicators []MacroIndicator) MacroIndicator {
	if len(indicators) == 0 {
		return MacroIndicator{}
	}
	rng = ensureRand(rng)
	return indicators[rng.Intn(len(indicators))]
}

func randomChoiceString(rng *rand.Rand, values []string) string {
	if len(values) == 0 {
		return ""
	}
	rng = ensureRand(rng)
	return values[rng.Intn(len(values))]
}

func randomChoiceDifferent(rng *rand.Rand, values []string, exclude string) string {
	if len(values) == 0 {
		return ""
	}
	rng = ensureRand(rng)
	if len(values) == 1 {
		return values[0]
	}
	for i := 0; i < randomChoiceAttempts; i++ {
		value := values[rng.Intn(len(values))]
		if !stringsEqualFold(value, exclude) {
			return value
		}
	}
	for _, value := range values {
		if !stringsEqualFold(value, exclude) {
			return value
		}
	}
	return values[0]
}

func macroCountryNames() []string {
	countries := make([]string, 0, len(macroProfiles))
	for _, profile := range macroProfiles {
		if strings.TrimSpace(profile.Country) == "" {
			continue
		}
		countries = append(countries, profile.Country)
	}
	sort.Strings(countries)
	return countries
}

func ensureRand(rng *rand.Rand) *rand.Rand {
	if rng != nil {
		return rng
	}
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}
