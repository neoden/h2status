package widgets

import (
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestNewBluetooth(t *testing.T) {
	b := NewBluetooth()

	if b.devices == nil {
		t.Error("devices map should be initialized")
	}
	if b.updates == nil {
		t.Error("updates channel should be initialized")
	}
	if b.conn != nil {
		t.Error("conn should be nil before Init()")
	}
}

func TestBluetooth_Update(t *testing.T) {
	b := NewBluetooth()
	// Update is a no-op (bluetooth updates via dbus signals)
	b.Update() // should not panic
}

func TestBluetooth_Updates(t *testing.T) {
	b := NewBluetooth()
	ch := b.Updates()

	if ch == nil {
		t.Error("Updates() should return non-nil channel")
	}
}

func TestBluetooth_GetBlock_NoDevices(t *testing.T) {
	b := NewBluetooth()

	block := b.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty when no devices", block)
	}
}

func TestBluetooth_GetBlock_OneDevice(t *testing.T) {
	b := NewBluetooth()
	b.devices["/org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF"] = &BluetoothDevice{
		Path:      "/org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF",
		Name:      "Sony WH-1000XM4",
		Connected: true,
		IsAudio:   true,
	}

	block := b.GetBlock()
	if block == "" {
		t.Error("GetBlock() = empty, want non-empty with connected device")
	}
	if !contains(block, "Sony WH-1000XM4") {
		t.Errorf("GetBlock() should contain device name: %s", block)
	}
	if !contains(block, "bluetooth") {
		t.Errorf("GetBlock() should have 'bluetooth' name: %s", block)
	}
}

func TestBluetooth_GetBlock_MultipleDevices(t *testing.T) {
	b := NewBluetooth()
	now := time.Now()

	b.devices["/dev1"] = &BluetoothDevice{
		Path:        "/dev1",
		Name:        "Headphones",
		Connected:   true,
		IsAudio:     true,
		ConnectedAt: now,
	}
	b.devices["/dev2"] = &BluetoothDevice{
		Path:        "/dev2",
		Name:        "Speaker",
		Connected:   true,
		IsAudio:     true,
		ConnectedAt: now.Add(time.Second),
	}

	block := b.GetBlock()
	// Should show first device name and +1 for additional
	if !contains(block, "Headphones") {
		t.Errorf("GetBlock() should contain first device: %s", block)
	}
	if !contains(block, "+1") {
		t.Errorf("GetBlock() should show +1 for additional device: %s", block)
	}
}

func TestBluetooth_GetBlock_DisconnectedDevice(t *testing.T) {
	b := NewBluetooth()
	b.devices["/dev1"] = &BluetoothDevice{
		Path:      "/dev1",
		Name:      "Headphones",
		Connected: false, // not connected
		IsAudio:   true,
	}

	block := b.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty for disconnected device", block)
	}
}

func TestBluetooth_GetBlock_NonAudioDevice(t *testing.T) {
	b := NewBluetooth()
	b.devices["/dev1"] = &BluetoothDevice{
		Path:      "/dev1",
		Name:      "Keyboard",
		Connected: true,
		IsAudio:   false, // not audio
	}

	block := b.GetBlock()
	if block != "" {
		t.Errorf("GetBlock() = %q, want empty for non-audio device", block)
	}
}

func TestBluetooth_GetConnectedAudioDevices_Empty(t *testing.T) {
	b := NewBluetooth()

	devices := b.GetConnectedAudioDevices()
	if len(devices) != 0 {
		t.Errorf("GetConnectedAudioDevices() = %d devices, want 0", len(devices))
	}
}

func TestBluetooth_GetConnectedAudioDevices_Filters(t *testing.T) {
	b := NewBluetooth()

	b.devices["/dev1"] = &BluetoothDevice{Name: "Audio1", Connected: true, IsAudio: true}
	b.devices["/dev2"] = &BluetoothDevice{Name: "Audio2", Connected: false, IsAudio: true}  // disconnected
	b.devices["/dev3"] = &BluetoothDevice{Name: "Keyboard", Connected: true, IsAudio: false} // not audio
	b.devices["/dev4"] = &BluetoothDevice{Name: "Audio3", Connected: true, IsAudio: true}

	devices := b.GetConnectedAudioDevices()
	if len(devices) != 2 {
		t.Errorf("GetConnectedAudioDevices() = %d devices, want 2", len(devices))
	}
}

