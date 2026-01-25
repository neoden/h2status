package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type RAMState struct {
	Total     uint64
	Available uint64
	Used      uint64
	Percent   int
}

func NewRAMState() *RAMState {
	return &RAMState{}
}

func (r *RAMState) Update() {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		l.Println("ram:", err)
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

func (r *RAMState) GetBlock() string {
	if r.Percent <= cfg.RAM.ShowAbove {
		return ""
	}

	text := fmt.Sprintf("\uf538 %d%% %s", r.Percent, formatBytes(r.Used))
	urgent := r.Percent > cfg.RAM.UrgentAbove
	return MakeBlock("ram", text, urgent)
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(b)/float64(div), "KMGTPE"[exp])
}
