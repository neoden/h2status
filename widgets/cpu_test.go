package widgets

import (
	"math"
	"testing"

	"github.com/spf13/afero"

	"neoden/h2status/config"
)

func TestParseCPUTime(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		expected CPUTime
		wantErr  bool
	}{
		{
			name:   "valid input",
			fields: []string{"1000", "200", "300", "4000"},
			expected: CPUTime{
				User:   1000,
				Nice:   200,
				System: 300,
				Idle:   4000,
				Total:  5500,
			},
			wantErr: false,
		},
		{
			name:   "zeros",
			fields: []string{"0", "0", "0", "0"},
			expected: CPUTime{
				User:   0,
				Nice:   0,
				System: 0,
				Idle:   0,
				Total:  0,
			},
			wantErr: false,
		},
		{
			name:   "large numbers",
			fields: []string{"1000000000", "200000000", "300000000", "4000000000"},
			expected: CPUTime{
				User:   1000000000,
				Nice:   200000000,
				System: 300000000,
				Idle:   4000000000,
				Total:  5500000000,
			},
			wantErr: false,
		},
		{
			name:    "invalid user",
			fields:  []string{"abc", "200", "300", "4000"},
			wantErr: true,
		},
		{
			name:    "invalid nice",
			fields:  []string{"1000", "abc", "300", "4000"},
			wantErr: true,
		},
		{
			name:    "invalid system",
			fields:  []string{"1000", "200", "abc", "4000"},
			wantErr: true,
		},
		{
			name:    "invalid idle",
			fields:  []string{"1000", "200", "300", "abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCPUTime(tt.fields)

			if tt.wantErr {
				if err == nil {
					t.Error("parseCPUTime() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseCPUTime() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("parseCPUTime() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestCalcCoreUsage(t *testing.T) {
	tests := []struct {
		name     string
		prev     CPUTime
		curr     CPUTime
		expected float64
	}{
		{
			name:     "50% usage",
			prev:     CPUTime{User: 100, Nice: 0, System: 0, Idle: 100, Total: 200},
			curr:     CPUTime{User: 200, Nice: 0, System: 0, Idle: 200, Total: 400},
			expected: 50.0,
		},
		{
			name:     "100% usage (no idle increase)",
			prev:     CPUTime{User: 100, Nice: 0, System: 0, Idle: 100, Total: 200},
			curr:     CPUTime{User: 300, Nice: 0, System: 0, Idle: 100, Total: 400},
			expected: 100.0,
		},
		{
			name:     "0% usage (only idle increase)",
			prev:     CPUTime{User: 100, Nice: 0, System: 0, Idle: 100, Total: 200},
			curr:     CPUTime{User: 100, Nice: 0, System: 0, Idle: 300, Total: 400},
			expected: 0.0,
		},
		{
			name:     "25% usage",
			prev:     CPUTime{User: 0, Nice: 0, System: 0, Idle: 0, Total: 0},
			curr:     CPUTime{User: 25, Nice: 0, System: 0, Idle: 75, Total: 100},
			expected: 25.0,
		},
		{
			name:     "zero delta (no change)",
			prev:     CPUTime{User: 100, Nice: 0, System: 0, Idle: 100, Total: 200},
			curr:     CPUTime{User: 100, Nice: 0, System: 0, Idle: 100, Total: 200},
			expected: 0.0,
		},
		{
			name:     "mixed usage (user + system)",
			prev:     CPUTime{User: 100, Nice: 0, System: 100, Idle: 800, Total: 1000},
			curr:     CPUTime{User: 200, Nice: 0, System: 200, Idle: 1600, Total: 2000},
			expected: 20.0, // 200 active / 1000 total delta
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calcCoreUsage(tt.prev, tt.curr)

			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("calcCoreUsage() = %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestCalcUsage(t *testing.T) {
	prev := &CPUSnapshot{
		Total: CPUTime{User: 100, Nice: 0, System: 0, Idle: 100, Total: 200},
		PerCore: []CPUTime{
			{User: 50, Nice: 0, System: 0, Idle: 50, Total: 100},
			{User: 50, Nice: 0, System: 0, Idle: 50, Total: 100},
		},
	}
	curr := &CPUSnapshot{
		Total: CPUTime{User: 200, Nice: 0, System: 0, Idle: 200, Total: 400},
		PerCore: []CPUTime{
			{User: 150, Nice: 0, System: 0, Idle: 50, Total: 200}, // 100% (only user increased)
			{User: 50, Nice: 0, System: 0, Idle: 150, Total: 200}, // 0% (only idle increased)
		},
	}

	usage := calcUsage(prev, curr)

	if math.Abs(usage.Total-50.0) > 0.001 {
		t.Errorf("calcUsage() Total = %f, want 50.0", usage.Total)
	}

	if len(usage.PerCore) != 2 {
		t.Fatalf("calcUsage() PerCore length = %d, want 2", len(usage.PerCore))
	}

	if math.Abs(usage.PerCore[0]-100.0) > 0.001 {
		t.Errorf("calcUsage() PerCore[0] = %f, want 100.0", usage.PerCore[0])
	}

	if math.Abs(usage.PerCore[1]-0.0) > 0.001 {
		t.Errorf("calcUsage() PerCore[1] = %f, want 0.0", usage.PerCore[1])
	}
}

func TestCPU_AddToHistory(t *testing.T) {
	cfg := config.CPUConfig{AverageSeconds: 3}
	cpu := NewCPU(cfg, afero.NewMemMapFs())

	// Add first usage
	cpu.addToHistory(CPUUsage{Total: 10, PerCore: []float64{10}})
	if len(cpu.history) != 1 {
		t.Errorf("history length = %d, want 1", len(cpu.history))
	}

	// Add second usage
	cpu.addToHistory(CPUUsage{Total: 20, PerCore: []float64{20}})
	if len(cpu.history) != 2 {
		t.Errorf("history length = %d, want 2", len(cpu.history))
	}

	// Add third usage (at capacity)
	cpu.addToHistory(CPUUsage{Total: 30, PerCore: []float64{30}})
	if len(cpu.history) != 3 {
		t.Errorf("history length = %d, want 3", len(cpu.history))
	}

	// Add fourth usage (should evict first)
	cpu.addToHistory(CPUUsage{Total: 40, PerCore: []float64{40}})
	if len(cpu.history) != 3 {
		t.Errorf("history length = %d, want 3", len(cpu.history))
	}

	// Check oldest was evicted
	if cpu.history[0].Total != 20 {
		t.Errorf("history[0].Total = %f, want 20", cpu.history[0].Total)
	}
	if cpu.history[2].Total != 40 {
		t.Errorf("history[2].Total = %f, want 40", cpu.history[2].Total)
	}
}

func TestCPU_GetAverageUsage(t *testing.T) {
	cfg := config.CPUConfig{AverageSeconds: 3}
	cpu := NewCPU(cfg, afero.NewMemMapFs())

	// Not enough history
	avg := cpu.GetAverageUsage()
	if avg != nil {
		t.Error("GetAverageUsage() should return nil when history is not full")
	}

	// Fill history
	cpu.addToHistory(CPUUsage{Total: 10, PerCore: []float64{5, 15}})
	cpu.addToHistory(CPUUsage{Total: 20, PerCore: []float64{10, 30}})
	cpu.addToHistory(CPUUsage{Total: 30, PerCore: []float64{15, 45}})

	avg = cpu.GetAverageUsage()
	if avg == nil {
		t.Fatal("GetAverageUsage() returned nil")
	}

	// Average of 10, 20, 30 = 20
	if math.Abs(avg.Total-20.0) > 0.001 {
		t.Errorf("GetAverageUsage().Total = %f, want 20.0", avg.Total)
	}

	// Average of core 0: (5 + 10 + 15) / 3 = 10
	if math.Abs(avg.PerCore[0]-10.0) > 0.001 {
		t.Errorf("GetAverageUsage().PerCore[0] = %f, want 10.0", avg.PerCore[0])
	}

	// Average of core 1: (15 + 30 + 45) / 3 = 30
	if math.Abs(avg.PerCore[1]-30.0) > 0.001 {
		t.Errorf("GetAverageUsage().PerCore[1] = %f, want 30.0", avg.PerCore[1])
	}
}

func TestCPU_Update(t *testing.T) {
	fs := afero.NewMemMapFs()

	// First snapshot
	afero.WriteFile(fs, "/proc/stat", []byte(`cpu  1000 200 300 4000 0 0 0 0 0 0
cpu0 500 100 150 2000 0 0 0 0 0 0
cpu1 500 100 150 2000 0 0 0 0 0 0
`), 0644)

	cfg := config.CPUConfig{AverageSeconds: 1}
	cpu := NewCPU(cfg, fs)
	cpu.Update()

	// First update just saves snapshot, no history yet
	if cpu.prevSnapshot == nil {
		t.Fatal("prevSnapshot should not be nil after first Update")
	}
	if len(cpu.history) != 0 {
		t.Errorf("history length = %d, want 0 after first update", len(cpu.history))
	}

	// Second snapshot - 50% usage (idle doubled, total doubled)
	afero.WriteFile(fs, "/proc/stat", []byte(`cpu  2000 400 600 8000 0 0 0 0 0 0
cpu0 1000 200 300 4000 0 0 0 0 0 0
cpu1 1000 200 300 4000 0 0 0 0 0 0
`), 0644)

	cpu.Update()

	if len(cpu.history) != 1 {
		t.Fatalf("history length = %d, want 1 after second update", len(cpu.history))
	}

	// Total: (2000-1000 + 400-200 + 600-300) / (11000-5500) = 1500/5500 ≈ 27.3%
	expectedUsage := 100 * float64(1500) / float64(5500)
	if math.Abs(cpu.history[0].Total-expectedUsage) > 0.1 {
		t.Errorf("history[0].Total = %f, want ~%f", cpu.history[0].Total, expectedUsage)
	}
}

func TestCPU_ReadSnapshot(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/proc/stat", []byte(`cpu  1000 200 300 4000 0 0 0 0 0 0
cpu0 500 100 150 2000 0 0 0 0 0 0
cpu1 500 100 150 2000 0 0 0 0 0 0
intr 12345
`), 0644)

	cpu := NewCPU(config.CPUConfig{}, fs)
	snapshot, err := cpu.readSnapshot()

	if err != nil {
		t.Fatalf("readSnapshot() error: %v", err)
	}

	if snapshot.Total.User != 1000 {
		t.Errorf("Total.User = %d, want 1000", snapshot.Total.User)
	}
	if snapshot.Total.Idle != 4000 {
		t.Errorf("Total.Idle = %d, want 4000", snapshot.Total.Idle)
	}
	if len(snapshot.PerCore) != 2 {
		t.Errorf("PerCore length = %d, want 2", len(snapshot.PerCore))
	}
}

func TestCPU_Update_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs() // empty

	cpu := NewCPU(config.CPUConfig{}, fs)
	cpu.Update() // should not panic

	if cpu.prevSnapshot != nil {
		t.Error("prevSnapshot should be nil when file not found")
	}
}

func TestCPU_GetBlock(t *testing.T) {
	tests := []struct {
		name          string
		total         float64
		perCore       []float64
		showAbove     int
		showCoreAbove int
		urgentAbove   int
		wantEmpty     bool
		wantUrgent    bool
		wantCores     bool // should show "X@Y%" format
	}{
		{
			name:          "below all thresholds - hidden",
			total:         30,
			perCore:       []float64{30, 30},
			showAbove:     50,
			showCoreAbove: 95,
			urgentAbove:   95,
			wantEmpty:     true,
		},
		{
			name:          "total above threshold - shown",
			total:         60,
			perCore:       []float64{60, 60},
			showAbove:     50,
			showCoreAbove: 95,
			urgentAbove:   95,
			wantEmpty:     false,
			wantCores:     false,
		},
		{
			name:          "core above threshold - shown with cores",
			total:         40,
			perCore:       []float64{30, 98},
			showAbove:     50,
			showCoreAbove: 95,
			urgentAbove:   95,
			wantEmpty:     false,
			wantCores:     true,
		},
		{
			name:          "urgent",
			total:         98,
			perCore:       []float64{98, 98},
			showAbove:     50,
			showCoreAbove: 95,
			urgentAbove:   95,
			wantEmpty:     false,
			wantUrgent:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.CPUConfig{
				ShowAbove:      tt.showAbove,
				ShowCoreAbove:  tt.showCoreAbove,
				UrgentAbove:    tt.urgentAbove,
				AverageSeconds: 1,
			}
			cpu := &CPU{
				cfg:     cfg,
				history: []CPUUsage{{Total: tt.total, PerCore: tt.perCore}},
			}

			block := cpu.GetBlock()

			if tt.wantEmpty && block != "" {
				t.Errorf("GetBlock() = %q, want empty", block)
			}
			if !tt.wantEmpty && block == "" {
				t.Error("GetBlock() = empty, want non-empty")
			}
			if tt.wantUrgent && !contains(block, `"urgent":true`) {
				t.Errorf("GetBlock() should be urgent: %s", block)
			}
			if tt.wantCores && !contains(block, "@") {
				t.Errorf("GetBlock() should contain core info: %s", block)
			}
		})
	}
}

func TestCPU_GetBlock_NoHistory(t *testing.T) {
	cpu := &CPU{
		cfg:     config.CPUConfig{AverageSeconds: 3},
		history: []CPUUsage{}, // not enough history
	}

	block := cpu.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty when no history", block)
	}
}
