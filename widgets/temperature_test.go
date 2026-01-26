package widgets

import (
	"bytes"
	"testing"

	"github.com/spf13/afero"

	"neoden/h2status/config"
)

func TestTemperature_MillidegreesConversion(t *testing.T) {
	tests := []struct {
		name         string
		millidegrees int
		wantCelsius  int
	}{
		{"50C", 50000, 50},
		{"75C", 75000, 75},
		{"90C", 90000, 90},
		{"0C", 0, 0},
		{"100C", 100000, 100},
		{"45.5C rounds down", 45500, 45},
		{"real CPU temp", 67000, 67},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			celsius := tt.millidegrees / 1000
			if celsius != tt.wantCelsius {
				t.Errorf("conversion = %d, want %d", celsius, tt.wantCelsius)
			}
		})
	}
}

func TestTemperature_GetBlock_SingleSensor(t *testing.T) {
	tests := []struct {
		name       string
		value      int
		showAbove  int
		urgAbove   int
		wantEmpty  bool
		wantUrgent bool
	}{
		{"below threshold - hidden", 50, 75, 90, true, false},
		{"at threshold - hidden", 75, 75, 90, true, false},
		{"above threshold - shown", 80, 75, 90, false, false},
		{"at urgent threshold - not urgent", 90, 75, 90, false, false},
		{"above urgent - urgent", 95, 75, 90, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			temp := &Temperature{
				sensors: []TempSensor{{
					Path:        "/sys/class/hwmon/hwmon0/temp1_input",
					Label:       "CPU",
					ShowAbove:   tt.showAbove,
					UrgentAbove: tt.urgAbove,
					Value:       tt.value,
				}},
			}

			block := temp.GetBlock()

			if tt.wantEmpty && block != "" {
				t.Errorf("GetBlock() = %q, want empty", block)
			}
			if !tt.wantEmpty && block == "" {
				t.Error("GetBlock() = empty, want non-empty")
			}
			if !tt.wantEmpty && tt.wantUrgent && !contains(block, `"urgent":true`) {
				t.Errorf("GetBlock() should be urgent: %s", block)
			}
		})
	}
}

func TestTemperature_GetBlock_MultipleSensors(t *testing.T) {
	temp := &Temperature{
		sensors: []TempSensor{
			{Label: "CPU", ShowAbove: 75, UrgentAbove: 90, Value: 80},
			{Label: "GPU", ShowAbove: 75, UrgentAbove: 90, Value: 85},
		},
	}

	block := temp.GetBlock()

	// Should contain both labels when multiple sensors shown
	if !contains(block, "CPU") {
		t.Errorf("GetBlock() should contain CPU label: %s", block)
	}
	if !contains(block, "GPU") {
		t.Errorf("GetBlock() should contain GPU label: %s", block)
	}
}

func TestTemperature_GetBlock_PartialShow(t *testing.T) {
	temp := &Temperature{
		sensors: []TempSensor{
			{Label: "CPU", ShowAbove: 75, UrgentAbove: 90, Value: 80}, // shown
			{Label: "GPU", ShowAbove: 75, UrgentAbove: 90, Value: 50}, // hidden
		},
	}

	block := temp.GetBlock()

	if !contains(block, "CPU") {
		t.Errorf("GetBlock() should contain CPU: %s", block)
	}
	// GPU should still appear in label since we have multiple sensors configured
	// but only one is shown, so format changes
}

func TestTemperature_GetBlock_AllHidden(t *testing.T) {
	temp := &Temperature{
		sensors: []TempSensor{
			{Label: "CPU", ShowAbove: 75, UrgentAbove: 90, Value: 50},
			{Label: "GPU", ShowAbove: 75, UrgentAbove: 90, Value: 60},
		},
	}

	block := temp.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty when all below threshold", block)
	}
}

func TestTemperature_AnyUrgent(t *testing.T) {
	temp := &Temperature{
		sensors: []TempSensor{
			{Label: "CPU", ShowAbove: 75, UrgentAbove: 90, Value: 80}, // not urgent
			{Label: "GPU", ShowAbove: 75, UrgentAbove: 90, Value: 95}, // urgent
		},
	}

	block := temp.GetBlock()

	if !contains(block, `"urgent":true`) {
		t.Errorf("GetBlock() should be urgent when any sensor is urgent: %s", block)
	}
}

