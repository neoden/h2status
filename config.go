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

type Config struct {
	Clock     ClockConfig     `toml:"clock"`
	Battery   BatteryConfig   `toml:"battery"`
	Bluetooth BluetoothConfig `toml:"bluetooth"`
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
