package widgets

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/afero"

	"neoden/h2status/config"
)

func TestBattery_PercentageCalculation(t *testing.T) {
	tests := []struct {
		name       string
		energyFull int
		energyNow  int
		expected   int
	}{
		{"full battery", 100000, 100000, 100},
		{"half battery", 100000, 50000, 50},
		{"quarter battery", 100000, 25000, 25},
		{"empty battery", 100000, 0, 0},
		{"75% battery", 80000, 60000, 75},
		{"real world values", 47520000, 35640000, 75}, // ~75%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate like Battery.Update() does
			var percentage int
			if tt.energyFull > 0 {
				percentage = 100 * tt.energyNow / tt.energyFull
			}

			if percentage != tt.expected {
				t.Errorf("percentage = %d, want %d", percentage, tt.expected)
			}
		})
	}
}

func TestBattery_RemainingTimeCalculation(t *testing.T) {
	tests := []struct {
		name       string
		energyFull int
		energyNow  int
		powerNow   int
		isCharging bool
		expected   time.Duration
	}{
		{
			name:       "2 hours discharging",
			energyNow:  20000000, // 20Wh in uWh
			powerNow:   10000000, // 10W in uW
			isCharging: false,
			expected:   2 * time.Hour,
		},
		{
			name:       "30 minutes discharging",
			energyNow:  5000000,  // 5Wh
			powerNow:   10000000, // 10W
			isCharging: false,
			expected:   30 * time.Minute,
		},
		{
			name:       "1 hour charging",
			energyFull: 50000000,
			energyNow:  40000000,
			powerNow:   10000000,
			isCharging: true,
			expected:   1 * time.Hour,
		},
		{
			name:       "real world discharging",
			energyNow:  35640000, // ~35.6Wh
			powerNow:   8500000,  // ~8.5W
			isCharging: false,
			expected:   4*time.Hour + 11*time.Minute, // 35.6/8.5 ≈ 4.19h
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var remaining time.Duration
			if tt.powerNow > 0 {
				if tt.isCharging {
					remaining = time.Duration((tt.energyFull-tt.energyNow)*1000/tt.powerNow) * time.Hour / 1000
				} else {
					remaining = time.Duration(tt.energyNow*1000/tt.powerNow) * time.Hour / 1000
				}
			}

			// Allow 1 minute tolerance for rounding
			diff := remaining - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Minute {
				t.Errorf("remaining = %v, want %v (diff: %v)", remaining, tt.expected, diff)
			}
		})
	}
}

func TestBattery_Update(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create battery directory structure
	fs.MkdirAll("/sys/class/power_supply/BAT0", 0755)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/capacity", []byte("75\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/status", []byte("Discharging\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/energy_full", []byte("50000000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/energy_now", []byte("37500000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/power_now", []byte("10000000\n"), 0644)

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if !b.Present {
		t.Error("Present = false, want true")
	}
	if b.Percentage != 75 {
		t.Errorf("Percentage = %d, want 75", b.Percentage)
	}
	if b.State != BatteryStateDischarging {
		t.Errorf("State = %v, want BatteryStateDischarging", b.State)
	}
}

func TestBattery_Update_Charging(t *testing.T) {
	fs := afero.NewMemMapFs()

	fs.MkdirAll("/sys/class/power_supply/BAT0", 0755)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/capacity", []byte("50\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/status", []byte("Charging\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/energy_full", []byte("50000000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/energy_now", []byte("25000000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/power_now", []byte("25000000\n"), 0644)

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if b.State != BatteryStateCharging {
		t.Errorf("State = %v, want BatteryStateCharging", b.State)
	}
}

func TestBattery_Update_NoBattery(t *testing.T) {
	fs := afero.NewMemMapFs() // empty

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if b.Present {
		t.Error("Present = true, want false")
	}
}

func TestBattery_Update_ChargeInsteadOfEnergy(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Some systems use charge_* instead of energy_*
	fs.MkdirAll("/sys/class/power_supply/BAT0", 0755)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/capacity", []byte("80\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/status", []byte("Discharging\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/charge_full", []byte("4000000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/charge_now", []byte("3200000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/current_now", []byte("1000000\n"), 0644)

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if !b.Present {
		t.Error("Present = false, want true")
	}
	if b.EnergyFull != 4000000 {
		t.Errorf("EnergyFull = %d, want 4000000", b.EnergyFull)
	}
}

func TestBattery_HandleClick(t *testing.T) {
	b := NewBattery(config.BatteryConfig{}, afero.NewMemMapFs())

	if b.Mode != BatteryModePercentage {
		t.Errorf("initial Mode = %d, want %d", b.Mode, BatteryModePercentage)
	}

	b.HandleClick(1)
	if b.Mode != BatteryModeRemainingTime {
		t.Errorf("after first click Mode = %d, want %d", b.Mode, BatteryModeRemainingTime)
	}

	b.HandleClick(1)
	if b.Mode != BatteryModePercentage {
		t.Errorf("after second click Mode = %d, want %d", b.Mode, BatteryModePercentage)
	}
}