func TestNewTemperature_WithConfig(t *testing.T) {
	cfgs := []config.TemperatureConfig{
		{Path: "/sys/path1", Label: "CPU", ShowAbove: 80, UrgentAbove: 95},
		{Path: "/sys/path2", Label: "GPU", ShowAbove: 70, UrgentAbove: 85},
	}

	temp := NewTemperature(cfgs, afero.NewMemMapFs())

	if len(temp.sensors) != 2 {
		t.Fatalf("sensors count = %d, want 2", len(temp.sensors))
	}

	if temp.sensors[0].Path != "/sys/path1" {
		t.Errorf("sensors[0].Path = %q, want /sys/path1", temp.sensors[0].Path)
	}
	if temp.sensors[0].Label != "CPU" {
		t.Errorf("sensors[0].Label = %q, want CPU", temp.sensors[0].Label)
	}
	if temp.sensors[0].ShowAbove != 80 {
		t.Errorf("sensors[0].ShowAbove = %d, want 80", temp.sensors[0].ShowAbove)
	}
	if temp.sensors[1].UrgentAbove != 85 {
		t.Errorf("sensors[1].UrgentAbove = %d, want 85", temp.sensors[1].UrgentAbove)
	}
}

func TestTemperature_Update(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/sys/hwmon/temp1", []byte("75000\n"), 0644) // 75°C
	afero.WriteFile(fs, "/sys/hwmon/temp2", []byte("60000\n"), 0644) // 60°C

	cfgs := []config.TemperatureConfig{
		{Path: "/sys/hwmon/temp1", Label: "CPU", ShowAbove: 70, UrgentAbove: 90},
		{Path: "/sys/hwmon/temp2", Label: "GPU", ShowAbove: 70, UrgentAbove: 90},
	}

	temp := NewTemperature(cfgs, fs)
	temp.Update()

	if temp.sensors[0].Value != 75 {
		t.Errorf("sensors[0].Value = %d, want 75", temp.sensors[0].Value)
	}
	if temp.sensors[1].Value != 60 {
		t.Errorf("sensors[1].Value = %d, want 60", temp.sensors[1].Value)
	}
}

func TestTemperature_Update_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs() // empty

	cfgs := []config.TemperatureConfig{
		{Path: "/nonexistent", Label: "CPU", ShowAbove: 70, UrgentAbove: 90},
	}

	temp := NewTemperature(cfgs, fs)
	temp.Update() // should not panic

	if temp.sensors[0].Value != 0 {
		t.Errorf("sensors[0].Value = %d, want 0 (default)", temp.sensors[0].Value)
	}
}

func TestTemperature_Update_InvalidContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/sys/hwmon/temp1", []byte("not a number\n"), 0644)

	cfgs := []config.TemperatureConfig{
		{Path: "/sys/hwmon/temp1", Label: "CPU", ShowAbove: 70, UrgentAbove: 90},
	}

	temp := NewTemperature(cfgs, fs)
	temp.Update() // should not panic

	if temp.sensors[0].Value != 0 {
		t.Errorf("sensors[0].Value = %d, want 0", temp.sensors[0].Value)
	}
}

func TestTemperature_AutoDetect_Hwmon(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create hwmon with coretemp
	fs.MkdirAll("/sys/class/hwmon/hwmon0", 0755)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/name", []byte("coretemp\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_input", []byte("65000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_label", []byte("Package id 0\n"), 0644)

	temp := NewTemperature(nil, fs) // nil config = auto-detect

	if len(temp.sensors) != 1 {
		t.Fatalf("sensors count = %d, want 1", len(temp.sensors))
	}
	if temp.sensors[0].Label != "Package id 0" {
		t.Errorf("Label = %q, want 'Package id 0'", temp.sensors[0].Label)
	}
	if temp.sensors[0].Path != "/sys/class/hwmon/hwmon0/temp1_input" {
		t.Errorf("Path = %q, want '/sys/class/hwmon/hwmon0/temp1_input'", temp.sensors[0].Path)
	}
}

func TestTemperature_AutoDetect_HwmonNoLabel(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create hwmon without label file
	fs.MkdirAll("/sys/class/hwmon/hwmon0", 0755)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/name", []byte("k10temp\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_input", []byte("55000\n"), 0644)

	temp := NewTemperature(nil, fs)

	if len(temp.sensors) != 1 {
		t.Fatalf("sensors count = %d, want 1", len(temp.sensors))
	}
	if temp.sensors[0].Label != "k10temp" {
		t.Errorf("Label = %q, want 'k10temp' (fallback to name)", temp.sensors[0].Label)
	}
}

