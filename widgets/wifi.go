package widgets

import (
	"fmt"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"

	"github.com/mdlayher/wifi"
)

type WiFi struct {
	cfg       config.WiFiConfig
	client    *wifi.Client
	Connected bool
	SSID      string
	Signal    int // dBm
}

func NewWiFi(cfg config.WiFiConfig) *WiFi {
	client, err := wifi.New()
	if err != nil {
		Log.Error("wifi init", "error", err)
		return &WiFi{cfg: cfg}
	}
	return &WiFi{cfg: cfg, client: client}
}

func (w *WiFi) Update() {
	if w.client == nil {
		return
	}

	w.Connected = false
	w.SSID = ""
	w.Signal = 0

	interfaces, err := w.client.Interfaces()
	if err != nil {
		Log.Error("wifi interfaces", "error", err)
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

func (w *WiFi) GetBlock() string {
	if !w.Connected {
		return ""
	}

	isHome := false
	for _, home := range w.cfg.HomeNetworks {
		if w.SSID == home {
			isHome = true
			break
		}
	}

	// Determine visibility based on show_mode
	switch w.cfg.ShowMode {
	case "always":
		// always show
	case "unknown":
		// show if unknown network OR weak signal
		if isHome && w.Signal >= w.cfg.ShowBelow {
			return ""
		}
	default: // "weak_signal"
		// show only when signal is weak
		if w.Signal >= w.cfg.ShowBelow {
			return ""
		}
	}

	// Show network name if not home, or just signal if home
	var text string
	if !isHome {
		text = fmt.Sprintf("\uf1eb %s %ddBm", w.SSID, w.Signal)
	} else {
		text = fmt.Sprintf("\uf1eb %ddBm", w.Signal)
	}

	urgent := w.Signal < w.cfg.UrgentBelow
	return swaybar.MakeBlock("wifi", text, urgent)
}