func TestBattery_HideLogic(t *testing.T) {
	tests := []struct {
		name                 string
		percentage           int
		state                BatteryState
		hideChargingAbove    int
		hideDischargingAbove int
		shouldHide           bool
	}{
		{
			name:              "charging above threshold - hide",
			percentage:        99,
			state:             BatteryStateCharging,
			hideChargingAbove: 98,
			shouldHide:        true,
		},
		{
			name:              "charging below threshold - show",
			percentage:        50,
			state:             BatteryStateCharging,
			hideChargingAbove: 98,
			shouldHide:        false,
		},
		{
			name:                 "discharging above threshold - hide",
			percentage:           50,
			state:                BatteryStateDischarging,
			hideDischargingAbove: 20,
			shouldHide:           true,
		},
		{
			name:                 "discharging below threshold - show",
			percentage:           15,
			state:                BatteryStateDischarging,
			hideDischargingAbove: 20,
			shouldHide:           false,
		},
		{
			name:              "charging at threshold - show",
			percentage:        98,
			state:             BatteryStateCharging,
			hideChargingAbove: 98,
			shouldHide:        false,
		},
		{
			name:                 "discharging at threshold - show",
			percentage:           20,
			state:                BatteryStateDischarging,
			hideDischargingAbove: 20,
			shouldHide:           false,
		},
		{
			name:              "full above threshold - hide",
			percentage:        100,
			state:             BatteryStateFull,
			hideChargingAbove: 98,
			shouldHide:        true,
		},
		{
			name:              "not charging above threshold - hide",
			percentage:        80,
			state:             BatteryStateNotCharging,
			hideChargingAbove: 70,
			shouldHide:        true,
		},
		{
			name:                 "unknown above threshold - hide",
			percentage:           50,
			state:                BatteryStateUnknown,
			hideDischargingAbove: 20,
			shouldHide:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Battery{
				cfg: config.BatteryConfig{
					HideChargingAbove:    tt.hideChargingAbove,
					HideDischargingAbove: tt.hideDischargingAbove,
				},
				Present:    true,
				Percentage: tt.percentage,
				State:      tt.state,
			}

			block := b.GetBlock()
			isHidden := block == ""

			if isHidden != tt.shouldHide {
				t.Errorf("GetBlock() hidden = %v, want %v", isHidden, tt.shouldHide)
			}
		})
	}
}

func TestBattery_NotPresent(t *testing.T) {
	b := &Battery{
		Present: false,
	}

	block := b.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty string when not present", block)
	}
}

func TestBattery_GetBlock_RemainingTimeMode(t *testing.T) {
	b := &Battery{
		cfg: config.BatteryConfig{
			HideDischargingAbove: 100,
		},
		Present:    true,
		Percentage: 50,
		State:      BatteryStateDischarging,
		Remaining:  2*time.Hour + 30*time.Minute,
		Mode:       BatteryModeRemainingTime,
	}

	block := b.GetBlock()
	if block == "" {
		t.Error("GetBlock() = empty, want non-empty")
	}
	if !contains(block, "2:30") {
		t.Errorf("GetBlock() should contain remaining time '2:30': %s", block)
	}
}

func TestBattery_Update_NoCapacityFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Battery directory exists but no capacity file
	fs.MkdirAll("/sys/class/power_supply/BAT0", 0755)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/status", []byte("Discharging\n"), 0644)
	// No capacity file - this battery should be skipped

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if b.Present {
		t.Error("Present = true, want false (no capacity file)")
	}
}

func TestBattery_Update_NoStatusFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Battery has capacity but no status file
	fs.MkdirAll("/sys/class/power_supply/BAT0", 0755)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/capacity", []byte("75\n"), 0644)
	// No status file - this battery should be skipped

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if b.Present {
		t.Error("Present = true, want false (no status file)")
	}
}

func TestBattery_Update_AllBatteriesInvalid(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Two battery directories, both invalid
	fs.MkdirAll("/sys/class/power_supply/BAT0", 0755)
	// BAT0 has no capacity file

	fs.MkdirAll("/sys/class/power_supply/BAT1", 0755)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT1/capacity", []byte("50\n"), 0644)
	// BAT1 has no status file

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if b.Present {
		t.Error("Present = true, want false (all batteries invalid)")
	}
}

func TestBattery_GetBlock_PercentageOver100(t *testing.T) {
	// ACPI drivers can report > 100% due to calibration issues
	// idx = percentage / 20, need >= 120 to trigger idx > 5 clamp
	b := &Battery{
		cfg: config.BatteryConfig{
			HideDischargingAbove: 200, // don't hide
		},
		Present:    true,
		Percentage: 125, // buggy ACPI
		State:      BatteryStateDischarging,
	}

	block := b.GetBlock()
	if block == "" {
		t.Error("GetBlock() = empty, want non-empty")
	}
	// Should show real percentage, icon clamped to full battery
	if !contains(block, "125%") {
		t.Errorf("GetBlock() should show real percentage '125%%': %s", block)
	}
}

