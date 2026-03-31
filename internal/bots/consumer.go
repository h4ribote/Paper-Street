package bots

import "math"

func ConsumerLuxuryShare(cci int64) float64 {
	if cci <= 0 {
		return 0.1
	}
	score := (float64(cci) - 80) / 60
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return 0.2 + score*0.6
}

func ClampQuantity(base int64, multiplier float64) int64 {
	if base <= 0 {
		return 0
	}
	qty := int64(math.Round(float64(base) * multiplier))
	if qty < 1 {
		return 1
	}
	return qty
}
