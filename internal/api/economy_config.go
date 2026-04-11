package api

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type EconomyConfig struct {
	MarketMakerSharePercent  int64               `yaml:"market_maker_share_percent"`
	LiquiditySharePercent    int64               `yaml:"liquidity_share_percent"`
	InitialCurrencySeedValue int64               `yaml:"initial_currency_seed_value"`
	Roles                    map[string]RoleSeed `yaml:"roles"`
}

type RoleSeed struct {
	Username string           `yaml:"username"`
	Balances map[string]int64 `yaml:"balances"`
}

func LoadEconomyConfig() (*EconomyConfig, error) {
	// 実行ディレクトリから configs/economy_config.yaml を探す
	configPath := filepath.Join("configs", "economy_config.yaml")

	// テスト時などのために、環境変数でもパスを指定できるようにしておく
	if envPath := os.Getenv("ECONOMY_CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// デフォルト値
			return defaultEconomyConfig(), nil
		}
		return nil, err
	}

	var conf EconomyConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

func defaultEconomyConfig() *EconomyConfig {
	groupA := map[string]int64{"ARC": 200_000, "BRB": 200_000, "DRL": 200_000}
	groupB := map[string]int64{"VND": 300_000, "VDP": 300_000}
	groupC := map[string]int64{"ZMR": 300_000, "RVD": 300_000}

	roles := map[string]RoleSeed{
		"market_maker":        {Username: "Market Maker", Balances: map[string]int64{"ARC": 5_000_000}},
		"liquidity_provider":  {Username: "Liquidity Provider", Balances: map[string]int64{}}, // dynamically handled
		"whale_northern":      {Username: "Whale Northern", Balances: map[string]int64{"ARC": 5_000_000, "BRB": 5_000_000, "DRL": 5_000_000}},
		"whale_oceanic":       {Username: "Whale Oceanic", Balances: map[string]int64{"VND": 7_500_000, "VDP": 7_500_000}},
		"whale_energy":        {Username: "Whale Energy", Balances: map[string]int64{"ZMR": 7_500_000, "RVD": 7_500_000}},
		"national_ai_arcadia": {Username: "National AI Arcadia", Balances: map[string]int64{"ARC": 30_000_000}},
	}

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
	for _, s := range nationalSeeds {
		roles[s.role] = RoleSeed{Username: s.username, Balances: map[string]int64{s.currency: 20_000_000, "ARC": 10_000_000}}
	}

	traders := []struct {
		role string
		set  map[string]int64
	}{
		{"momentum_chaser_a", groupA}, {"momentum_chaser_b", groupB}, {"momentum_chaser_c", groupC},
		{"dip_buyer_a", groupA}, {"dip_buyer_b", groupB}, {"dip_buyer_c", groupC},
		{"reversal_sniper_a", groupA}, {"reversal_sniper_b", groupB}, {"reversal_sniper_c", groupC},
		{"grid_trader_a", groupA}, {"grid_trader_b", groupB}, {"grid_trader_c", groupC},
		{"news_reactor", groupA}, {"arbitrageur", groupA}, {"yield_hunter", groupA}, {"public_consumer", groupA},
	}
	for _, entry := range traders {
		// username is done via strings.ReplaceAll in initial_allocation but here we can define it.
		username := "bot " + entry.role
		roles[entry.role] = RoleSeed{Username: username, Balances: entry.set}
	}

	return &EconomyConfig{
		MarketMakerSharePercent:  20,
		LiquiditySharePercent:    30,
		InitialCurrencySeedValue: 20_000_000,
		Roles:                    roles,
	}
}
