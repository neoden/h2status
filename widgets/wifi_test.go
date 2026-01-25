package widgets

import (
	"errors"
	"testing"

	"neoden/h2status/config"

	"github.com/mdlayher/wifi"
)

// Mock WiFi client for testing
type mockWiFiClient struct {
	interfaces    []*wifi.Interface
	interfacesErr error
	bssMap        map[*wifi.Interface]*wifi.BSS
	bssErr        error
}

func (m *mockWiFiClient) Interfaces() ([]*wifi.Interface, error) {
	return m.interfaces, m.interfacesErr
}

func (m *mockWiFiClient) BSS(iface *wifi.Interface) (*wifi.BSS, error) {
	if m.bssErr != nil {
		return nil, m.bssErr
	}
	if bss, ok := m.bssMap[iface]; ok {
		return bss, nil
	}
	return nil, errors.New("no BSS")
}

func TestWiFi_GetBlock_NotConnected(t *testing.T) {
	w := &WiFi{
		cfg:       config.WiFiConfig{},
		Connected: false,
	}

	block := w.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty when not connected", block)
	}
}

func TestWiFi_GetBlock_Connected(t *testing.T) {
	w := &WiFi{
		cfg: config.WiFiConfig{
			ShowMode:  "always",
			ShowBelow: -50,
		},
		Connected: true,
		SSID:      "TestNetwork",
		Signal:    -45,
	}

	block := w.GetBlock()
	if block == "" {
		t.Error("GetBlock() = empty, want non-empty when connected")
	}
	if !contains(block, "TestNetwork") {
		t.Errorf("GetBlock() should contain SSID: %s", block)
	}
	if !contains(block, "-45") {
		t.Errorf("GetBlock() should contain signal strength: %s", block)
	}
}

func TestWiFi_GetBlock_HomeNetwork(t *testing.T) {
	w := &WiFi{
		cfg: config.WiFiConfig{
			HomeNetworks: []string{"HomeWiFi", "WorkWiFi"},
			ShowMode:     "always",
			ShowBelow:    -50,
		},
		Connected: true,
		SSID:      "HomeWiFi",
		Signal:    -40,
	}

	block := w.GetBlock()
	if block == "" {
		t.Error("GetBlock() = empty, want non-empty")
	}
	// Home network should show only signal, not SSID
	if contains(block, "HomeWiFi") {
		t.Errorf("GetBlock() should not contain home SSID: %s", block)
	}
	if !contains(block, "-40") {
		t.Errorf("GetBlock() should contain signal: %s", block)
	}
}

func TestWiFi_GetBlock_UnknownNetwork(t *testing.T) {
	w := &WiFi{
		cfg: config.WiFiConfig{
			HomeNetworks: []string{"HomeWiFi"},
			ShowMode:     "always",
		},
		Connected: true,
		SSID:      "CoffeeShop",
		Signal:    -60,
	}

	block := w.GetBlock()
	// Unknown network should show SSID
	if !contains(block, "CoffeeShop") {
		t.Errorf("GetBlock() should contain unknown SSID: %s", block)
	}
}

func TestWiFi_GetBlock_ShowMode_WeakSignal(t *testing.T) {
	tests := []struct {
		name      string
		signal    int
		showBelow int
		wantEmpty bool
	}{
		{"strong signal - hide", -40, -50, true},
		{"at threshold - hide", -50, -50, true},
		{"weak signal - show", -60, -50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WiFi{
				cfg: config.WiFiConfig{
					ShowMode:  "weak_signal",
					ShowBelow: tt.showBelow,
				},
				Connected: true,
				SSID:      "Test",
				Signal:    tt.signal,
			}

			block := w.GetBlock()
			if tt.wantEmpty && block != "" {
				t.Errorf("GetBlock() = %q, want empty", block)
			}
			if !tt.wantEmpty && block == "" {
				t.Error("GetBlock() = empty, want non-empty")
			}
		})
	}
}

func TestWiFi_GetBlock_ShowMode_Unknown(t *testing.T) {
	tests := []struct {
		name         string
		ssid         string
		homeNetworks []string
		signal       int
		showBelow    int
		wantEmpty    bool
	}{
		{"home network strong signal - hide", "HomeWiFi", []string{"HomeWiFi"}, -40, -50, true},
		{"home network weak signal - show", "HomeWiFi", []string{"HomeWiFi"}, -60, -50, false},
		{"unknown network strong signal - show", "CoffeeShop", []string{"HomeWiFi"}, -40, -50, false},
		{"unknown network weak signal - show", "CoffeeShop", []string{"HomeWiFi"}, -60, -50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WiFi{
				cfg: config.WiFiConfig{
					HomeNetworks: tt.homeNetworks,
					ShowMode:     "unknown",
					ShowBelow:    tt.showBelow,
				},
				Connected: true,
				SSID:      tt.ssid,
				Signal:    tt.signal,
			}

			block := w.GetBlock()
			if tt.wantEmpty && block != "" {
				t.Errorf("GetBlock() = %q, want empty", block)
			}
			if !tt.wantEmpty && block == "" {
				t.Error("GetBlock() = empty, want non-empty")
			}
		})
	}
}

