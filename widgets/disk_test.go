package widgets

import (
	"testing"

	"neoden/h2status/config"
)

func TestDiskInfo_Calculations(t *testing.T) {
	tests := []struct {
		name        string
		free        uint64
		total       uint64
		wantFreeGB  int
		wantFreePct int
	}{
		{"100GB free of 500GB", 100 * 1024 * 1024 * 1024, 500 * 1024 * 1024 * 1024, 100, 20},
		{"50GB free of 500GB", 50 * 1024 * 1024 * 1024, 500 * 1024 * 1024 * 1024, 50, 10},
		{"1TB free of 2TB", 1024 * 1024 * 1024 * 1024, 2 * 1024 * 1024 * 1024 * 1024, 1024, 50},
		{"empty disk", 0, 500 * 1024 * 1024 * 1024, 0, 0},
		{"full disk", 500 * 1024 * 1024 * 1024, 500 * 1024 * 1024 * 1024, 500, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			freeGB := int(tt.free / (1024 * 1024 * 1024))
			var freePct int
			if tt.total > 0 {
				freePct = int(100 * tt.free / tt.total)
			}

			if freeGB != tt.wantFreeGB {
				t.Errorf("freeGB = %d, want %d", freeGB, tt.wantFreeGB)
			}
			if freePct != tt.wantFreePct {
				t.Errorf("freePct = %d, want %d", freePct, tt.wantFreePct)
			}
		})
	}
}

