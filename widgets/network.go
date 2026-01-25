package widgets

import (
	"bufio"
	"os"
	"strings"

	"neoden/h2status/swaybar"
)

type Network struct {
	DefaultRouteIface string
}

func NewNetwork() *Network {
	return &Network{}
}

func (n *Network) Update() {
	n.DefaultRouteIface = GetDefaultRouteIface()
}

func (n *Network) GetBlock() string {
	if n.DefaultRouteIface != "" {
		return ""
	}
	return swaybar.MakeBlock("network", "\uf071 No network", true)
}

// GetDefaultRouteIface returns the interface name for default route, or empty string if none
func GetDefaultRouteIface() string {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		Log.Error("network", "error", err)
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
		// Destination 00000000 = default route
		if fields[1] == "00000000" {
			return fields[0]
		}
	}
	return ""
}
