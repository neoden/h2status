package main

import (
	"os"
	"path/filepath"
)

type VPNState struct {
	Active    bool
	FullRoute bool // true if default route goes through VPN
	Iface     string
}

func NewVPNState() *VPNState {
	return &VPNState{}
}

func (v *VPNState) Update() {
	v.Active = false
	v.FullRoute = false
	v.Iface = ""

	defaultIface := getDefaultRouteIface()

	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		l.Println("vpn:", err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !matchesVPNPattern(name) {
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

func matchesVPNPattern(name string) bool {
	for _, pattern := range cfg.VPN.Interfaces {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func (v *VPNState) GetBlock() string {
	if !v.Active {
		return ""
	}

	var text string
	if v.FullRoute {
		text = "\uf023 " // lock icon - full VPN
	} else {
		text = "\uf09c " // unlock icon - split tunneling
	}

	return MakeBlock("vpn", text, false)
}
