package widgets

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestParseRoute(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "default route via wlan0",
			content: `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	00000000	0102A8C0	0003	0	0	600	00000000	0	0	0
wlan0	0002A8C0	00000000	0001	0	0	600	00FFFFFF	0	0	0
`,
			want: "wlan0",
		},
		{
			name: "default route via eth0",
			content: `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
eth0	00000000	0101A8C0	0003	0	0	100	00000000	0	0	0
eth0	0001A8C0	00000000	0001	0	0	100	00FFFFFF	0	0	0
`,
			want: "eth0",
		},
		{
			name: "no default route",
			content: `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	0002A8C0	00000000	0001	0	0	600	00FFFFFF	0	0	0
`,
			want: "",
		},
		{
			name: "empty file",
			content: `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
`,
			want: "",
		},
		{
			name: "vpn tunnel as default",
			content: `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
tun0	00000000	00000000	0001	0	0	0	80000000	0	0	0
wlan0	0002A8C0	00000000	0001	0	0	600	00FFFFFF	0	0	0
`,
			want: "tun0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with route content
			tmpDir := t.TempDir()
			routeFile := filepath.Join(tmpDir, "route")
			if err := os.WriteFile(routeFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Parse route file manually (same logic as GetDefaultRouteIface)
			result := parseRouteFile(routeFile)

			if result != tt.want {
				t.Errorf("parseRouteFile() = %q, want %q", result, tt.want)
			}
		})
	}
}

// Helper to parse route file (mirrors GetDefaultRouteIface logic)
func parseRouteFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if fields[1] == "00000000" {
			return fields[0]
		}
	}
	return ""
}

func TestNetwork_GetBlock(t *testing.T) {
	tests := []struct {
		name      string
		iface     string
		wantEmpty bool
	}{
		{"has default route - no block", "wlan0", true},
		{"no default route - show warning", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Network{DefaultRouteIface: tt.iface}
			block := n.GetBlock()

			if tt.wantEmpty && block != "" {
				t.Errorf("GetBlock() = %q, want empty", block)
			}
			if !tt.wantEmpty {
				if block == "" {
					t.Error("GetBlock() = empty, want warning block")
				}
				if !contains(block, "No network") {
					t.Errorf("GetBlock() should contain 'No network': %s", block)
				}
				if !contains(block, `"urgent":true`) {
					t.Errorf("GetBlock() should be urgent: %s", block)
				}
			}
		})
	}
}

func TestGetDefaultRouteIface_WithAfero(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	00000000	0102A8C0	0003	0	0	600	00000000	0	0	0
wlan0	0002A8C0	00000000	0001	0	0	600	00FFFFFF	0	0	0
`), 0644)

	result := GetDefaultRouteIface(fs)
	if result != "wlan0" {
		t.Errorf("GetDefaultRouteIface() = %q, want wlan0", result)
	}
}

func TestNetwork_Update(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
eth0	00000000	0101A8C0	0003	0	0	100	00000000	0	0	0
`), 0644)

	n := NewNetwork(fs)
	n.Update()

	if n.DefaultRouteIface != "eth0" {
		t.Errorf("DefaultRouteIface = %q, want eth0", n.DefaultRouteIface)
	}
}

func TestNetwork_Update_NoRoute(t *testing.T) {
	fs := afero.NewMemMapFs() // empty

	n := NewNetwork(fs)
	n.Update()

	if n.DefaultRouteIface != "" {
		t.Errorf("DefaultRouteIface = %q, want empty", n.DefaultRouteIface)
	}
}

func TestGetDefaultRouteIface_NoDefaultRoute(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Route file exists but has no default route (no 00000000 destination)
	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	0002A8C0	00000000	0001	0	0	600	00FFFFFF	0	0	0
`), 0644)

	result := GetDefaultRouteIface(fs)
	if result != "" {
		t.Errorf("GetDefaultRouteIface() = %q, want empty (no default route)", result)
	}
}

func TestGetDefaultRouteIface_MalformedLine(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Route file with malformed line (less than 2 fields)
	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0
eth0	00000000	0101A8C0	0003	0	0	100	00000000	0	0	0
`), 0644)

	result := GetDefaultRouteIface(fs)
	// Should skip malformed line and find default route on eth0
	if result != "eth0" {
		t.Errorf("GetDefaultRouteIface() = %q, want eth0", result)
	}
}
