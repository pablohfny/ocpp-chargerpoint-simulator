package utils

import "math"

func RoundFloat(val float64, precision uint) float64 {
	if precision <= 1 {
		return val
	}
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
