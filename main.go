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
	cfg           *config.Config
	widgets       []widgets.Widget
	clickHandlers map[string]widgets.ClickHandler
	updates       chan struct{}
}

func NewApp(cfg *config.Config) *App {
	app := &App{
		cfg:           cfg,
		clickHandlers: make(map[string]widgets.ClickHandler),
		updates:       make(chan struct{}, 1),
	}

	fs := afero.NewOsFs()

	// Initialize widgets based on config
	if cfg.CPU.Enabled {
		app.addWidget(widgets.NewCPU(cfg.CPU, fs))
	}
	if cfg.RAM.Enabled {
		app.addWidget(widgets.NewRAM(cfg.RAM, fs))
	}
	if len(cfg.Temperature) > 0 || true { // always try to auto-detect
		app.addWidget(widgets.NewTemperature(cfg.Temperature, fs))
	}
	if len(cfg.Disk) > 0 {
		app.addWidget(widgets.NewDisk(cfg.Disk))
	}
	if cfg.WiFi.Enabled {
		app.addWidget(widgets.NewWiFi(cfg.WiFi))
	}
	if cfg.Network.Enabled {
		app.addWidget(widgets.NewNetwork(fs))
	}
	if cfg.VPN.Enabled {
		app.addWidget(widgets.NewVPN(cfg.VPN, fs))
	}
	if cfg.Bluetooth.Enabled {
		bt := widgets.NewBluetooth(app.updates)
		if err := bt.Init(); err != nil {
			fmt.Fprintln(os.Stderr, "bluetooth init:", err)
		}
		app.addWidget(bt)
	}
	if cfg.Battery.Enabled {
		app.addWidget(widgets.NewBattery(cfg.Battery, fs))
	}
	if cfg.Clock.Enabled {
		app.addWidget(widgets.NewClock(cfg.Clock))
	}

	return app
}

func (app *App) addWidget(w widgets.Widget) {
	app.widgets = append(app.widgets, w)
	if ch, ok := w.(widgets.ClickHandler); ok {
		app.clickHandlers[ch.ClickName()] = ch
	}
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

	return "[" + strings.Join(blocks, ",") + "],"
}

func (app *App) HandleClick(event swaybar.ClickEvent) {
	if ch, ok := app.clickHandlers[event.Name]; ok {
		ch.HandleClick(event.Button)
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

	for {
		select {
		case <-tickCh:
			app.Update()
		case <-app.updates:
			// async widget updated, just re-render
		case event := <-eventsCh:
			app.HandleClick(event)
		}
		fmt.Println(app.Render())
	}
}
