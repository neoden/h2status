package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type TempSensor struct {
	Path        string
	Label       string
	ShowAbove   int
	UrgentAbove int
	Value       int // current temperature in °C
}

type TemperatureState struct {
	sensors []TempSensor
}

func NewTemperatureState() *TemperatureState {
	t := &TemperatureState{}

	if len(cfg.Temperature) > 0 {
		// Use configured sensors
		for _, tc := range cfg.Temperature {
			t.sensors = append(t.sensors, TempSensor{
				Path:        tc.Path,
				Label:       tc.Label,
				ShowAbove:   tc.ShowAbove,
				UrgentAbove: tc.UrgentAbove,
			})
		}
	} else {
		// Auto-detect
		t.sensors = autoDetectTempSensors()
	}

	return t
}

func autoDetectTempSensors() []TempSensor {
	var sensors []TempSensor

	// Priority order for hwmon/thermal names
	priorities := []string{"coretemp", "k10temp", "x86_pkg", "cpu", "pkg", "acpitz"}

	// Try hwmon first (more accurate)
	hwmons, _ := filepath.Glob("/sys/class/hwmon/hwmon*")
	for _, prio := range priorities {
		for _, hwmon := range hwmons {
			nameBytes, err := os.ReadFile(filepath.Join(hwmon, "name"))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(string(nameBytes))

			if strings.Contains(strings.ToLower(name), prio) {
				// Find temp input file (usually temp1_input)
				tempFile := filepath.Join(hwmon, "temp1_input")
				if _, err := os.Stat(tempFile); err != nil {
					continue
				}

				// Try to get label
				label := name
				labelBytes, err := os.ReadFile(filepath.Join(hwmon, "temp1_label"))
				if err == nil {
					label = strings.TrimSpace(string(labelBytes))
				}

				sensors = append(sensors, TempSensor{
					Path:        tempFile,
					Label:       label,
					ShowAbove:   75,
					UrgentAbove: 90,
				})
				return sensors
			}
		}
	}

	// Fallback to thermal_zone
	zones, _ := filepath.Glob("/sys/class/thermal/thermal_zone*")
	for _, prio := range priorities {
		for _, zone := range zones {
			typeBytes, err := os.ReadFile(filepath.Join(zone, "type"))
			if err != nil {
				continue
			}
			zoneType := strings.TrimSpace(string(typeBytes))

			if strings.Contains(strings.ToLower(zoneType), prio) {
				sensors = append(sensors, TempSensor{
					Path:        filepath.Join(zone, "temp"),
					Label:       zoneType,
					ShowAbove:   75,
					UrgentAbove: 90,
				})
				return sensors
			}
		}
	}

	return sensors
}

func (t *TemperatureState) Update() {
	for i := range t.sensors {
		data, err := os.ReadFile(t.sensors[i].Path)
		if err != nil {
			l.Println("temperature:", err)
			continue
		}

		val, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			continue
		}

		// Convert from millidegrees to degrees
		t.sensors[i].Value = val / 1000
	}
}

func (t *TemperatureState) GetBlock() string {
	var parts []string
	var anyUrgent bool
	showMultiple := len(t.sensors) > 1

	for _, sensor := range t.sensors {
		if sensor.Value <= sensor.ShowAbove {
			continue
		}

		if sensor.Value > sensor.UrgentAbove {
			anyUrgent = true
		}

		if showMultiple {
			parts = append(parts, fmt.Sprintf("%s %d°", sensor.Label, sensor.Value))
		} else {
			parts = append(parts, fmt.Sprintf("%d°", sensor.Value))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	text := "\uf2c9 " + strings.Join(parts, " | ")
	return MakeBlock("temperature", text, anyUrgent)
}