func TestBluetooth_GetConnectedAudioDevices_SortByConnectionTime(t *testing.T) {
	b := NewBluetooth()
	now := time.Now()

	b.devices["/dev1"] = &BluetoothDevice{
		Name:        "Second",
		Connected:   true,
		IsAudio:     true,
		ConnectedAt: now.Add(time.Second),
	}
	b.devices["/dev2"] = &BluetoothDevice{
		Name:        "First",
		Connected:   true,
		IsAudio:     true,
		ConnectedAt: now,
	}

	devices := b.GetConnectedAudioDevices()
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].Name != "First" {
		t.Errorf("first device = %q, want 'First' (earliest connected)", devices[0].Name)
	}
	if devices[1].Name != "Second" {
		t.Errorf("second device = %q, want 'Second'", devices[1].Name)
	}
}

func TestBluetooth_GetConnectedAudioDevices_SortAlphabeticalFallback(t *testing.T) {
	b := NewBluetooth()
	sameTime := time.Now()

	b.devices["/dev1"] = &BluetoothDevice{
		Name:        "Zebra",
		Connected:   true,
		IsAudio:     true,
		ConnectedAt: sameTime,
	}
	b.devices["/dev2"] = &BluetoothDevice{
		Name:        "Alpha",
		Connected:   true,
		IsAudio:     true,
		ConnectedAt: sameTime,
	}

	devices := b.GetConnectedAudioDevices()
	if devices[0].Name != "Alpha" {
		t.Errorf("first device = %q, want 'Alpha' (alphabetical)", devices[0].Name)
	}
}

func TestBluetooth_notifyUpdate(t *testing.T) {
	b := NewBluetooth()

	// First notify should succeed
	b.notifyUpdate()

	select {
	case <-b.updates:
		// good
	default:
		t.Error("expected update notification")
	}
}

func TestBluetooth_notifyUpdate_NonBlocking(t *testing.T) {
	b := NewBluetooth()

	// Fill the channel (buffer size is 1)
	b.notifyUpdate()

	// Second notify should not block
	done := make(chan bool)
	go func() {
		b.notifyUpdate()
		done <- true
	}()

	select {
	case <-done:
		// good, didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("notifyUpdate() blocked when channel was full")
	}
}

func TestBluetooth_updateDevice_NewDevice(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF")

	props := map[string]dbus.Variant{
		"Alias":     dbus.MakeVariant("Test Device"),
		"Connected": dbus.MakeVariant(true),
		"UUIDs":     dbus.MakeVariant([]string{audioSinkUUID}),
	}

	b.updateDevice(path, props)

	dev, exists := b.devices[path]
	if !exists {
		t.Fatal("device should be created")
	}
	if dev.Name != "Test Device" {
		t.Errorf("Name = %q, want 'Test Device'", dev.Name)
	}
	if !dev.Connected {
		t.Error("Connected = false, want true")
	}
	if !dev.IsAudio {
		t.Error("IsAudio = false, want true")
	}
}

func TestBluetooth_updateDevice_ExistingDevice(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/dev1")

	// Create initial device
	b.devices[path] = &BluetoothDevice{
		Path:      path,
		Name:      "Old Name",
		Connected: false,
	}

	// Update with new props
	props := map[string]dbus.Variant{
		"Alias":     dbus.MakeVariant("New Name"),
		"Connected": dbus.MakeVariant(true),
	}

	b.updateDevice(path, props)

	dev := b.devices[path]
	if dev.Name != "New Name" {
		t.Errorf("Name = %q, want 'New Name'", dev.Name)
	}
	if !dev.Connected {
		t.Error("Connected = false, want true")
	}
}

