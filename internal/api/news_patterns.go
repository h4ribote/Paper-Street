package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/h4ribote/Paper-Street/internal/models"
)

type newsPatternLibrary struct {
	Version    string                         `json:"version"`
	Meta       newsPatternMeta                `json:"meta"`
	Categories map[string]newsPatternCategory `json:"categories"`
}

type newsPatternMeta struct {
	Languages []string `json:"languages"`
}

type newsPatternCategory struct {
	ID          string        `json:"id"`
	NameEn      string        `json:"name_en"`
	NameJa      string        `json:"name_ja"`
	Description string        `json:"description"`
	Patterns    []newsPattern `json:"patterns"`
}

type newsPattern struct {
	ID                 string    `json:"id"`
	HeadlineTemplateEn string    `json:"headline_template_en"`
	HeadlineTemplateJa string    `json:"headline_template_ja"`
	BodyTemplateEn     string    `json:"body_template_en"`
	BodyTemplateJa     string    `json:"body_template_ja"`
	SentimentRange     []float64 `json:"sentiment_range"`
	ImpactScope        []string  `json:"impact_scope"`
	Variables          []string  `json:"variables"`
}

var (
	newsPatternsOnce sync.Once
	newsPatterns     *newsPatternLibrary
	newsPatternsErr  error
	newsPlaceholder  = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)
)

func loadNewsPatterns() (*newsPatternLibrary, error) {
	newsPatternsOnce.Do(func() {
		path, err := resolveNewsPatternsPath()
		if err != nil {
			newsPatternsErr = err
			return
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			newsPatternsErr = err
			return
		}
		var library newsPatternLibrary
		if err := json.Unmarshal(payload, &library); err != nil {
			newsPatternsErr = err
			return
		}
		if len(library.Categories) == 0 {
			newsPatternsErr = fmt.Errorf("news patterns missing categories")
			return
		}
		newsPatterns = &library
	})
	if newsPatterns == nil {
		return nil, newsPatternsErr
	}
	return newsPatterns, newsPatternsErr
}

func resolveNewsPatternsPath() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to resolve news pattern path")
	}
	baseDir := filepath.Dir(filename)
	return filepath.Clean(filepath.Join(baseDir, "..", "..", "docs", "news_patterns.json")), nil
}

func (s *MarketStore) generatePatternNews(now time.Time) []NewsItem {
	library, err := loadNewsPatterns()
	if err != nil || library == nil {
		return nil
	}
	var items []NewsItem
	earningsCategory, ok := library.Categories["EARNINGS"]
	if ok {
		assets := s.Assets(AssetFilter{})
		for _, asset := range assets {
			pattern := choosePattern(earningsCategory.Patterns, int(asset.ID))
			if pattern == nil {
				continue
			}
			vars := assetNewsVariables(asset, s.marketPriceLocked(asset.ID), now)
			items = append(items, buildNewsItem(earningsCategory.ID, asset.ID, *pattern, vars))
		}
	}
	if len(s.macroIndicators) == 0 {
		return items
	}
	indicators := make([]MacroIndicator, len(s.macroIndicators))
	copy(indicators, s.macroIndicators)
	sort.Slice(indicators, func(i, j int) bool {
		if indicators[i].Country == indicators[j].Country {
			return indicators[i].Type < indicators[j].Type
		}
		return indicators[i].Country < indicators[j].Country
	})
	for idx, indicator := range indicators {
		categoryID := macroCategoryForIndicator(indicator.Type)
		category, ok := library.Categories[categoryID]
		if !ok {
			continue
		}
		pattern := chooseMacroPattern(category.Patterns, indicator.Type, idx)
		if pattern == nil {
			continue
		}
		vars := macroNewsVariables(indicator, now)
		items = append(items, buildNewsItem(category.ID, 0, *pattern, vars))
	}
	return items
}

