package main

import (
	"fmt"
	"strings"

	"golang.org/x/sys/unix"
)

type DiskInfo struct {
	Path        string
	Free        uint64
	Total       uint64
	FreeGB      int
	FreePercent int
	Urgent      bool
	ShowBelow   int
	Unit        string
}

type DiskState struct {
	disks []DiskInfo
}

func NewDiskState() *DiskState {
	return &DiskState{
		disks: make([]DiskInfo, len(cfg.Disk)),
	}
}

func (d *DiskState) Update() {
	for i, diskCfg := range cfg.Disk {
		var stat unix.Statfs_t
		if err := unix.Statfs(diskCfg.Path, &stat); err != nil {
			l.Println("disk:", err)
			continue
		}

		free := stat.Bavail * uint64(stat.Bsize)
		total := stat.Blocks * uint64(stat.Bsize)
		freeGB := int(free / (1024 * 1024 * 1024))
		freePercent := 0
		if total > 0 {
			freePercent = int(100 * free / total)
		}

		unit := diskCfg.Unit
		if unit == "" {
			unit = "gb"
		}

		var urgent bool
		if unit == "percent" {
			urgent = freePercent < diskCfg.UrgentBelow
		} else {
			urgent = freeGB < diskCfg.UrgentBelow
		}

		d.disks[i] = DiskInfo{
			Path:        diskCfg.Path,
			Free:        free,
			Total:       total,
			FreeGB:      freeGB,
			FreePercent: freePercent,
			Urgent:      urgent,
			ShowBelow:   diskCfg.ShowBelow,
			Unit:        unit,
		}
	}
}

func (d *DiskState) GetBlock() string {
	var parts []string
	var anyUrgent bool
	showMultiple := len(cfg.Disk) > 1

	for _, disk := range d.disks {
		var value int
		if disk.Unit == "percent" {
			value = disk.FreePercent
		} else {
			value = disk.FreeGB
		}

		if value >= disk.ShowBelow {
			continue
		}

		if disk.Urgent {
			anyUrgent = true
		}

		var valueStr string
		if disk.Unit == "percent" {
			valueStr = fmt.Sprintf("%d%%", disk.FreePercent)
		} else {
			valueStr = formatBytes(disk.Free)
		}

		if showMultiple {
			parts = append(parts, fmt.Sprintf("%s %s", disk.Path, valueStr))
		} else {
			parts = append(parts, valueStr)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	text := "\uf0a0 " + strings.Join(parts, " | ")
	return MakeBlock("disk", text, anyUrgent)
}
