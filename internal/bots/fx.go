package bots

import "math"

func PoolRate(pool LiquidityPool) float64 {
	if pool.CurrentTick == 0 {
		return 0
	}
	return math.Pow(1.0001, float64(pool.CurrentTick))
}

func FXDeviation(theoretical int64, market float64) float64 {
	if theoretical <= 0 || market <= 0 {
		return 0
	}
	return (market - float64(theoretical)) / float64(theoretical)
}