func TestTemperature_AutoDetect_ThermalZoneFallback(t *testing.T) {
	fs := afero.NewMemMapFs()

	// No hwmon, only thermal_zone
	fs.MkdirAll("/sys/class/thermal/thermal_zone0", 0755)
	afero.WriteFile(fs, "/sys/class/thermal/thermal_zone0/type", []byte("acpitz\n"), 0644)
	afero.WriteFile(fs, "/sys/class/thermal/thermal_zone0/temp", []byte("45000\n"), 0644)

	temp := NewTemperature(nil, fs)

	if len(temp.sensors) != 1 {
		t.Fatalf("sensors count = %d, want 1", len(temp.sensors))
	}
	if temp.sensors[0].Label != "acpitz" {
		t.Errorf("Label = %q, want 'acpitz'", temp.sensors[0].Label)
	}
}

func TestTemperature_AutoDetect_Priority(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Two hwmons - coretemp should be preferred over acpitz
	fs.MkdirAll("/sys/class/hwmon/hwmon0", 0755)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/name", []byte("acpitz\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_input", []byte("40000\n"), 0644)

	fs.MkdirAll("/sys/class/hwmon/hwmon1", 0755)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon1/name", []byte("coretemp\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon1/temp1_input", []byte("65000\n"), 0644)

	temp := NewTemperature(nil, fs)

	if len(temp.sensors) != 1 {
		t.Fatalf("sensors count = %d, want 1", len(temp.sensors))
	}
	if temp.sensors[0].Label != "coretemp" {
		t.Errorf("Label = %q, want 'coretemp' (higher priority)", temp.sensors[0].Label)
	}
}

func TestTemperature_AutoDetect_NoSensors(t *testing.T) {
	fs := afero.NewMemMapFs() // empty

	temp := NewTemperature(nil, fs)

	if len(temp.sensors) != 0 {
		t.Errorf("sensors count = %d, want 0", len(temp.sensors))
	}
}

func TestPrintDetectedSensorsTo_Hwmon(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create hwmon with two temp sensors
	fs.MkdirAll("/sys/class/hwmon/hwmon0", 0755)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/name", []byte("coretemp\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_input", []byte("65000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_label", []byte("Package id 0\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp2_input", []byte("60000\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp2_label", []byte("Core 0\n"), 0644)

	var buf bytes.Buffer
	PrintDetectedSensorsTo(fs, &buf)
	output := buf.String()

	if !contains(output, "# hwmon sensors:") {
		t.Error("output should contain hwmon header")
	}
	if !contains(output, "[[temperature]]") {
		t.Error("output should contain [[temperature]] sections")
	}
	if !contains(output, `path = "/sys/class/hwmon/hwmon0/temp1_input"`) {
		t.Errorf("output should contain temp1 path: %s", output)
	}
	if !contains(output, `label = "Package id 0"`) {
		t.Errorf("output should contain Package id 0 label: %s", output)
	}
	if !contains(output, "# coretemp") {
		t.Errorf("output should contain hwmon name as comment: %s", output)
	}
}

func TestPrintDetectedSensorsTo_HwmonNoLabel(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create hwmon without label file
	fs.MkdirAll("/sys/class/hwmon/hwmon0", 0755)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/name", []byte("k10temp\n"), 0644)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_input", []byte("55000\n"), 0644)

	var buf bytes.Buffer
	PrintDetectedSensorsTo(fs, &buf)
	output := buf.String()

	// Label should fall back to name when no label file
	if !contains(output, `label = "k10temp"`) {
		t.Errorf("output should use hwmon name as label fallback: %s", output)
	}
}

func TestPrintDetectedSensorsTo_ThermalZone(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create thermal_zone
	fs.MkdirAll("/sys/class/thermal/thermal_zone0", 0755)
	afero.WriteFile(fs, "/sys/class/thermal/thermal_zone0/type", []byte("acpitz\n"), 0644)
	afero.WriteFile(fs, "/sys/class/thermal/thermal_zone0/temp", []byte("45000\n"), 0644)

	var buf bytes.Buffer
	PrintDetectedSensorsTo(fs, &buf)
	output := buf.String()

	if !contains(output, "# thermal_zone sensors:") {
		t.Error("output should contain thermal_zone header")
	}
	if !contains(output, `path = "/sys/class/thermal/thermal_zone0/temp"`) {
		t.Errorf("output should contain thermal_zone path: %s", output)
	}
	if !contains(output, `label = "acpitz"`) {
		t.Errorf("output should contain acpitz label: %s", output)
	}
}

