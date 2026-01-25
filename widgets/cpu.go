package widgets

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/afero"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"
)

type CPUSnapshot struct {
	Total   CPUTime
	PerCore []CPUTime
}

type CPUTime struct {
	User   uint64
	Nice   uint64
	System uint64
	Idle   uint64
	Total  uint64
}

type CPUUsage struct {
	Total   float64
	PerCore []float64
}

type CPU struct {
	fs           afero.Fs
	cfg          config.CPUConfig
	prevSnapshot *CPUSnapshot
	history      []CPUUsage
}

func NewCPU(cfg config.CPUConfig, fs afero.Fs) *CPU {
	return &CPU{
		fs:      fs,
		cfg:     cfg,
		history: make([]CPUUsage, 0, cfg.AverageSeconds),
	}
}

func (c *CPU) Update() {
	snapshot, err := c.readSnapshot()
	if err != nil {
		Log.Error("cpu", "error", err)
		return
	}

	if c.prevSnapshot != nil {
		usage := calcUsage(c.prevSnapshot, snapshot)
		c.addToHistory(usage)
	}

	c.prevSnapshot = snapshot
}

func (c *CPU) addToHistory(usage CPUUsage) {
	if len(c.history) >= c.cfg.AverageSeconds {
		copy(c.history, c.history[1:])
		c.history = c.history[:c.cfg.AverageSeconds-1]
	}
	c.history = append(c.history, usage)
}

func (c *CPU) GetAverageUsage() *CPUUsage {
	if len(c.history) < c.cfg.AverageSeconds {
		return nil
	}

	numCores := len(c.history[0].PerCore)
	avg := CPUUsage{
		PerCore: make([]float64, numCores),
	}

	for _, u := range c.history {
		avg.Total += u.Total
		for i, core := range u.PerCore {
			avg.PerCore[i] += core
		}
	}

	n := float64(len(c.history))
	avg.Total /= n
	for i := range avg.PerCore {
		avg.PerCore[i] /= n
	}

	return &avg
}

func (c *CPU) GetBlock() string {
	avg := c.GetAverageUsage()
	if avg == nil {
		return ""
	}

	// Find cores above threshold
	var hotCores int
	var maxCore float64
	for _, core := range avg.PerCore {
		if core > maxCore {
			maxCore = core
		}
		if core > float64(c.cfg.ShowCoreAbove) {
			hotCores++
		}
	}

	showTotal := avg.Total > float64(c.cfg.ShowAbove)
	showCores := hotCores > 0

	if !showTotal && !showCores {
		return ""
	}

	var text string
	if showCores {
		text = fmt.Sprintf("\uf2db %d%% (%d@%d%%)", int(avg.Total), hotCores, int(maxCore))
	} else {
		text = fmt.Sprintf("\uf2db %d%%", int(avg.Total))
	}

	urgent := avg.Total > float64(c.cfg.UrgentAbove)
	return swaybar.MakeBlock("cpu", text, urgent)
}

func (c *CPU) readSnapshot() (*CPUSnapshot, error) {
	f, err := c.fs.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	snapshot := &CPUSnapshot{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		time, err := parseCPUTime(fields[1:])
		if err != nil {
			continue
		}

		if fields[0] == "cpu" {
			snapshot.Total = time
		} else {
			snapshot.PerCore = append(snapshot.PerCore, time)
		}
	}

	return snapshot, scanner.Err()
}

func parseCPUTime(fields []string) (CPUTime, error) {
	var t CPUTime
	var err error

	t.User, err = strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return t, err
	}
	t.Nice, err = strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return t, err
	}
	t.System, err = strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return t, err
	}
	t.Idle, err = strconv.ParseUint(fields[3], 10, 64)
	if err != nil {
		return t, err
	}

	t.Total = t.User + t.Nice + t.System + t.Idle
	return t, nil
}

func calcUsage(prev, curr *CPUSnapshot) CPUUsage {
	usage := CPUUsage{
		PerCore: make([]float64, len(curr.PerCore)),
	}

	usage.Total = calcCoreUsage(prev.Total, curr.Total)

	for i := range curr.PerCore {
		if i < len(prev.PerCore) {
			usage.PerCore[i] = calcCoreUsage(prev.PerCore[i], curr.PerCore[i])
		}
	}

	return usage
}

func calcCoreUsage(prev, curr CPUTime) float64 {
	totalDelta := curr.Total - prev.Total
	if totalDelta == 0 {
		return 0
	}
	idleDelta := curr.Idle - prev.Idle
	return 100 * float64(totalDelta-idleDelta) / float64(totalDelta)
}
