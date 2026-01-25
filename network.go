package main

import (
	"bufio"
	"os"
	"strings"
)

type NetworkState struct {
	DefaultRouteIface string
}

func NewNetworkState() *NetworkState {
	return &NetworkState{}
}

func (n *NetworkState) Update() {
	n.DefaultRouteIface = getDefaultRouteIface()
}

func (n *NetworkState) GetBlock() string {
	if n.DefaultRouteIface != "" {
		return ""
	}
	return MakeBlock("network", "\uf071 No network", true)
}

// getDefaultRouteIface returns the interface name for default route, or empty string if none
func getDefaultRouteIface() string {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		l.Println("network:", err)
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