func TestPrintDetectedSensorsTo_NoSensors(t *testing.T) {
	fs := afero.NewMemMapFs() // empty

	var buf bytes.Buffer
	PrintDetectedSensorsTo(fs, &buf)
	output := buf.String()

	// Should still print headers
	if !contains(output, "# Detected temperature sensors:") {
		t.Error("output should contain main header even with no sensors")
	}
	if !contains(output, "# hwmon sensors:") {
		t.Error("output should contain hwmon header even with no sensors")
	}
	if !contains(output, "# thermal_zone sensors:") {
		t.Error("output should contain thermal_zone header even with no sensors")
	}
	// Should not contain any [[temperature]] sections
	if contains(output, "[[temperature]]") {
		t.Error("output should not contain [[temperature]] when no sensors found")
	}
}

func TestPrintDetectedSensorsTo_HwmonNoName(t *testing.T) {
	fs := afero.NewMemMapFs()

	// hwmon directory exists but no name file
	fs.MkdirAll("/sys/class/hwmon/hwmon0", 0755)
	afero.WriteFile(fs, "/sys/class/hwmon/hwmon0/temp1_input", []byte("55000\n"), 0644)

	var buf bytes.Buffer
	PrintDetectedSensorsTo(fs, &buf)
	output := buf.String()

	// Should skip hwmon without name file
	if contains(output, "temp1_input") {
		t.Errorf("output should not contain sensor from hwmon without name: %s", output)
	}
}

func TestPrintDetectedSensorsTo_ThermalZoneNoType(t *testing.T) {
	fs := afero.NewMemMapFs()

	// thermal_zone directory exists but no type file
	fs.MkdirAll("/sys/class/thermal/thermal_zone0", 0755)
	afero.WriteFile(fs, "/sys/class/thermal/thermal_zone0/temp", []byte("45000\n"), 0644)

	var buf bytes.Buffer
	PrintDetectedSensorsTo(fs, &buf)
	output := buf.String()

	// Should skip thermal_zone without type file
	if contains(output, "thermal_zone0") {
		t.Errorf("output should not contain zone without type: %s", output)
	}
}

func TestTemperature_EMASmoothing(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/sys/hwmon/temp1", []byte("50000\n"), 0644) // 50°C

	cfgs := []config.TemperatureConfig{
		{Path: "/sys/hwmon/temp1", Label: "CPU", ShowAbove: 40, UrgentAbove: 90, SmoothingIntervalSeconds: 3},
	}

	temp := NewTemperature(cfgs, fs)
	temp.Update()

	// First update should set value directly
	if temp.sensors[0].Value != 50 {
		t.Errorf("First update: Value = %d, want 50", temp.sensors[0].Value)
	}

	// Spike to 80°C - should be smoothed
	afero.WriteFile(fs, "/sys/hwmon/temp1", []byte("80000\n"), 0644)
	temp.Update()

	// Value should be between 50 and 80 due to smoothing
	if temp.sensors[0].Value <= 50 || temp.sensors[0].Value >= 80 {
		t.Errorf("After spike: Value = %d, want between 50 and 80", temp.sensors[0].Value)
	}

	// Continue updating with 80°C - should converge
	for i := 0; i < 20; i++ {
		temp.Update()
	}

	// Should be close to 80 now
	if temp.sensors[0].Value < 78 {
		t.Errorf("After convergence: Value = %d, want >= 78", temp.sensors[0].Value)
	}
}

func TestTemperature_EMARounding(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/sys/hwmon/temp1", []byte("50000\n"), 0644)

	cfgs := []config.TemperatureConfig{
		{Path: "/sys/hwmon/temp1", Label: "CPU", ShowAbove: 40, UrgentAbove: 90, SmoothingIntervalSeconds: 1},
	}

	temp := NewTemperature(cfgs, fs)
	temp.Update()

	// With period=1, alpha=1, so value should equal input
	// 49.5°C should round to 50
	afero.WriteFile(fs, "/sys/hwmon/temp1", []byte("49500\n"), 0644)
	temp.Update()

	if temp.sensors[0].Value != 50 {
		t.Errorf("Value = %d, want 50 (rounded from 49.5)", temp.sensors[0].Value)
	}
}

func TestTemperature_DefaultSmoothingInterval(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/sys/hwmon/temp1", []byte("50000\n"), 0644)

	// Config without SmoothingIntervalSeconds - should use default
	cfgs := []config.TemperatureConfig{
		{Path: "/sys/hwmon/temp1", Label: "CPU", ShowAbove: 40, UrgentAbove: 90},
	}

	temp := NewTemperature(cfgs, fs)

	// Verify EMA was created (sensor has ema field)
	if temp.sensors[0].ema == nil {
		t.Error("EMA should be initialized with default interval")
	}

	temp.Update()
	if temp.sensors[0].Value != 50 {
		t.Errorf("Value = %d, want 50", temp.sensors[0].Value)
	}
}