func TestWiFi_GetBlock_ShowMode_Always(t *testing.T) {
	w := &WiFi{
		cfg: config.WiFiConfig{
			HomeNetworks: []string{"HomeWiFi"},
			ShowMode:     "always",
			ShowBelow:    -50,
		},
		Connected: true,
		SSID:      "HomeWiFi",
		Signal:    -30, // very strong
	}

	block := w.GetBlock()
	if block == "" {
		t.Error("GetBlock() with 'always' mode should never be empty when connected")
	}
}

func TestWiFi_GetBlock_Urgent(t *testing.T) {
	tests := []struct {
		name        string
		signal      int
		urgentBelow int
		wantUrgent  bool
	}{
		{"above threshold - not urgent", -50, -70, false},
		{"at threshold - not urgent", -70, -70, false},
		{"below threshold - urgent", -80, -70, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WiFi{
				cfg: config.WiFiConfig{
					ShowMode:    "always",
					UrgentBelow: tt.urgentBelow,
				},
				Connected: true,
				SSID:      "Test",
				Signal:    tt.signal,
			}

			block := w.GetBlock()
			hasUrgent := contains(block, `"urgent":true`)
			if hasUrgent != tt.wantUrgent {
				t.Errorf("urgent = %v, want %v (block: %s)", hasUrgent, tt.wantUrgent, block)
			}
		})
	}
}

func TestWiFi_Update_NoClient(t *testing.T) {
	w := &WiFi{
		cfg:    config.WiFiConfig{},
		client: nil, // no wifi client
	}

	// Should not panic
	w.Update()

	if w.Connected {
		t.Error("Connected should be false when no client")
	}
}

func TestWiFi_Update_InterfacesError(t *testing.T) {
	mock := &mockWiFiClient{
		interfacesErr: errors.New("nl80211 error"),
	}

	w := &WiFi{
		cfg:    config.WiFiConfig{},
		client: mock,
	}

	w.Update()

	if w.Connected {
		t.Error("Connected should be false on interfaces error")
	}
}

func TestWiFi_Update_Connected(t *testing.T) {
	iface := &wifi.Interface{
		Type: wifi.InterfaceTypeStation,
	}

	mock := &mockWiFiClient{
		interfaces: []*wifi.Interface{iface},
		bssMap: map[*wifi.Interface]*wifi.BSS{
			iface: {
				SSID:   "TestNetwork",
				Signal: -4500, // -45 dBm in 1/100 dBm
			},
		},
	}

	w := &WiFi{
		cfg:    config.WiFiConfig{},
		client: mock,
	}

	w.Update()

	if !w.Connected {
		t.Error("Connected = false, want true")
	}
	if w.SSID != "TestNetwork" {
		t.Errorf("SSID = %q, want 'TestNetwork'", w.SSID)
	}
	if w.Signal != -45 {
		t.Errorf("Signal = %d, want -45", w.Signal)
	}
}

func TestWiFi_Update_SkipsNonStation(t *testing.T) {
	apIface := &wifi.Interface{
		Type: wifi.InterfaceTypeAP, // Access Point, not Station
	}
	stationIface := &wifi.Interface{
		Type: wifi.InterfaceTypeStation,
	}

	mock := &mockWiFiClient{
		interfaces: []*wifi.Interface{apIface, stationIface},
		bssMap: map[*wifi.Interface]*wifi.BSS{
			stationIface: {
				SSID:   "CorrectNetwork",
				Signal: -5000,
			},
		},
	}

	w := &WiFi{
		cfg:    config.WiFiConfig{},
		client: mock,
	}

	w.Update()

	if w.SSID != "CorrectNetwork" {
		t.Errorf("SSID = %q, want 'CorrectNetwork' (should skip AP interface)", w.SSID)
	}
}

func TestWiFi_Update_BSSError(t *testing.T) {
	iface := &wifi.Interface{
		Type: wifi.InterfaceTypeStation,
	}

	mock := &mockWiFiClient{
		interfaces: []*wifi.Interface{iface},
		bssErr:     errors.New("not associated"),
	}

	w := &WiFi{
		cfg:    config.WiFiConfig{},
		client: mock,
	}

	w.Update()

	if w.Connected {
		t.Error("Connected should be false when BSS returns error")
	}
}

func TestWiFi_Update_NoInterfaces(t *testing.T) {
	mock := &mockWiFiClient{
		interfaces: []*wifi.Interface{},
	}

	w := &WiFi{
		cfg:    config.WiFiConfig{},
		client: mock,
	}

	w.Update()

	if w.Connected {
		t.Error("Connected should be false with no interfaces")
	}
}

func TestWiFi_Update_ResetsState(t *testing.T) {
	mock := &mockWiFiClient{
		interfaces: []*wifi.Interface{},
	}

	w := &WiFi{
		cfg:       config.WiFiConfig{},
		client:    mock,
		Connected: true,
		SSID:      "OldNetwork",
		Signal:    -50,
	}

	w.Update()

	if w.Connected {
		t.Error("Connected should be reset to false")
	}
	if w.SSID != "" {
		t.Errorf("SSID = %q, should be reset to empty", w.SSID)
	}
	if w.Signal != 0 {
		t.Errorf("Signal = %d, should be reset to 0", w.Signal)
	}
}