func TestBluetooth_updateDevice_ConnectionTime(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/dev1")

	// Create disconnected device
	b.devices[path] = &BluetoothDevice{
		Path:      path,
		Connected: false,
	}

	before := time.Now()

	// Connect the device
	props := map[string]dbus.Variant{
		"Connected": dbus.MakeVariant(true),
	}
	b.updateDevice(path, props)

	after := time.Now()

	dev := b.devices[path]
	if dev.ConnectedAt.Before(before) || dev.ConnectedAt.After(after) {
		t.Errorf("ConnectedAt = %v, should be between %v and %v", dev.ConnectedAt, before, after)
	}
}

func TestBluetooth_updateDevice_PartialProps(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/dev1")

	// Only update Alias
	props := map[string]dbus.Variant{
		"Alias": dbus.MakeVariant("Device Name"),
	}

	b.updateDevice(path, props)

	dev := b.devices[path]
	if dev.Name != "Device Name" {
		t.Errorf("Name = %q, want 'Device Name'", dev.Name)
	}
	if dev.Connected {
		t.Error("Connected should be false (not in props)")
	}
}

func TestBluetooth_processManagedObjects(t *testing.T) {
	b := NewBluetooth()

	managed := map[dbus.ObjectPath]map[string]map[string]dbus.Variant{
		"/org/bluez/hci0/dev_AA": {
			"org.bluez.Device1": {
				"Alias":     dbus.MakeVariant("Headphones"),
				"Connected": dbus.MakeVariant(true),
				"UUIDs":     dbus.MakeVariant([]string{audioSinkUUID}),
			},
		},
		"/org/bluez/hci0/dev_BB": {
			"org.bluez.Device1": {
				"Alias":     dbus.MakeVariant("Speaker"),
				"Connected": dbus.MakeVariant(false),
			},
		},
		"/org/bluez/hci0/dev_CC": {
			"org.bluez.Adapter1": {}, // not a device, should be ignored
		},
	}

	b.processManagedObjects(managed)

	if len(b.devices) != 2 {
		t.Errorf("devices count = %d, want 2", len(b.devices))
	}

	dev := b.devices["/org/bluez/hci0/dev_AA"]
	if dev == nil {
		t.Fatal("device AA not found")
	}
	if dev.Name != "Headphones" {
		t.Errorf("Name = %q, want 'Headphones'", dev.Name)
	}
	if !dev.Connected {
		t.Error("Connected = false, want true")
	}
	if !dev.IsAudio {
		t.Error("IsAudio = false, want true")
	}
}

func TestBluetooth_processManagedObjects_Empty(t *testing.T) {
	b := NewBluetooth()

	b.processManagedObjects(nil)

	if len(b.devices) != 0 {
		t.Errorf("devices count = %d, want 0", len(b.devices))
	}
}

func TestBluetooth_handleSignal_PropertiesChanged(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/org/bluez/hci0/dev_AA")

	// Create existing device
	b.devices[path] = &BluetoothDevice{
		Path:      path,
		Name:      "Old Name",
		Connected: false,
	}

	signal := &dbus.Signal{
		Path: path,
		Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
		Body: []interface{}{
			"org.bluez.Device1",
			map[string]dbus.Variant{
				"Alias":     dbus.MakeVariant("New Name"),
				"Connected": dbus.MakeVariant(true),
			},
			[]string{},
		},
	}

	b.handleSignal(signal)

	dev := b.devices[path]
	if dev.Name != "New Name" {
		t.Errorf("Name = %q, want 'New Name'", dev.Name)
	}
	if !dev.Connected {
		t.Error("Connected = false, want true")
	}

	// Should have sent update notification
	select {
	case <-b.updates:
		// good
	default:
		t.Error("expected update notification")
	}
}

func TestBluetooth_handleSignal_PropertiesChanged_WrongInterface(t *testing.T) {
	b := NewBluetooth()

	signal := &dbus.Signal{
		Path: "/org/bluez/hci0",
		Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
		Body: []interface{}{
			"org.bluez.Adapter1", // not Device1
			map[string]dbus.Variant{},
			[]string{},
		},
	}

	b.handleSignal(signal)

	if len(b.devices) != 0 {
		t.Error("should not create device for non-Device1 interface")
	}
}

