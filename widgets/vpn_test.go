package widgets

import (
	"testing"

	"github.com/spf13/afero"

	"neoden/h2status/config"
)

func TestVPN_MatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		iface    string
		want     bool
	}{
		{"tun0 matches tun*", []string{"tun*"}, "tun0", true},
		{"tun1 matches tun*", []string{"tun*"}, "tun1", true},
		{"wg0 matches wg*", []string{"wg*"}, "wg0", true},
		{"tap0 matches tap*", []string{"tap*"}, "tap0", true},
		{"eth0 does not match vpn patterns", []string{"tun*", "wg*", "tap*"}, "eth0", false},
		{"wlan0 does not match vpn patterns", []string{"tun*", "wg*", "tap*"}, "wlan0", false},
		{"cscotun0 matches cscotun*", []string{"cscotun*"}, "cscotun0", true},
		{"exact match", []string{"tun0"}, "tun0", true},
		{"exact match fails", []string{"tun0"}, "tun1", false},
		{"multiple patterns - first matches", []string{"tun*", "wg*"}, "tun0", true},
		{"multiple patterns - second matches", []string{"tun*", "wg*"}, "wg0", true},
		{"empty patterns", []string{}, "tun0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &VPN{cfg: config.VPNConfig{Interfaces: tt.patterns}}
			result := v.matchesPattern(tt.iface)

			if result != tt.want {
				t.Errorf("matchesPattern(%q) = %v, want %v", tt.iface, result, tt.want)
			}
		})
	}
}

func TestVPN_GetBlock(t *testing.T) {
	tests := []struct {
		name      string
		active    bool
		fullRoute bool
		wantEmpty bool
		wantIcon  string
	}{
		{"inactive - hidden", false, false, true, ""},
		{"active full route - lock icon", true, true, false, "\uf023"},
		{"active split tunnel - unlock icon", true, false, false, "\uf09c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &VPN{
				Active:    tt.active,
				FullRoute: tt.fullRoute,
			}

			block := v.GetBlock()

			if tt.wantEmpty && block != "" {
				t.Errorf("GetBlock() = %q, want empty", block)
			}
			if !tt.wantEmpty {
				if block == "" {
					t.Error("GetBlock() = empty, want non-empty")
				}
				if tt.wantIcon != "" && !contains(block, tt.wantIcon) {
					t.Errorf("GetBlock() should contain icon %q: %s", tt.wantIcon, block)
				}
			}
		})
	}
}

func TestVPN_GetBlock_NeverUrgent(t *testing.T) {
	v := &VPN{Active: true, FullRoute: true}
	block := v.GetBlock()

	if contains(block, `"urgent":true`) {
		t.Errorf("VPN block should never be urgent: %s", block)
	}
}

func TestNewVPN(t *testing.T) {
	cfg := config.VPNConfig{
		Enabled:    true,
		Interfaces: []string{"tun*", "wg*"},
	}

	v := NewVPN(cfg, afero.NewMemMapFs())

	if len(v.cfg.Interfaces) != 2 {
		t.Errorf("cfg.Interfaces length = %d, want 2", len(v.cfg.Interfaces))
	}
	if v.Active {
		t.Error("Active should be false initially")
	}
	if v.FullRoute {
		t.Error("FullRoute should be false initially")
	}
}

func TestVPN_Update(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create route file with tun0 as default
	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
tun0	00000000	00000000	0001	0	0	0	80000000	0	0	0
`), 0644)

	// Create network interfaces
	fs.MkdirAll("/sys/class/net/tun0", 0755)
	afero.WriteFile(fs, "/sys/class/net/tun0/operstate", []byte("up\n"), 0644)

	fs.MkdirAll("/sys/class/net/wlan0", 0755)
	afero.WriteFile(fs, "/sys/class/net/wlan0/operstate", []byte("up\n"), 0644)

	cfg := config.VPNConfig{Interfaces: []string{"tun*", "wg*"}}
	v := NewVPN(cfg, fs)
	v.Update()

	if !v.Active {
		t.Error("Active = false, want true")
	}
	if v.Iface != "tun0" {
		t.Errorf("Iface = %q, want tun0", v.Iface)
	}
	if !v.FullRoute {
		t.Error("FullRoute = false, want true (tun0 is default route)")
	}
}

func TestVPN_Update_SplitTunnel(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Default route is wlan0, not VPN
	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	00000000	0102A8C0	0003	0	0	600	00000000	0	0	0
`), 0644)

	// VPN interface is up but not default route
	fs.MkdirAll("/sys/class/net/wg0", 0755)
	afero.WriteFile(fs, "/sys/class/net/wg0/operstate", []byte("unknown\n"), 0644)

	cfg := config.VPNConfig{Interfaces: []string{"wg*"}}
	v := NewVPN(cfg, fs)
	v.Update()

	if !v.Active {
		t.Error("Active = false, want true")
	}
	if v.FullRoute {
		t.Error("FullRoute = true, want false (split tunnel)")
	}
}

func TestVPN_Update_NoVPN(t *testing.T) {
	fs := afero.NewMemMapFs()

	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	00000000	0102A8C0	0003	0	0	600	00000000	0	0	0
`), 0644)

	// No VPN interfaces
	fs.MkdirAll("/sys/class/net/wlan0", 0755)
	afero.WriteFile(fs, "/sys/class/net/wlan0/operstate", []byte("up\n"), 0644)

	cfg := config.VPNConfig{Interfaces: []string{"tun*", "wg*"}}
	v := NewVPN(cfg, fs)
	v.Update()

	if v.Active {
		t.Error("Active = true, want false")
	}
}

func TestVPN_Update_InterfaceDown(t *testing.T) {
	fs := afero.NewMemMapFs()

	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	00000000	0102A8C0	0003	0	0	600	00000000	0	0	0
`), 0644)

	// VPN interface exists but is down
	fs.MkdirAll("/sys/class/net/tun0", 0755)
	afero.WriteFile(fs, "/sys/class/net/tun0/operstate", []byte("down\n"), 0644)

	cfg := config.VPNConfig{Interfaces: []string{"tun*"}}
	v := NewVPN(cfg, fs)
	v.Update()

	if v.Active {
		t.Error("Active = true, want false (interface is down)")
	}
}

func TestVPN_Update_NoOperstateFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	afero.WriteFile(fs, "/proc/net/route", []byte(`Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	00000000	0102A8C0	0003	0	0	600	00000000	0	0	0
`), 0644)

	// VPN interface directory exists but no operstate file
	fs.MkdirAll("/sys/class/net/tun0", 0755)
	// No operstate file

	cfg := config.VPNConfig{Interfaces: []string{"tun*"}}
	v := NewVPN(cfg, fs)
	v.Update()

	if v.Active {
		t.Error("Active = true, want false (no operstate file)")
	}
}