func TestBattery_UrgentBelow(t *testing.T) {
	tests := []struct {
		name        string
		percentage  int
		urgentBelow int
		wantUrgent  bool
	}{
		{"below threshold", 5, 10, true},
		{"at threshold", 10, 10, false},
		{"above threshold", 15, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Battery{
				cfg: config.BatteryConfig{
					UrgentBelow:          tt.urgentBelow,
					HideDischargingAbove: 100,
				},
				Present:    true,
				Percentage: tt.percentage,
				State:      BatteryStateDischarging,
			}

			block := b.GetBlock()

			// Check if block contains "urgent":true
			hasUrgent := len(block) > 0 && contains(block, `"urgent":true`)
			if hasUrgent != tt.wantUrgent {
				t.Errorf("urgent = %v, want %v (block: %s)", hasUrgent, tt.wantUrgent, block)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseBatteryStatus(t *testing.T) {
	tests := []struct {
		status string
		want   BatteryState
	}{
		{"Charging", BatteryStateCharging},
		{"Discharging", BatteryStateDischarging},
		{"Full", BatteryStateFull},
		{"Not charging", BatteryStateNotCharging},
		{"Unknown", BatteryStateUnknown},
		{"", BatteryStateUnknown},
		{"Something else", BatteryStateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := parseBatteryStatus(tt.status)
			if got != tt.want {
				t.Errorf("parseBatteryStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestBatteryState_Priority(t *testing.T) {
	// Charging > Discharging > NotCharging > Full > Unknown
	if BatteryStateCharging.priority() <= BatteryStateDischarging.priority() {
		t.Error("Charging should have higher priority than Discharging")
	}
	if BatteryStateDischarging.priority() <= BatteryStateNotCharging.priority() {
		t.Error("Discharging should have higher priority than NotCharging")
	}
	if BatteryStateNotCharging.priority() <= BatteryStateFull.priority() {
		t.Error("NotCharging should have higher priority than Full")
	}
	if BatteryStateFull.priority() <= BatteryStateUnknown.priority() {
		t.Error("Full should have higher priority than Unknown")
	}
}

func TestBattery_Update_MultipleBatteries_StateAggregation(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		want     BatteryState
	}{
		{"both charging", []string{"Charging", "Charging"}, BatteryStateCharging},
		{"both discharging", []string{"Discharging", "Discharging"}, BatteryStateDischarging},
		{"one charging one discharging", []string{"Discharging", "Charging"}, BatteryStateCharging},
		{"one full one discharging", []string{"Full", "Discharging"}, BatteryStateDischarging},
		{"one full one not charging", []string{"Full", "Not charging"}, BatteryStateNotCharging},
		{"both full", []string{"Full", "Full"}, BatteryStateFull},
		{"unknown and full", []string{"Unknown", "Full"}, BatteryStateFull},
		{"unknown and discharging", []string{"Unknown", "Discharging"}, BatteryStateDischarging},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			for i, status := range tt.statuses {
				dir := fmt.Sprintf("/sys/class/power_supply/BAT%d", i)
				fs.MkdirAll(dir, 0755)
				afero.WriteFile(fs, dir+"/capacity", []byte("50\n"), 0644)
				afero.WriteFile(fs, dir+"/status", []byte(status+"\n"), 0644)
				afero.WriteFile(fs, dir+"/energy_full", []byte("50000000\n"), 0644)
				afero.WriteFile(fs, dir+"/energy_now", []byte("25000000\n"), 0644)
				afero.WriteFile(fs, dir+"/power_now", []byte("10000000\n"), 0644)
			}

			b := NewBattery(config.BatteryConfig{}, fs)
			b.Update()

			if b.State != tt.want {
				t.Errorf("State = %v, want %v", b.State, tt.want)
			}
		})
	}
}

func TestBattery_RemainingReset(t *testing.T) {
	fs := afero.NewMemMapFs()

	fs.MkdirAll("/sys/class/power_supply/BAT0", 0755)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/capacity", []byte("50\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/status", []byte("Discharging\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/energy_full", []byte("50000000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/energy_now", []byte("25000000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/power_now", []byte("10000000\n"), 0644)

	b := NewBattery(config.BatteryConfig{}, fs)
	b.Update()

	if b.Remaining == 0 {
		t.Fatal("Remaining should be non-zero when discharging with power_now > 0")
	}

	// Change to Full status (no remaining time)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/status", []byte("Full\n"), 0644)
	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/power_now", []byte("0\n"), 0644)
	b.Update()

	if b.Remaining != 0 {
		t.Errorf("Remaining = %v, want 0 after switching to Full", b.Remaining)
	}
}
