package entities

import "testing"

// The battery belongs to the EV: unplugging means the car left, so its state
// of charge and session meter must not linger on the connector.
func TestUnplugCableResetsBatteryState(t *testing.T) {
	tests := []struct {
		name    string
		soc     int16
		meterWh float64
	}{
		{name: "after a completed charge", soc: 100, meterWh: 22500},
		{name: "after a partial charge", soc: 37, meterWh: 4200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			point := NewChargerPoint(1)
			point.CablePlugged = true
			point.Soc = tt.soc
			point.MeterValue = tt.meterWh

			if err := point.UnplugCable(); err != nil {
				t.Fatalf("UnplugCable() error = %v", err)
			}
			if point.Soc != 0 {
				t.Errorf("Soc after unplug = %d, want 0", point.Soc)
			}
			if point.MeterValue != 0 {
				t.Errorf("MeterValue after unplug = %v, want 0", point.MeterValue)
			}
		})
	}
}

func TestUnplugCableWithoutCableFails(t *testing.T) {
	point := NewChargerPoint(1)
	point.Soc = 55

	if err := point.UnplugCable(); err == nil {
		t.Fatal("UnplugCable() with no cable should fail")
	}
	if point.Soc != 55 {
		t.Errorf("Soc must be untouched on failed unplug, got %d", point.Soc)
	}
}
