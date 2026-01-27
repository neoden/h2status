package widgets

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/afero"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"
	"neoden/h2status/util"
)

type TempSensor struct {
	Path        string
	Label       string
	ShowAbove   int
	UrgentAbove int
	Value       int // smoothed temperature in C
	ema         *util.EMA
}

type Temperature struct {
	fs      afero.Fs
	sensors []TempSensor
}

func NewTemperature(cfgs []config.TemperatureConfig, fs afero.Fs) *Temperature {
	t := &Temperature{fs: fs}

	if len(cfgs) > 0 {
		// Use configured sensors
		for _, tc := range cfgs {
			interval := tc.SmoothingIntervalSeconds
			if interval == 0 {
				interval = DefaultSmoothingInterval
			}
			t.sensors = append(t.sensors, TempSensor{
				Path:        tc.Path,
				Label:       tc.Label,
				ShowAbove:   tc.ShowAbove,
				UrgentAbove: tc.UrgentAbove,
				ema:         util.NewEMA(interval),
			})
		}
	} else {
		// Auto-detect
		t.sensors = t.autoDetectSensors()
	}

	return t
}

func PrintDetectedSensors() {
	PrintDetectedSensorsTo(afero.NewOsFs(), os.Stdout)
}

func PrintDetectedSensorsTo(fs afero.Fs, w io.Writer) {
	fmt.Fprintln(w, "# Detected temperature sensors:")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# hwmon sensors:")
	fmt.Fprintln(w)

	// hwmon sensors
	hwmons, _ := afero.Glob(fs, "/sys/class/hwmon/hwmon*")
	for _, hwmon := range hwmons {
		nameBytes, err := afero.ReadFile(fs, filepath.Join(hwmon, "name"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameBytes))

		// Find all temp inputs
		temps, _ := afero.Glob(fs, filepath.Join(hwmon, "temp*_input"))
		for _, tempPath := range temps {
			// Extract temp number (temp1_input -> 1)
			base := filepath.Base(tempPath)
			num := strings.TrimSuffix(strings.TrimPrefix(base, "temp"), "_input")

			label := name
			labelPath := filepath.Join(hwmon, fmt.Sprintf("temp%s_label", num))
			if labelBytes, err := afero.ReadFile(fs, labelPath); err == nil {
				label = strings.TrimSpace(string(labelBytes))
			}

			fmt.Fprintln(w, "[[temperature]]")
			fmt.Fprintf(w, "path = %q\n", tempPath)
			fmt.Fprintf(w, "label = %q  # %s\n", label, name)
			fmt.Fprintln(w, "show_above = 75")
			fmt.Fprintln(w, "urgent_above = 90")
			fmt.Fprintln(w)
		}
	}

	fmt.Fprintln(w, "# thermal_zone sensors:")
	fmt.Fprintln(w)

	// thermal_zone sensors
	zones, _ := afero.Glob(fs, "/sys/class/thermal/thermal_zone*")
	for _, zone := range zones {
		typeBytes, err := afero.ReadFile(fs, filepath.Join(zone, "type"))
		if err != nil {
			continue
		}
		zoneType := strings.TrimSpace(string(typeBytes))
		tempPath := filepath.Join(zone, "temp")

		fmt.Fprintln(w, "[[temperature]]")
		fmt.Fprintf(w, "path = %q\n", tempPath)
		fmt.Fprintf(w, "label = %q\n", zoneType)
		fmt.Fprintln(w, "show_above = 75")
		fmt.Fprintln(w, "urgent_above = 90")
		fmt.Fprintln(w)
	}
}

func (t *Temperature) autoDetectSensors() []TempSensor {
	var sensors []TempSensor

	// Priority order for hwmon/thermal names
	priorities := []string{"coretemp", "k10temp", "x86_pkg", "cpu", "pkg", "acpitz"}

	// Try hwmon first (more accurate)
	hwmons, _ := afero.Glob(t.fs, "/sys/class/hwmon/hwmon*")
	for _, prio := range priorities {
		for _, hwmon := range hwmons {
			nameBytes, err := afero.ReadFile(t.fs, filepath.Join(hwmon, "name"))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(string(nameBytes))

			if strings.Contains(strings.ToLower(name), prio) {
				// Find temp input file (usually temp1_input)
				tempFile := filepath.Join(hwmon, "temp1_input")
				if _, err := t.fs.Stat(tempFile); err != nil {
					continue
				}

				// Try to get label
				label := name
				labelBytes, err := afero.ReadFile(t.fs, filepath.Join(hwmon, "temp1_label"))
				if err == nil {
					label = strings.TrimSpace(string(labelBytes))
				}

				sensors = append(sensors, TempSensor{
					Path:        tempFile,
					Label:       label,
					ShowAbove:   75,
					UrgentAbove: 90,
					ema:         util.NewEMA(DefaultSmoothingInterval),
				})
				return sensors
			}
		}
	}

	// Fallback to thermal_zone
	zones, _ := afero.Glob(t.fs, "/sys/class/thermal/thermal_zone*")
	for _, prio := range priorities {
		for _, zone := range zones {
			typeBytes, err := afero.ReadFile(t.fs, filepath.Join(zone, "type"))
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
					ema:         util.NewEMA(DefaultSmoothingInterval),
				})
				return sensors
			}
		}
	}

	return sensors
}

func (t *Temperature) Update() {
	for i := range t.sensors {
		data, err := afero.ReadFile(t.fs, t.sensors[i].Path)
		if err != nil {
			Log.Error("temperature", "error", err)
			continue
		}

		val, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			continue
		}

		// Convert from millidegrees to degrees and apply EMA smoothing
		rawTemp := float64(val) / 1000
		t.sensors[i].Value = int(math.Round(t.sensors[i].ema.Update(rawTemp)))
	}
}

func (t *Temperature) GetBlock() string {
	var parts []string
	var anyUrgent bool
	showMultiple := len(t.sensors) > 1

	for _, sensor := range t.sensors {
		if !sensor.ema.Ready() || sensor.Value <= sensor.ShowAbove {
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
	return swaybar.MakeBlock("temperature", text, anyUrgent)
}
