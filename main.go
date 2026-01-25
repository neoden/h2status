package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

var l = log.New(os.Stderr, "", 0)

var cfg *Config
var batteryState *BatteryState
var bluetoothState *BluetoothState
var cpuState *CPUState
var ramState *RAMState
var diskState *DiskState
var wifiState *WiFiState
var networkState *NetworkState
var vpnState *VPNState

func HandleClickEvents(ch chan ClickEvent) {
	var event ClickEvent
	scanner := bufio.NewScanner(os.Stdin)

	jsonObjects := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		inObject := 0
		for i := 0; i < len(data); i++ {
			switch data[i] {
			case '{':
				if inObject == 0 {
					advance = i
				}
				inObject++
			case '}':
				inObject--
				if inObject == 0 {
					return i + 1, data[advance : i+1], nil
				}
			}
		}
		return
	}
	scanner.Split(jsonObjects)

	for scanner.Scan() {
		if scanner.Text() == "[" {
			continue
		}
		err := json.Unmarshal(scanner.Bytes(), &event)
		if err != nil {
			fmt.Println(scanner.Text())
			continue
		}
		ch <- event
	}
}

func Render() string {
	blocks := []string{}

	if cpuState != nil {
		if cpuBlock := cpuState.GetBlock(); cpuBlock != "" {
			blocks = append(blocks, cpuBlock)
		}
	}
	if ramState != nil {
		if ramBlock := ramState.GetBlock(); ramBlock != "" {
			blocks = append(blocks, ramBlock)
		}
	}
	if diskState != nil {
		if diskBlock := diskState.GetBlock(); diskBlock != "" {
			blocks = append(blocks, diskBlock)
		}
	}
	if wifiState != nil {
		if wifiBlock := wifiState.GetBlock(); wifiBlock != "" {
			blocks = append(blocks, wifiBlock)
		}
	}
	if networkState != nil {
		if networkBlock := networkState.GetBlock(); networkBlock != "" {
			blocks = append(blocks, networkBlock)
		}
	}
	if vpnState != nil {
		if vpnBlock := vpnState.GetBlock(); vpnBlock != "" {
			blocks = append(blocks, vpnBlock)
		}
	}
	if bluetoothState != nil {
		if btBlock := bluetoothState.GetBlock(); btBlock != "" {
			blocks = append(blocks, btBlock)
		}
	}
	if batteryState != nil {
		if batteryBlock := batteryState.GetBlock(); batteryBlock != "" {
			blocks = append(blocks, batteryBlock)
		}
	}
	blocks = append(blocks, GetCurrentTimeBlock(cfg.Clock.Format))

	return "[" + strings.Join(blocks, ",") + "],"
}

func main() {
	var err error
	cfg, err = LoadConfig()
	if err != nil {
		l.Println("config load:", err)
	}

	if cfg.Battery.Enabled {
		batteryState = NewBatteryState()
	}
	if cfg.Bluetooth.Enabled {
		bluetoothState = NewBluetoothState()
		if err := bluetoothState.Init(); err != nil {
			l.Println("bluetooth init:", err)
		}
	}
	if cfg.CPU.Enabled {
		cpuState = NewCPUState(cfg.CPU.AverageSeconds)
	}
	if cfg.RAM.Enabled {
		ramState = NewRAMState()
	}
	if len(cfg.Disk) > 0 {
		diskState = NewDiskState()
	}
	if cfg.WiFi.Enabled {
		wifiState = NewWiFiState()
	}
	if cfg.Network.Enabled {
		networkState = NewNetworkState()
	}
	if cfg.VPN.Enabled {
		vpnState = NewVPNState()
	}

	SendHeader()

	clockCh := make(chan uint64)
	eventsCh := make(chan ClickEvent)

	go StartClock(clockCh, 1, 0)
	go HandleClickEvents(eventsCh)

	for {
		select {
		case <-clockCh:
			if batteryState != nil {
				batteryState.Update()
			}
			if cpuState != nil {
				cpuState.Update()
			}
			if ramState != nil {
				ramState.Update()
			}
			if diskState != nil {
				diskState.Update()
			}
			if wifiState != nil {
				wifiState.Update()
			}
			if networkState != nil {
				networkState.Update()
			}
			if vpnState != nil {
				vpnState.Update()
			}
		case <-bluetoothState.updates:
			// bluetooth state updated, just re-render
		case event := <-eventsCh:
			if event.Name == "power_supply" {
				batteryState.Mode = (batteryState.Mode + 1) % 2
			}
		}
		fmt.Println(Render())
	}
}