func TestDisk_GetBlock_SingleDisk(t *testing.T) {
	tests := []struct {
		name       string
		freeGB     int
		freePct    int
		showBelow  int
		urgBelow   int
		unit       string
		wantEmpty  bool
		wantUrgent bool
	}{
		{"GB mode - above threshold", 50, 10, 20, 5, "gb", true, false},
		{"GB mode - below threshold", 15, 3, 20, 5, "gb", false, false},
		{"GB mode - urgent", 3, 1, 20, 5, "gb", false, true},
		{"percent mode - above threshold", 50, 25, 20, 5, "percent", true, false},
		{"percent mode - below threshold", 50, 15, 20, 5, "percent", false, false},
		{"percent mode - urgent", 50, 3, 20, 5, "percent", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Disk{
				cfgs: []config.DiskConfig{{
					Path:        "/",
					ShowBelow:   tt.showBelow,
					UrgentBelow: tt.urgBelow,
					Unit:        tt.unit,
				}},
				disks: []DiskInfo{{
					Path:        "/",
					FreeGB:      tt.freeGB,
					FreePercent: tt.freePct,
					Free:        uint64(tt.freeGB) * 1024 * 1024 * 1024,
					ShowBelow:   tt.showBelow,
					Unit:        tt.unit,
					Urgent:      (tt.unit == "percent" && tt.freePct < tt.urgBelow) || (tt.unit != "percent" && tt.freeGB < tt.urgBelow),
				}},
			}

			block := d.GetBlock()

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

func TestDisk_GetBlock_MultipleDisks(t *testing.T) {
	d := &Disk{
		cfgs: []config.DiskConfig{
			{Path: "/", ShowBelow: 20, UrgentBelow: 5, Unit: "gb"},
			{Path: "/home", ShowBelow: 50, UrgentBelow: 10, Unit: "gb"},
		},
		disks: []DiskInfo{
			{Path: "/", FreeGB: 15, FreePercent: 10, Free: 15 * 1024 * 1024 * 1024, ShowBelow: 20, Unit: "gb", Urgent: false},
			{Path: "/home", FreeGB: 30, FreePercent: 20, Free: 30 * 1024 * 1024 * 1024, ShowBelow: 50, Unit: "gb", Urgent: false},
		},
	}

	block := d.GetBlock()

	// Should contain both paths
	if !contains(block, "/") {
		t.Errorf("GetBlock() should contain root path: %s", block)
	}
	if !contains(block, "/home") {
		t.Errorf("GetBlock() should contain /home path: %s", block)
	}
}

func TestDisk_GetBlock_AllHidden(t *testing.T) {
	d := &Disk{
		cfgs: []config.DiskConfig{
			{Path: "/", ShowBelow: 20, UrgentBelow: 5, Unit: "gb"},
		},
		disks: []DiskInfo{
			{Path: "/", FreeGB: 50, FreePercent: 30, ShowBelow: 20, Unit: "gb", Urgent: false},
		},
	}

	block := d.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty when above threshold", block)
	}
}

func TestDisk_UrgentLogic(t *testing.T) {
	tests := []struct {
		name     string
		unit     string
		freeGB   int
		freePct  int
		urgBelow int
		want     bool
	}{
		{"gb mode urgent", "gb", 3, 10, 5, true},
		{"gb mode not urgent", "gb", 10, 10, 5, false},
		{"percent mode urgent", "percent", 100, 3, 5, true},
		{"percent mode not urgent", "percent", 100, 10, 5, false},
		{"empty unit defaults to gb - urgent", "", 3, 50, 5, true},
		{"empty unit defaults to gb - not urgent", "", 10, 50, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := tt.unit
			if unit == "" {
				unit = "gb"
			}

			var urgent bool
			if unit == "percent" {
				urgent = tt.freePct < tt.urgBelow
			} else {
				urgent = tt.freeGB < tt.urgBelow
			}

			if urgent != tt.want {
				t.Errorf("urgent = %v, want %v", urgent, tt.want)
			}
		})
	}
}

func TestNewDisk(t *testing.T) {
	cfgs := []config.DiskConfig{
		{Path: "/", ShowBelow: 20, UrgentBelow: 5},
		{Path: "/home", ShowBelow: 50, UrgentBelow: 10},
	}

	d := NewDisk(cfgs)

	if len(d.cfgs) != 2 {
		t.Errorf("cfgs length = %d, want 2", len(d.cfgs))
	}
	if len(d.disks) != 2 {
		t.Errorf("disks length = %d, want 2", len(d.disks))
	}
	if d.cfgs[0].Path != "/" {
		t.Errorf("cfgs[0].Path = %q, want /", d.cfgs[0].Path)
	}
}

func TestNewDisk_Empty(t *testing.T) {
	d := NewDisk(nil)

	if d.cfgs != nil {
		t.Errorf("cfgs = %v, want nil", d.cfgs)
	}
	if len(d.disks) != 0 {
		t.Errorf("disks length = %d, want 0", len(d.disks))
	}
}

func TestDisk_Update_RealFS(t *testing.T) {
	// Test on real filesystem - / should always exist
	cfgs := []config.DiskConfig{
		{Path: "/", ShowBelow: 1000000, UrgentBelow: 1, Unit: "gb"},
	}

	d := NewDisk(cfgs)
	d.Update()

	if d.disks[0].Total == 0 {
		t.Error("Total = 0, want non-zero for root filesystem")
	}
	if d.disks[0].Path != "/" {
		t.Errorf("Path = %q, want /", d.disks[0].Path)
	}
	// Free should be less than or equal to Total
	if d.disks[0].Free > d.disks[0].Total {
		t.Errorf("Free (%d) > Total (%d)", d.disks[0].Free, d.disks[0].Total)
	}
}

func TestDisk_Update_NonexistentPath(t *testing.T) {
	cfgs := []config.DiskConfig{
		{Path: "/nonexistent/path/that/should/not/exist", ShowBelow: 100, UrgentBelow: 10},
	}

	d := NewDisk(cfgs)
	d.Update() // Should not panic, just log error

	// Disk info should remain zero-valued
	if d.disks[0].Total != 0 {
		t.Errorf("Total = %d, want 0 for nonexistent path", d.disks[0].Total)
	}
}

func TestDisk_Update_UnitDefault(t *testing.T) {
	cfgs := []config.DiskConfig{
		{Path: "/", ShowBelow: 1000000, UrgentBelow: 1, Unit: ""}, // empty unit
	}

	d := NewDisk(cfgs)
	d.Update()

	if d.disks[0].Unit != "gb" {
		t.Errorf("Unit = %q, want 'gb' (default)", d.disks[0].Unit)
	}
}

func TestDisk_Update_PercentUnit(t *testing.T) {
	cfgs := []config.DiskConfig{
		{Path: "/", ShowBelow: 100, UrgentBelow: 1, Unit: "percent"},
	}

	d := NewDisk(cfgs)
	d.Update()

	if d.disks[0].Unit != "percent" {
		t.Errorf("Unit = %q, want 'percent'", d.disks[0].Unit)
	}
	if d.disks[0].FreePercent < 0 || d.disks[0].FreePercent > 100 {
		t.Errorf("FreePercent = %d, should be 0-100", d.disks[0].FreePercent)
	}
}
