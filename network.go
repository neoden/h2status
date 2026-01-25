package main

import (
	"bufio"
	"os"
	"strings"
)

type NetworkState struct {
	HasDefaultRoute bool
}

func NewNetworkState() *NetworkState {
	return &NetworkState{}
}

func (n *NetworkState) Update() {
	n.HasDefaultRoute = false

	f, err := os.Open("/proc/net/route")
	if err != nil {
		l.Println("network:", err)
		return
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
			n.HasDefaultRoute = true
			return
		}
	}
}

func (n *NetworkState) GetBlock() string {
	if n.HasDefaultRoute {
		return ""
	}
	return MakeBlock("network", "\uf071 No network", true)
}
