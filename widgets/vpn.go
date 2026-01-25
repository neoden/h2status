package widgets

import (
	"os"
	"path/filepath"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"
)

type VPN struct {
	cfg       config.VPNConfig
	Active    bool
	FullRoute bool // true if default route goes through VPN
	Iface     string
}

func NewVPN(cfg config.VPNConfig) *VPN {
	return &VPN{cfg: cfg}
}

func (v *VPN) Update() {
	v.Active = false
	v.FullRoute = false
	v.Iface = ""

	defaultIface := GetDefaultRouteIface()

	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		Log.Error("vpn", "error", err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !v.matchesPattern(name) {
			continue
		}

		// Check if interface is up
		operstate, err := os.ReadFile(filepath.Join("/sys/class/net", name, "operstate"))
		if err != nil {
			continue
		}
		if string(operstate) != "up\n" && string(operstate) != "unknown\n" {
			continue
		}

		v.Active = true
		v.Iface = name
		if name == defaultIface {
			v.FullRoute = true
		}
		break
	}
}

func (v *VPN) matchesPattern(name string) bool {
	for _, pattern := range v.cfg.Interfaces {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func (v *VPN) GetBlock() string {
	if !v.Active {
		return ""
	}

	var text string
	if v.FullRoute {
		text = "\uf023 " // lock icon - full VPN
	} else {
		text = "\uf09c " // unlock icon - split tunneling
	}

	return swaybar.MakeBlock("vpn", text, false)
}
