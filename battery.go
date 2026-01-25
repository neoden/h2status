package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

const BATTERY_STATE_MODE_PECENTAGE = 0
const BATTERY_STATE_MODE_REMAINING_TIME = 1

type BatteryState struct {
	Present    bool
	Percentage int
	Status     string
	EnergyFull int
	EnergyNow  int
	PowerNow   int
	Remaining  time.Duration
	IsCharging bool
	Mode       int
}

func NewBatteryState() *BatteryState {
	return &BatteryState{}
}

func (b *BatteryState) Update() {
	path := "/sys/class/power_supply/BAT0/"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		b.Present = false
		return
	}
	b.Present = true

	percentage, err := readInt(path + "capacity")
	if err != nil {
		l.Println(err)
	}
	b.Percentage = percentage

	status, err := ioutil.ReadFile(path + "status")
	if err != nil {
		l.Println(err)
	}
	b.Status = strings.Trim(string(status), "\n")
	b.IsCharging = b.Status == "Charging"

	power_now, err := readInt(path + "power_now")
	if err != nil || power_now == 0 {
		l.Println(err)
		return
	}
	b.PowerNow = power_now

	energy_now, err := readInt(path + "energy_now")
	if err != nil {
		l.Println(err)
		return
	}
	b.EnergyNow = energy_now

	energy_full, err := readInt(path + "energy_full")
	if err != nil {
		l.Println(err)
		return
	}
	b.EnergyFull = energy_full

	if b.IsCharging {
		b.Remaining = time.Duration(((energy_full - energy_now) * 1000 / power_now)) * time.Hour / 1000
	} else {
		b.Remaining = time.Duration(b.EnergyNow*1000/b.PowerNow) * time.Hour / 1000
	}
}

func (b *BatteryState) GetBlock() string {
	if !b.Present {
		return ""
	}

	if b.IsCharging && b.Percentage > cfg.Battery.HideChargingAbove {
		return ""
	}
	if !b.IsCharging && b.Percentage > cfg.Battery.HideDischargingAbove {
		return ""
	}

	var symbols [6]string = [6]string{"\uf244", "\uf243", "\uf242", "\uf241", "\uf240", "\uf240"}
	var symbol = symbols[0]
	var text = ""

	if b.IsCharging {
		symbol = "\uf1e6"
	} else {
		symbol = symbols[b.Percentage/20]
	}

	if b.Mode == BATTERY_STATE_MODE_PECENTAGE {
		text = fmt.Sprintf("%s %d%%", symbol, b.Percentage)
	} else if b.Mode == BATTERY_STATE_MODE_REMAINING_TIME {
		text = fmt.Sprintf("%s %s", symbol, fmtDuration(b.Remaining))
	}

	return MakeBlock("power_supply", text, b.Percentage < cfg.Battery.UrgentBelow)
}

func readInt(file string) (int, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return 0, err
	}
	value, _ := strconv.ParseInt(strings.Trim(string(content), "\n"), 10, 32)
	return int(value), nil
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%d:%02d", h, m)
}