func TestBluetooth_handleSignal_InterfacesAdded(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/org/bluez/hci0/dev_NEW")

	signal := &dbus.Signal{
		Name: "org.freedesktop.DBus.ObjectManager.InterfacesAdded",
		Body: []interface{}{
			path,
			map[string]map[string]dbus.Variant{
				"org.bluez.Device1": {
					"Alias":     dbus.MakeVariant("New Device"),
					"Connected": dbus.MakeVariant(true),
				},
			},
		},
	}

	b.handleSignal(signal)

	if len(b.devices) != 1 {
		t.Fatalf("devices count = %d, want 1", len(b.devices))
	}

	dev := b.devices[path]
	if dev.Name != "New Device" {
		t.Errorf("Name = %q, want 'New Device'", dev.Name)
	}
}

func TestBluetooth_handleSignal_InterfacesRemoved(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/org/bluez/hci0/dev_AA")

	// Create existing device
	b.devices[path] = &BluetoothDevice{
		Path: path,
		Name: "To Be Removed",
	}

	signal := &dbus.Signal{
		Name: "org.freedesktop.DBus.ObjectManager.InterfacesRemoved",
		Body: []interface{}{
			path,
			[]string{"org.bluez.Device1"},
		},
	}

	b.handleSignal(signal)

	if len(b.devices) != 0 {
		t.Error("device should be removed")
	}
}

func TestBluetooth_handleSignal_InterfacesRemoved_WrongInterface(t *testing.T) {
	b := NewBluetooth()
	path := dbus.ObjectPath("/org/bluez/hci0/dev_AA")

	b.devices[path] = &BluetoothDevice{Path: path}

	signal := &dbus.Signal{
		Name: "org.freedesktop.DBus.ObjectManager.InterfacesRemoved",
		Body: []interface{}{
			path,
			[]string{"org.bluez.MediaControl1"}, // not Device1
		},
	}

	b.handleSignal(signal)

	if len(b.devices) != 1 {
		t.Error("device should NOT be removed for non-Device1 interface")
	}
}

func TestBluetooth_handleSignal_InvalidBody(t *testing.T) {
	b := NewBluetooth()

	// Should not panic on invalid signal bodies
	signals := []*dbus.Signal{
		// PropertiesChanged - various invalid bodies
		{Name: "org.freedesktop.DBus.Properties.PropertiesChanged", Body: nil},
		{Name: "org.freedesktop.DBus.Properties.PropertiesChanged", Body: []interface{}{"only one"}},
		{Name: "org.freedesktop.DBus.Properties.PropertiesChanged", Body: []interface{}{123, "wrong type for iface"}},
		{Name: "org.freedesktop.DBus.Properties.PropertiesChanged", Body: []interface{}{"org.bluez.Device1", "wrong type for props"}},

		// InterfacesAdded - various invalid bodies
		{Name: "org.freedesktop.DBus.ObjectManager.InterfacesAdded", Body: []interface{}{}},
		{Name: "org.freedesktop.DBus.ObjectManager.InterfacesAdded", Body: []interface{}{"not a path", map[string]map[string]dbus.Variant{}}},
		{Name: "org.freedesktop.DBus.ObjectManager.InterfacesAdded", Body: []interface{}{dbus.ObjectPath("/path"), "not a map"}},

		// InterfacesRemoved - various invalid bodies
		{Name: "org.freedesktop.DBus.ObjectManager.InterfacesRemoved", Body: []interface{}{}},
		{Name: "org.freedesktop.DBus.ObjectManager.InterfacesRemoved", Body: []interface{}{"not a path", []string{}}},
		{Name: "org.freedesktop.DBus.ObjectManager.InterfacesRemoved", Body: []interface{}{dbus.ObjectPath("/path"), "not a slice"}},

		// Unknown signal
		{Name: "unknown.signal", Body: []interface{}{}},
	}

	for _, signal := range signals {
		b.handleSignal(signal) // should not panic
	}

	// No devices should be created from invalid signals
	if len(b.devices) != 0 {
		t.Errorf("devices = %d, want 0 (no valid signals)", len(b.devices))
	}
}
