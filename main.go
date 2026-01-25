package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"
	"neoden/h2status/widgets"
)

type App struct {
	cfg       *config.Config
	widgets   []widgets.Widget
	bluetooth *widgets.Bluetooth
	battery   *widgets.Battery
	clock     *widgets.Clock
}

func NewApp(cfg *config.Config) *App {
	app := &App{
		cfg:   cfg,
		clock: widgets.NewClock(cfg.Clock.Formats),
	}

	fs := afero.NewOsFs()

	// Initialize widgets based on config
	if cfg.CPU.Enabled {
		app.widgets = append(app.widgets, widgets.NewCPU(cfg.CPU, fs))
	}
	if cfg.RAM.Enabled {
		app.widgets = append(app.widgets, widgets.NewRAM(cfg.RAM, fs))
	}
	if len(cfg.Temperature) > 0 || true { // always try to auto-detect
		app.widgets = append(app.widgets, widgets.NewTemperature(cfg.Temperature, fs))
	}
	if len(cfg.Disk) > 0 {
		app.widgets = append(app.widgets, widgets.NewDisk(cfg.Disk))
	}
	if cfg.WiFi.Enabled {
		app.widgets = append(app.widgets, widgets.NewWiFi(cfg.WiFi))
	}
	if cfg.Network.Enabled {
		app.widgets = append(app.widgets, widgets.NewNetwork(fs))
	}
	if cfg.VPN.Enabled {
		app.widgets = append(app.widgets, widgets.NewVPN(cfg.VPN, fs))
	}
	if cfg.Bluetooth.Enabled {
		app.bluetooth = widgets.NewBluetooth()
		if err := app.bluetooth.Init(); err != nil {
			fmt.Fprintln(os.Stderr, "bluetooth init:", err)
		}
		app.widgets = append(app.widgets, app.bluetooth)
	}
	if cfg.Battery.Enabled {
		app.battery = widgets.NewBattery(cfg.Battery, fs)
		app.widgets = append(app.widgets, app.battery)
	}

	return app
}

func (app *App) Update() {
	for _, w := range app.widgets {
		w.Update()
	}
}

func (app *App) Render() string {
	var blocks []string

	for _, w := range app.widgets {
		if block := w.GetBlock(); block != "" {
			blocks = append(blocks, block)
		}
	}

	// Clock is always last
	blocks = append(blocks, app.clock.GetBlock())

	return "[" + strings.Join(blocks, ",") + "],"
}

func (app *App) HandleClick(event swaybar.ClickEvent) {
	switch event.Name {
	case "power_supply":
		if app.battery != nil {
			app.battery.HandleClick(event.Button)
		}
	case "clock":
		app.clock.HandleClick(event.Button)
	}
}

func HandleClickEvents(ch chan swaybar.ClickEvent) {
	var event swaybar.ClickEvent
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
			continue
		}
		ch <- event
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--print-sensors" {
		widgets.PrintDetectedSensors()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load:", err)
	}

	app := NewApp(cfg)

	swaybar.SendHeader()

	tickCh, err := widgets.StartTicker(1, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "timer:", err)
		os.Exit(1)
	}

	eventsCh := make(chan swaybar.ClickEvent)
	go HandleClickEvents(eventsCh)

	// Get bluetooth updates channel if available
	var bluetoothCh <-chan struct{}
	if app.bluetooth != nil {
		bluetoothCh = app.bluetooth.Updates()
	}

	for {
		select {
		case <-tickCh:
			app.Update()
		case <-bluetoothCh:
			// bluetooth state updated, just re-render
		case event := <-eventsCh:
			app.HandleClick(event)
		}
		fmt.Println(app.Render())
	}
}
