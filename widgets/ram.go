package widgets

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"
)

type RAM struct {
	cfg       config.RAMConfig
	Total     uint64
	Available uint64
	Used      uint64
	Percent   int
}

func NewRAM(cfg config.RAMConfig) *RAM {
	return &RAM{cfg: cfg}
}

func (r *RAM) Update() {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		Log.Error("ram", "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			r.Total = value * 1024
		case "MemAvailable:":
			r.Available = value * 1024
		}
	}

	r.Used = r.Total - r.Available
	if r.Total > 0 {
		r.Percent = int(100 * r.Used / r.Total)
	}
}

func (r *RAM) GetBlock() string {
	if r.Percent <= r.cfg.ShowAbove {
		return ""
	}

	text := fmt.Sprintf("\uf538 %d%% %s", r.Percent, FormatBytes(r.Used))
	urgent := r.Percent > r.cfg.UrgentAbove
	return swaybar.MakeBlock("ram", text, urgent)
}
