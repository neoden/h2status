package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/godbus/dbus/v5"
)

// Audio Sink (A2DP) profile UUID - identifies devices that can receive audio (headphones, speakers)
const audioSinkUUID = "0000110b-0000-1000-8000-00805f9b34fb"

type BluetoothDevice struct {
	Path        dbus.ObjectPath
	Name        string
	Connected   bool
	ConnectedAt time.Time
	IsAudio     bool
}

type BluetoothState struct {
	conn    *dbus.Conn
	devices map[dbus.ObjectPath]*BluetoothDevice
	updates chan struct{}
}

func NewBluetoothState() *BluetoothState {
	return &BluetoothState{
		devices: make(map[dbus.ObjectPath]*BluetoothDevice),
		updates: make(chan struct{}, 1),
	}
}

func (b *BluetoothState) Init() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	b.conn = conn

	// Get all existing devices
	obj := conn.Object("org.bluez", "/")
	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err = obj.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&managed)
	if err != nil {
		return err
	}

	for path, ifaces := range managed {
		if device, ok := ifaces["org.bluez.Device1"]; ok {
			b.updateDevice(path, device)
		}
	}

	// Subscribe to property changes
	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.bluez',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged'")

	// Subscribe to new/removed devices
	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.bluez',interface='org.freedesktop.DBus.ObjectManager'")

	go b.listenSignals()

	return nil
}

func (b *BluetoothState) updateDevice(path dbus.ObjectPath, props map[string]dbus.Variant) {
	dev, exists := b.devices[path]
	if !exists {
		dev = &BluetoothDevice{Path: path}
		b.devices[path] = dev
	}

	if name, ok := props["Alias"]; ok {
		dev.Name = name.Value().(string)
	}
	if connected, ok := props["Connected"]; ok {
		wasConnected := dev.Connected
		dev.Connected = connected.Value().(bool)
		if dev.Connected && !wasConnected {
			dev.ConnectedAt = time.Now()
		}
	}
	if uuids, ok := props["UUIDs"]; ok {
		for _, uuid := range uuids.Value().([]string) {
			if uuid == audioSinkUUID {
				dev.IsAudio = true
				break
			}
		}
	}
}

func (b *BluetoothState) listenSignals() {
	ch := make(chan *dbus.Signal, 10)
	b.conn.Signal(ch)

	for signal := range ch {
		switch signal.Name {
		case "org.freedesktop.DBus.Properties.PropertiesChanged":
			if len(signal.Body) < 2 {
				continue
			}
			iface := signal.Body[0].(string)
			if iface != "org.bluez.Device1" {
				continue
			}
			changed := signal.Body[1].(map[string]dbus.Variant)
			b.updateDevice(signal.Path, changed)
			b.notifyUpdate()

		case "org.freedesktop.DBus.ObjectManager.InterfacesAdded":
			if len(signal.Body) < 2 {
				continue
			}
			path := signal.Body[0].(dbus.ObjectPath)
			ifaces := signal.Body[1].(map[string]map[string]dbus.Variant)
			if device, ok := ifaces["org.bluez.Device1"]; ok {
				b.updateDevice(path, device)
				b.notifyUpdate()
			}

		case "org.freedesktop.DBus.ObjectManager.InterfacesRemoved":
			if len(signal.Body) < 2 {
				continue
			}
			path := signal.Body[0].(dbus.ObjectPath)
			ifaces := signal.Body[1].([]string)
			for _, iface := range ifaces {
				if iface == "org.bluez.Device1" {
					delete(b.devices, path)
					b.notifyUpdate()
					break
				}
			}
		}
	}
}

func (b *BluetoothState) notifyUpdate() {
	select {
	case b.updates <- struct{}{}:
	default:
	}
}

func (b *BluetoothState) GetConnectedAudioDevices() []*BluetoothDevice {
	var result []*BluetoothDevice
	for _, dev := range b.devices {
		if dev.Connected && dev.IsAudio {
			result = append(result, dev)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		// Sort by connection time, earliest first; fallback to alphabetical
		if result[i].ConnectedAt.Equal(result[j].ConnectedAt) {
			return result[i].Name < result[j].Name
		}
		return result[i].ConnectedAt.Before(result[j].ConnectedAt)
	})
	return result
}

func (b *BluetoothState) GetBlock() string {
	devices := b.GetConnectedAudioDevices()
	if len(devices) == 0 {
		return ""
	}

	text := "\uf025 " + devices[0].Name
	if len(devices) > 1 {
		text += fmt.Sprintf(" +%d", len(devices)-1)
	}

	return MakeBlock("bluetooth", text, false)
}
