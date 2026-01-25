package main

import (
	"fmt"

	"github.com/mdlayher/wifi"
)

type WiFiState struct {
	client    *wifi.Client
	Connected bool
	SSID      string
	Signal    int // dBm
}

func NewWiFiState() *WiFiState {
	client, err := wifi.New()
	if err != nil {
		l.Println("wifi init:", err)
		return &WiFiState{}
	}
	return &WiFiState{client: client}
}

func (w *WiFiState) Update() {
	if w.client == nil {
		return
	}

	w.Connected = false
	w.SSID = ""
	w.Signal = 0

	interfaces, err := w.client.Interfaces()
	if err != nil {
		l.Println("wifi interfaces:", err)
		return
	}

	for _, iface := range interfaces {
		if iface.Type != wifi.InterfaceTypeStation {
			continue
		}

		bss, err := w.client.BSS(iface)
		if err != nil {
			continue
		}

		w.Connected = true
		w.SSID = bss.SSID
		w.Signal = int(bss.Signal) / 100 // signal is in 1/100 dBm
		break
	}
}

func (w *WiFiState) GetBlock() string {
	if !w.Connected {
		return ""
	}

	isHome := false
	for _, home := range cfg.WiFi.HomeNetworks {
		if w.SSID == home {
			isHome = true
			break
		}
	}

	// Hide if home network and signal is good
	if isHome && w.Signal >= cfg.WiFi.ShowBelow {
		return ""
	}

	// Hide if signal is good and no home networks configured (show only weak signal)
	if len(cfg.WiFi.HomeNetworks) == 0 && w.Signal >= cfg.WiFi.ShowBelow {
		return ""
	}

	// Show network name if not home, or signal strength if weak
	var text string
	if !isHome {
		text = fmt.Sprintf("\uf1eb %s %ddBm", w.SSID, w.Signal)
	} else {
		text = fmt.Sprintf("\uf1eb %ddBm", w.Signal)
	}

	urgent := w.Signal < cfg.WiFi.UrgentBelow
	return MakeBlock("wifi", text, urgent)
}
