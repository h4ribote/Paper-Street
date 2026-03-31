package bots

func BondYield(baseCoupon int64, price int64) float64 {
	if baseCoupon <= 0 || price <= 0 {
		return 0
	}
	return float64(baseCoupon) / float64(price)
}

func EquityYield(dividendPerUnit int64, price int64) float64 {
	if dividendPerUnit <= 0 || price <= 0 {
		return 0
	}
	return float64(dividendPerUnit) / float64(price)
}

func YieldPreference(bondYield, equityYield, premium float64) string {
	if bondYield > equityYield+premium {
		return "BOND"
	}
	if equityYield > bondYield+premium {
		return "EQUITY"
	}
	return ""
}
