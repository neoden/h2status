package widgets

import (
	"testing"

	"github.com/spf13/afero"

	"neoden/h2status/config"
)

func TestRAM_PercentCalculation(t *testing.T) {
	tests := []struct {
		name      string
		total     uint64
		available uint64
		wantUsed  uint64
		wantPct   int
	}{
		{"50% used", 16 * 1024 * 1024 * 1024, 8 * 1024 * 1024 * 1024, 8 * 1024 * 1024 * 1024, 50},
		{"75% used", 16 * 1024 * 1024 * 1024, 4 * 1024 * 1024 * 1024, 12 * 1024 * 1024 * 1024, 75},
		{"0% used", 16 * 1024 * 1024 * 1024, 16 * 1024 * 1024 * 1024, 0, 0},
		{"100% used", 16 * 1024 * 1024 * 1024, 0, 16 * 1024 * 1024 * 1024, 100},
		{"real world", 32712568 * 1024, 20000000 * 1024, 12712568 * 1024, 38},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RAM{
				Total:     tt.total,
				Available: tt.available,
			}
			r.Used = r.Total - r.Available
			if r.Total > 0 {
				r.Percent = int(100 * r.Used / r.Total)
			}

			if r.Used != tt.wantUsed {
				t.Errorf("Used = %d, want %d", r.Used, tt.wantUsed)
			}
			if r.Percent != tt.wantPct {
				t.Errorf("Percent = %d, want %d", r.Percent, tt.wantPct)
			}
		})
	}
}

func TestRAM_GetBlock(t *testing.T) {
	tests := []struct {
		name      string
		percent   int
		used      uint64
		showAbove int
		urgent    int
		wantEmpty bool
		wantUrg   bool
	}{
		{"below threshold - hidden", 50, 8 * 1024 * 1024 * 1024, 70, 90, true, false},
		{"at threshold - hidden", 70, 8 * 1024 * 1024 * 1024, 70, 90, true, false},
		{"above threshold - shown", 75, 12 * 1024 * 1024 * 1024, 70, 90, false, false},
		{"urgent", 95, 15 * 1024 * 1024 * 1024, 70, 90, false, true},
		{"at urgent threshold - not urgent", 90, 14 * 1024 * 1024 * 1024, 70, 90, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RAM{
				cfg:     config.RAMConfig{ShowAbove: tt.showAbove, UrgentAbove: tt.urgent},
				Percent: tt.percent,
				Used:    tt.used,
			}

			block := r.GetBlock()

			if tt.wantEmpty && block != "" {
				t.Errorf("GetBlock() = %q, want empty", block)
			}
			if !tt.wantEmpty && block == "" {
				t.Error("GetBlock() = empty, want non-empty")
			}
			if !tt.wantEmpty && tt.wantUrg && !contains(block, `"urgent":true`) {
				t.Errorf("GetBlock() should be urgent: %s", block)
			}
		})
	}
}

func TestRAM_Update(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/proc/meminfo", []byte(`MemTotal:       16000000 kB
MemFree:         1000000 kB
MemAvailable:    8000000 kB
Buffers:          500000 kB
Cached:          4000000 kB
`), 0644)

	r := NewRAM(config.RAMConfig{}, fs)
	r.Update()

	expectedTotal := uint64(16000000 * 1024)
	expectedAvailable := uint64(8000000 * 1024)
	expectedUsed := expectedTotal - expectedAvailable
	expectedPercent := 50

	if r.Total != expectedTotal {
		t.Errorf("Total = %d, want %d", r.Total, expectedTotal)
	}
	if r.Available != expectedAvailable {
		t.Errorf("Available = %d, want %d", r.Available, expectedAvailable)
	}
	if r.Used != expectedUsed {
		t.Errorf("Used = %d, want %d", r.Used, expectedUsed)
	}
	if r.Percent != expectedPercent {
		t.Errorf("Percent = %d, want %d", r.Percent, expectedPercent)
	}
}

func TestRAM_Update_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs() // empty

	r := NewRAM(config.RAMConfig{}, fs)
	r.Update() // should not panic

	if r.Total != 0 {
		t.Errorf("Total = %d, want 0", r.Total)
	}
}
