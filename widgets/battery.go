package widgets

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"
)

const (
	BatteryModePercentage    = 0
	BatteryModeRemainingTime = 1
)

type BatteryState int

const (
	BatteryStateUnknown BatteryState = iota
	BatteryStateCharging
	BatteryStateDischarging
	BatteryStateFull
	BatteryStateNotCharging
)

func (s BatteryState) IsOnAC() bool {
	return s == BatteryStateCharging || s == BatteryStateFull || s == BatteryStateNotCharging
}

// priority returns the precedence for aggregating multiple battery states.
// Higher value = higher priority when combining states.
func (s BatteryState) priority() int {
	switch s {
	case BatteryStateCharging:
		return 5
	case BatteryStateDischarging:
		return 4
	case BatteryStateNotCharging:
		return 3
	case BatteryStateFull:
		return 2
	default: // BatteryStateUnknown
		return 1
	}
}

func parseBatteryStatus(status string) BatteryState {
	switch status {
	case "Charging":
		return BatteryStateCharging
	case "Discharging":
		return BatteryStateDischarging
	case "Full":
		return BatteryStateFull
	case "Not charging":
		return BatteryStateNotCharging
	default:
		return BatteryStateUnknown
	}
}

type BatteryInfo struct {
	Path       string
	Status     string
	EnergyFull int
	EnergyNow  int
	PowerNow   int
}

type Battery struct {
	fs         afero.Fs
	cfg        config.BatteryConfig
	batteries  []BatteryInfo
	Present    bool
	Percentage int
	EnergyFull int
	EnergyNow  int
	PowerNow   int
	Remaining  time.Duration
	State      BatteryState
	Mode       int
}

func NewBattery(cfg config.BatteryConfig, fs afero.Fs) *Battery {
	return &Battery{cfg: cfg, fs: fs}
}

func (b *Battery) readInt(path string) int {
	content, err := afero.ReadFile(b.fs, path)
	if err != nil {
		return 0
	}
	val, _ := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 32)
	return int(val)
}

func (b *Battery) Update() {
	b.batteries = nil
	b.Present = false
	b.EnergyFull = 0
	b.EnergyNow = 0
	b.PowerNow = 0
	b.Remaining = 0
	b.State = BatteryStateUnknown

	// Find all batteries
	matches, err := afero.Glob(b.fs, "/sys/class/power_supply/BAT*")
	if err != nil || len(matches) == 0 {
		return
	}

	for _, path := range matches {
		bat := BatteryInfo{Path: path}

		// Check if battery is present
		if _, err := b.fs.Stat(filepath.Join(path, "capacity")); err != nil {
			continue
		}

		// Read status
		status, err := afero.ReadFile(b.fs, filepath.Join(path, "status"))
		if err != nil {
			continue
		}
		bat.Status = strings.TrimSpace(string(status))

		// Read energy values
		bat.EnergyFull = b.readInt(filepath.Join(path, "energy_full"))
		bat.EnergyNow = b.readInt(filepath.Join(path, "energy_now"))
		bat.PowerNow = b.readInt(filepath.Join(path, "power_now"))

		// If energy_* not available, try charge_* (some systems use this)
		if bat.EnergyFull == 0 {
			bat.EnergyFull = b.readInt(filepath.Join(path, "charge_full"))
			bat.EnergyNow = b.readInt(filepath.Join(path, "charge_now"))
			bat.PowerNow = b.readInt(filepath.Join(path, "current_now"))
		}

		b.batteries = append(b.batteries, bat)

		// Accumulate totals
		b.EnergyFull += bat.EnergyFull
		b.EnergyNow += bat.EnergyNow
		b.PowerNow += bat.PowerNow

		// Determine state using priority (Charging > Discharging > NotCharging > Full > Unknown)
		batState := parseBatteryStatus(bat.Status)
		if batState.priority() > b.State.priority() {
			b.State = batState
		}
	}

	if len(b.batteries) == 0 {
		return
	}

	b.Present = true

	// Calculate combined percentage
	if b.EnergyFull > 0 {
		b.Percentage = 100 * b.EnergyNow / b.EnergyFull
	}

	// Calculate remaining time
	if b.PowerNow > 0 {
		if b.State == BatteryStateCharging {
			b.Remaining = time.Duration((b.EnergyFull-b.EnergyNow)*1000/b.PowerNow) * time.Hour / 1000
		} else if b.State == BatteryStateDischarging {
			b.Remaining = time.Duration(b.EnergyNow*1000/b.PowerNow) * time.Hour / 1000
		}
	}
}

func (b *Battery) GetBlock() string {
	if !b.Present {
		return ""
	}

	if b.State.IsOnAC() && b.Percentage > b.cfg.HideChargingAbove {
		return ""
	}
	if !b.State.IsOnAC() && b.Percentage > b.cfg.HideDischargingAbove {
		return ""
	}

	batteryLevelSymbols := [6]string{"\uf244", "\uf243", "\uf242", "\uf241", "\uf240", "\uf240"}
	var symbol string
	var text string

	if b.State == BatteryStateCharging {
		symbol = "\uf1e6" // nf-fa-plug
	} else {
		idx := b.Percentage / 20
		if idx > 5 {
			idx = 5
		}
		symbol = batteryLevelSymbols[idx]
	}

	if b.Mode == BatteryModePercentage {
		text = fmt.Sprintf("%s %d%%", symbol, b.Percentage)
	} else if b.Mode == BatteryModeRemainingTime {
		text = fmt.Sprintf("%s %s", symbol, FormatDuration(b.Remaining))
	}

	return swaybar.MakeBlock("power_supply", text, b.Percentage < b.cfg.UrgentBelow)
}

func (b *Battery) ClickName() string {
	return "power_supply"
}

func (b *Battery) HandleClick(button int) {
	b.Mode = (b.Mode + 1) % 2
}
