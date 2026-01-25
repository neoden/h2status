package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type BatteryConfig struct {
	Enabled              bool `toml:"enabled"`
	HideChargingAbove    int  `toml:"hide_charging_above"`
	HideDischargingAbove int  `toml:"hide_discharging_above"`
	UrgentBelow          int  `toml:"urgent_below"`
}

type BluetoothConfig struct {
	Enabled bool `toml:"enabled"`
}

type ClockConfig struct {
	Format string `toml:"format"`
}

type CPUConfig struct {
	Enabled        bool `toml:"enabled"`
	ShowAbove      int  `toml:"show_above"`
	ShowCoreAbove  int  `toml:"show_core_above"`
	AverageSeconds int  `toml:"average_seconds"`
	UrgentAbove    int  `toml:"urgent_above"`
}

type RAMConfig struct {
	Enabled     bool `toml:"enabled"`
	ShowAbove   int  `toml:"show_above"`
	UrgentAbove int  `toml:"urgent_above"`
}

type DiskConfig struct {
	Path        string `toml:"path"`
	ShowBelow   int    `toml:"show_below"`
	UrgentBelow int    `toml:"urgent_below"`
	Unit        string `toml:"unit"` // "gb" or "percent"
}

type WiFiConfig struct {
	Enabled      bool     `toml:"enabled"`
	ShowMode     string   `toml:"show_mode"`     // "weak_signal", "unknown", "always"
	ShowBelow    int      `toml:"show_below"`    // signal strength in dBm (e.g. -70)
	UrgentBelow  int      `toml:"urgent_below"`
	HomeNetworks []string `toml:"home_networks"` // used when show_mode = "unknown"
}

type NetworkConfig struct {
	Enabled bool `toml:"enabled"`
}

type VPNConfig struct {
	Enabled    bool     `toml:"enabled"`
	Interfaces []string `toml:"interfaces"`
}

type TemperatureConfig struct {
	Path        string `toml:"path"`
	Label       string `toml:"label"`
	ShowAbove   int    `toml:"show_above"`
	UrgentAbove int    `toml:"urgent_above"`
}

type Config struct {
	Clock     ClockConfig     `toml:"clock"`
	Battery   BatteryConfig   `toml:"battery"`
	Bluetooth BluetoothConfig `toml:"bluetooth"`
	CPU       CPUConfig       `toml:"cpu"`
	RAM       RAMConfig       `toml:"ram"`
	Disk      []DiskConfig    `toml:"disk"`
	WiFi      WiFiConfig      `toml:"wifi"`
	Network   NetworkConfig   `toml:"network"`
	VPN         VPNConfig           `toml:"vpn"`
	Temperature []TemperatureConfig `toml:"temperature"`
}

func DefaultConfig() *Config {
	return &Config{
		Clock: ClockConfig{
			Format: "15:04",
		},
		Battery: BatteryConfig{
			Enabled:              true,
			HideChargingAbove:    98,
			HideDischargingAbove: 20,
			UrgentBelow:          10,
		},
		Bluetooth: BluetoothConfig{
			Enabled: true,
		},
		CPU: CPUConfig{
			Enabled:        true,
			ShowAbove:      50,
			ShowCoreAbove:  95,
			AverageSeconds: 5,
			UrgentAbove:    95,
		},
		RAM: RAMConfig{
			Enabled:     true,
			ShowAbove:   70,
			UrgentAbove: 90,
		},
		Disk: []DiskConfig{
			{
				Path:        "/",
				ShowBelow:   20,
				UrgentBelow: 5,
				Unit:        "gb",
			},
		},
		WiFi: WiFiConfig{
			Enabled:      true,
			ShowMode:     "weak_signal",
			ShowBelow:    -70,
			UrgentBelow:  -80,
			HomeNetworks: []string{},
		},
		Network: NetworkConfig{
			Enabled: true,
		},
		VPN: VPNConfig{
			Enabled:    true,
			Interfaces: []string{"tun*", "wg*", "tap*", "cscotun*"},
		},
	}
}

func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	configDir, err := os.UserConfigDir()
	if err != nil {
		return cfg, err
	}

	configPath := filepath.Join(configDir, "h2status", "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return cfg, nil
	}

	_, err = toml.DecodeFile(configPath, cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