func sortedAssets(assets map[int64]models.Asset) []models.Asset {
	list := make([]models.Asset, 0, len(assets))
	for _, asset := range assets {
		list = append(list, asset)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list
}

func choosePattern(patterns []newsPattern, seed int) *newsPattern {
	if len(patterns) == 0 {
		return nil
	}
	index := seed % len(patterns)
	if index < 0 {
		index = -index
	}
	return &patterns[index]
}

func chooseMacroPattern(patterns []newsPattern, indicatorType string, seed int) *newsPattern {
	if len(patterns) == 0 {
		return nil
	}
	indicatorType = strings.ToUpper(strings.TrimSpace(indicatorType))
	switch indicatorType {
	case "GDP_GROWTH", "CPI", "UNEMPLOYMENT", "UNEMPLOYMENT_RATE", "UER", "INTEREST_RATE":
	default:
		return nil
	}
	for _, pattern := range patterns {
		if indicatorMatchesPattern(indicatorType, pattern) {
			return &pattern
		}
	}
	return choosePattern(patterns, seed)
}

func indicatorMatchesPattern(indicatorType string, pattern newsPattern) bool {
	switch indicatorType {
	case "GDP_GROWTH":
		return patternHasVariable(pattern, "gdp_growth")
	case "CPI":
		return patternHasVariable(pattern, "cpi")
	case "UNEMPLOYMENT", "UNEMPLOYMENT_RATE", "UER":
		return patternHasVariable(pattern, "unemployment")
	case "INTEREST_RATE":
		return patternHasVariable(pattern, "rate")
	default:
		return false
	}
}

func patternHasVariable(pattern newsPattern, variable string) bool {
	for _, name := range pattern.Variables {
		if stringsEqualFold(name, variable) {
			return true
		}
	}
	return false
}

func macroCategoryForIndicator(indicatorType string) string {
	switch strings.ToUpper(strings.TrimSpace(indicatorType)) {
	case "INTEREST_RATE":
		return "CENTRAL_BANK"
	default:
		return "MACRO"
	}
}

func buildNewsItem(category string, assetID int64, pattern newsPattern, variables map[string]string) NewsItem {
	headlineTemplate, bodyTemplate := selectNewsTemplates(pattern)
	headline := fillNewsTemplate(headlineTemplate, variables)
	body := fillNewsTemplate(bodyTemplate, variables)
	sentiment := patternSentiment(pattern)
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

func selectNewsTemplates(pattern newsPattern) (string, string) {
	headline := strings.TrimSpace(pattern.HeadlineTemplateJa)
	if headline == "" {
		headline = strings.TrimSpace(pattern.HeadlineTemplateEn)
	}
	body := strings.TrimSpace(pattern.BodyTemplateJa)
	if body == "" {
		body = strings.TrimSpace(pattern.BodyTemplateEn)
	}
	return headline, body
}

func fillNewsTemplate(template string, variables map[string]string) string {
	if template == "" {
		return ""
	}
	return newsPlaceholder.ReplaceAllStringFunc(template, func(match string) string {
		key := match[1 : len(match)-1]
		if value, ok := variables[key]; ok && strings.TrimSpace(value) != "" {
			return value
		}
		return "N/A"
	})
}

func fillImpactScope(scopes []string, variables map[string]string) []string {
	if len(scopes) == 0 {
		return nil
	}
	filled := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		value := fillNewsTemplate(scope, variables)
		if strings.TrimSpace(value) == "" {
			continue
		}
		filled = append(filled, value)
	}
	return filled
}

func patternSentiment(pattern newsPattern) float64 {
	if len(pattern.SentimentRange) < 2 {
		return 0
	}
	return (pattern.SentimentRange[0] + pattern.SentimentRange[1]) / 2
}

func sentimentImpact(sentiment float64) string {
	switch {
	case sentiment >= 0.05:
		return "POSITIVE"
	case sentiment <= -0.05:
		return "NEGATIVE"
	default:
		return "NEUTRAL"
	}
}

func assetNewsVariables(asset models.Asset, basePrice int64, now time.Time) map[string]string {
	vars := defaultNewsVariables(now)
	vars["company_name"] = stringOrDefault(asset.Name, vars["company_name"])
	vars["ticker"] = stringOrDefault(asset.Symbol, vars["ticker"])
	vars["sector"] = stringOrDefault(asset.Sector, vars["sector"])
	if basePrice <= 0 {
		basePrice = defaultAssetPrice
	}
	price := float64(basePrice) / 100
	eps := price / 50
	revenue := price / 5
	vars["eps"] = fmt.Sprintf("%.2f", eps)
	vars["eps_est"] = fmt.Sprintf("%.2f", eps*0.96)
	vars["revenue"] = fmt.Sprintf("%.1f", revenue)
	vars["revenue_est"] = fmt.Sprintf("%.1f", revenue*0.94)
	vars["product"] = productForSector(vars["sector"])
	return vars
}

func macroNewsVariables(indicator MacroIndicator, now time.Time) map[string]string {
	vars := defaultNewsVariables(now)
	vars["country"] = stringOrDefault(indicator.Country, vars["country"])
	vars["currency"] = currencyForCountry(indicator.Country, vars["currency"])
	switch strings.ToUpper(strings.TrimSpace(indicator.Type)) {
	case "GDP_GROWTH":
		vars["gdp_growth"] = formatMacroPercent(indicator.Value)
		vars["gdp_est"] = formatMacroPercent(maxInt64(indicator.Value-20, 0))
	case "CPI":
		vars["cpi"] = formatMacroPercent(indicator.Value)
	case "UNEMPLOYMENT", "UNEMPLOYMENT_RATE", "UER":
		vars["unemployment"] = formatMacroPercent(indicator.Value)
	case "INTEREST_RATE":
		vars["rate"] = formatMacroPercent(indicator.Value)
		vars["basis_points"] = "25"
		vars["central_bank"] = centralBankForCountry(indicator.Country, vars["central_bank"])
	}
	return vars
}

func defaultNewsVariables(now time.Time) map[string]string {
	quarter := (int(now.Month())-1)/3 + 1
	return map[string]string{
		"company_name": "Omni Dynamics",
		"ticker":       "OMNI",
		"sector":       "TECH",
		"quarter":      fmt.Sprintf("%d", quarter),
		"eps":          "2.50",
		"eps_est":      "2.35",
		"revenue":      "12.0",
		"revenue_est":  "11.3",
		"product":      "AI platforms",
		"country":      "Arcadia",
		"currency":     "ARC",
		"gdp_growth":   "3.1",
		"gdp_est":      "2.9",
		"cpi":          "2.1",
		"unemployment": "4.2",
		"central_bank": "Bank of Arcadia",
		"rate":         "4.75",
		"basis_points": "25",
		"person_name":  "Alex Rivera",
		"location":     "Neo Venice",
	}
}

func formatMacroPercent(value int64) string {
	return fmt.Sprintf("%.2f", float64(value)/100)
}

func productForSector(sector string) string {
	switch strings.ToUpper(strings.TrimSpace(sector)) {
	case "TECH":
		return "AI platforms"
	case "ENERGY":
		return "fusion output"
	case "METAL":
		return "rare earth alloys"
	default:
		return "core products"
	}
}

func currencyForCountry(country string, fallback string) string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "ARCADIA":
		return "ARC"
	case "NEO VENICE":
		return "VND"
	case "BOROS FEDERATION":
		return "BRB"
	case "EL DORADO":
		return "DRL"
	case "SAN VERDE":
		return "VDP"
	case "NOVAYA ZEMLYA":
		return "ZMR"
	case "PEARL RIVER ZONE":
		return "RVD"
	default:
		return fallback
	}
}

func centralBankForCountry(country, fallback string) string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "ARCADIA":
		return "Bank of Arcadia"
	case "NEO VENICE":
		return "Venice Monetary Authority"
	case "BOROS FEDERATION":
		return "Boros Central Bank"
	case "EL DORADO":
		return "El Dorado Reserve"
	case "SAN VERDE":
		return "San Verde Monetary Council"
	case "NOVAYA ZEMLYA":
		return "Zemlya Central Bank"
	case "PEARL RIVER ZONE":
		return "Pearl River Authority"
	default:
		return fallback
	}
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
