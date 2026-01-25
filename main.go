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
var batteryState = &BatteryState{}
var bluetoothState = NewBluetoothState()

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

	if cfg.Bluetooth.Enabled {
		if btBlock := bluetoothState.GetBlock(); btBlock != "" {
			blocks = append(blocks, btBlock)
		}
	}
	if cfg.Battery.Enabled {
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

	SendHeader()

	if cfg.Bluetooth.Enabled {
		if err := bluetoothState.Init(); err != nil {
			l.Println("bluetooth init:", err)
		}
	}

	clockCh := make(chan uint64)
	eventsCh := make(chan ClickEvent)

	go StartClock(clockCh, 1, 0)
	go HandleClickEvents(eventsCh)

	for {
		select {
		case <-clockCh:
			batteryState.Update()
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
