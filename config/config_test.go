package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	// Clock defaults
	if len(cfg.Clock.Formats) != 2 {
		t.Errorf("Clock.Formats length = %d, want 2", len(cfg.Clock.Formats))
	}
	if cfg.Clock.Formats[0] != "%H:%M" {
		t.Errorf("Clock.Formats[0] = %q, want %%H:%%M", cfg.Clock.Formats[0])
	}

	// Battery defaults
	if !cfg.Battery.Enabled {
		t.Error("Battery.Enabled = false, want true")
	}
	if cfg.Battery.HideChargingAbove != 98 {
		t.Errorf("Battery.HideChargingAbove = %d, want 98", cfg.Battery.HideChargingAbove)
	}
	if cfg.Battery.HideDischargingAbove != 20 {
		t.Errorf("Battery.HideDischargingAbove = %d, want 20", cfg.Battery.HideDischargingAbove)
	}
	if cfg.Battery.UrgentBelow != 10 {
		t.Errorf("Battery.UrgentBelow = %d, want 10", cfg.Battery.UrgentBelow)
	}

	// Bluetooth defaults
	if !cfg.Bluetooth.Enabled {
		t.Error("Bluetooth.Enabled = false, want true")
	}

	// CPU defaults
	if !cfg.CPU.Enabled {
		t.Error("CPU.Enabled = false, want true")
	}
	if cfg.CPU.ShowAbove != 50 {
		t.Errorf("CPU.ShowAbove = %d, want 50", cfg.CPU.ShowAbove)
	}
	if cfg.CPU.ShowCoreAbove != 95 {
		t.Errorf("CPU.ShowCoreAbove = %d, want 95", cfg.CPU.ShowCoreAbove)
	}
	if cfg.CPU.SmoothingIntervalSeconds != 3 {
		t.Errorf("CPU.SmoothingIntervalSeconds = %d, want 3", cfg.CPU.SmoothingIntervalSeconds)
	}
	if cfg.CPU.UrgentAbove != 95 {
		t.Errorf("CPU.UrgentAbove = %d, want 95", cfg.CPU.UrgentAbove)
	}

	// RAM defaults
	if !cfg.RAM.Enabled {
		t.Error("RAM.Enabled = false, want true")
	}
	if cfg.RAM.ShowAbove != 70 {
		t.Errorf("RAM.ShowAbove = %d, want 70", cfg.RAM.ShowAbove)
	}
	if cfg.RAM.UrgentAbove != 90 {
		t.Errorf("RAM.UrgentAbove = %d, want 90", cfg.RAM.UrgentAbove)
	}

	// Disk defaults
	if len(cfg.Disk) != 1 {
		t.Fatalf("Disk length = %d, want 1", len(cfg.Disk))
	}
	if cfg.Disk[0].Path != "/" {
		t.Errorf("Disk[0].Path = %q, want /", cfg.Disk[0].Path)
	}
	if cfg.Disk[0].ShowBelow != 20 {
		t.Errorf("Disk[0].ShowBelow = %d, want 20", cfg.Disk[0].ShowBelow)
	}
	if cfg.Disk[0].UrgentBelow != 5 {
		t.Errorf("Disk[0].UrgentBelow = %d, want 5", cfg.Disk[0].UrgentBelow)
	}
	if cfg.Disk[0].Unit != "gb" {
		t.Errorf("Disk[0].Unit = %q, want gb", cfg.Disk[0].Unit)
	}

	// WiFi defaults
	if !cfg.WiFi.Enabled {
		t.Error("WiFi.Enabled = false, want true")
	}
	if cfg.WiFi.ShowMode != "weak_signal" {
		t.Errorf("WiFi.ShowMode = %q, want weak_signal", cfg.WiFi.ShowMode)
	}
	if cfg.WiFi.ShowBelow != -70 {
		t.Errorf("WiFi.ShowBelow = %d, want -70", cfg.WiFi.ShowBelow)
	}
	if cfg.WiFi.UrgentBelow != -80 {
		t.Errorf("WiFi.UrgentBelow = %d, want -80", cfg.WiFi.UrgentBelow)
	}

	// Network defaults
	if !cfg.Network.Enabled {
		t.Error("Network.Enabled = false, want true")
	}

	// VPN defaults
	if !cfg.VPN.Enabled {
		t.Error("VPN.Enabled = false, want true")
	}
	if len(cfg.VPN.Interfaces) != 4 {
		t.Errorf("VPN.Interfaces length = %d, want 4", len(cfg.VPN.Interfaces))
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Save original HOME and restore after test
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CONFIG_HOME", origXDG)
	}()

	// Use temp dir as config home
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() error = %v, want nil", err)
	}

	// Should return defaults
	if cfg.CPU.ShowAbove != 50 {
		t.Errorf("CPU.ShowAbove = %d, want 50 (default)", cfg.CPU.ShowAbove)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Save original and restore after test
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Create temp config directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "h2status")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write test config
	configContent := `
[cpu]
enabled = false
show_above = 75
smoothing_interval_seconds = 10

[battery]
urgent_below = 15

[[disk]]
path = "/home"
show_below = 10
`
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check overridden values
	if cfg.CPU.Enabled {
		t.Error("CPU.Enabled = true, want false")
	}
	if cfg.CPU.ShowAbove != 75 {
		t.Errorf("CPU.ShowAbove = %d, want 75", cfg.CPU.ShowAbove)
	}
	if cfg.CPU.SmoothingIntervalSeconds != 10 {
		t.Errorf("CPU.SmoothingIntervalSeconds = %d, want 10", cfg.CPU.SmoothingIntervalSeconds)
	}
	if cfg.Battery.UrgentBelow != 15 {
		t.Errorf("Battery.UrgentBelow = %d, want 15", cfg.Battery.UrgentBelow)
	}

	// Check that non-overridden values keep defaults
	if cfg.RAM.ShowAbove != 70 {
		t.Errorf("RAM.ShowAbove = %d, want 70 (default)", cfg.RAM.ShowAbove)
	}
}

func TestLoad_InvalidToml(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "h2status")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write invalid TOML
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("invalid [[ toml"), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid TOML, got nil")
	}
}

func TestLoad_UnknownKeys(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "h2status")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Config with unknown keys
	configContent := `
[cpu]
enabled = true
unknown_key = "value"
average_seconds = 5
`
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Capture stderr
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	_, err := Load()

	w.Close()
	os.Stderr = origStderr

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	var buf [1024]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if !strings.Contains(output, "cpu.unknown_key") {
		t.Errorf("expected warning about cpu.unknown_key, got: %s", output)
	}
	if !strings.Contains(output, "cpu.average_seconds") {
		t.Errorf("expected warning about cpu.average_seconds (renamed), got: %s", output)
	}
}
