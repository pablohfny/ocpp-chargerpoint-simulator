package entities

import "math"

// DefaultBatteryCapacityKWh is the virtual EV battery size assumed when none is
// configured.
const DefaultBatteryCapacityKWh = 60.0

// DeriveBatteryPercent returns the battery charge level to display.
//
// The simulator already models a state of charge: the meter increment loop
// advances ChargerPoint.Soc on every tick, so that value is authoritative
// whenever a transaction is running. Outside a transaction Soc is 0 while the
// meter may still hold energy, so the level falls back to the energy delivered
// measured against the configured battery capacity.
func DeriveBatteryPercent(soc int16, meterWh, capacityKWh float64) int {
	if soc > 0 {
		return clampPercent(float64(soc))
	}

	if capacityKWh <= 0 {
		capacityKWh = DefaultBatteryCapacityKWh
	}
	if meterWh <= 0 {
		return 0
	}

	return clampPercent(meterWh / (capacityKWh * 1000) * 100)
}

// BatteryPercent returns the connector's battery level for the given capacity.
func (point *ChargerPoint) BatteryPercent(capacityKWh float64) int {
	return DeriveBatteryPercent(point.Soc, point.MeterValue, capacityKWh)
}

// clampPercent bounds a percentage to the 0..100 range.
func clampPercent(value float64) int {
	return int(math.Round(math.Max(0, math.Min(100, value))))
}
