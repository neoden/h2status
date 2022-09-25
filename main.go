package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

var l = log.New(os.Stderr, "", 0)

const BATTERY_STATE_MODE_PECENTAGE = 0
const BATTERY_STATE_MODE_REMAINING_TIME = 1

type BatteryState struct {
	Percentage int
	Status     string
	EnergyFull int
	EnergyNow  int
	PowerNow   int
	Remaining  time.Duration
	IsCharging bool
	Mode       int
}

var batteryState = BatteryState{}

func (b *BatteryState) Update() {
	path := "/sys/class/power_supply/BAT0/"

	percentage, err := ReadInt(path + "capacity")
	if err != nil {
		l.Println(err)
	}
	b.Percentage = percentage

	status, err := ioutil.ReadFile(path + "status")
	if err != nil {
		l.Println(err)
	}
	b.Status = strings.Trim(string(status), "\n")
	b.IsCharging = b.Status == "Charging"

	power_now, err := ReadInt(path + "power_now")
	if err != nil || power_now == 0 {
		l.Println(err)
		return
	}
	b.PowerNow = power_now

	energy_now, err := ReadInt(path + "energy_now")
	if err != nil {
		l.Println(err)
		return
	}
	b.EnergyNow = energy_now

	energy_full, err := ReadInt(path + "energy_full")
	if err != nil {
		l.Println(err)
		return
	}
	b.EnergyFull = energy_full

	if b.IsCharging {
		b.Remaining = time.Duration(((energy_full - energy_now) * 1000 / power_now)) * time.Hour / 1000
	} else {
		b.Remaining = time.Duration(b.EnergyNow*1000/b.PowerNow) * time.Hour / 1000
	}
}

func StartClock(ch chan uint64, seconds int64, nanoseconds int64) {
	fd, err := unix.TimerfdCreate(unix.CLOCK_REALTIME, 0)

	if err != nil {
		fmt.Println(err)
		close(ch)
	}

	err = unix.TimerfdSettime(fd, unix.TFD_TIMER_ABSTIME, &unix.ItimerSpec{
		Interval: unix.Timespec{
			Sec:  seconds,
			Nsec: nanoseconds,
		},
		Value: unix.Timespec{
			Sec:  seconds,
			Nsec: nanoseconds,
		},
	}, nil)

	if err != nil {
		fmt.Println(err)
		close(ch)
	}

	file := os.NewFile(uintptr(fd), "timerfd")
	defer file.Close()

	buffer := make([]byte, 8)

	for {
		_, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
			}
			break
		}
		ch <- binary.BigEndian.Uint64(buffer)
	}
}

func GetCurrentTimeBlock(format string) string {
	dt := time.Now()
	return MakeBlock("time", dt.Format(format), false)
}

func ReadInt(file string) (int, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return 0, err
	}
	value, _ := strconv.ParseInt(strings.Trim(string(content), "\n"), 10, 32)
	return int(value), nil
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%d:%02d", h, m)
}

func (b *BatteryState) GetBatteryStatusBlock() string {
	var symbols [6]string = [6]string{"\uf244", "\uf243", "\uf242", "\uf241", "\uf240", "\uf240"}
	var symbol = symbols[0]
	var text = ""

	if b.IsCharging {
		symbol = "\uf1e6"
	} else {
		symbol = symbols[b.Percentage/20]
	}

	if b.Mode == BATTERY_STATE_MODE_PECENTAGE {
		text = fmt.Sprintf("%s %d%%", symbol, b.Percentage)
	} else if b.Mode == BATTERY_STATE_MODE_REMAINING_TIME {
		text = fmt.Sprintf("%s %s", symbol, fmtDuration(b.Remaining))
	}

	return MakeBlock("power_supply", text, b.Percentage < 10)
}

func HandleClickEvents(ch chan ClickEvent, f *os.File) {
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
		// skip first line
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
	return "[" +
		batteryState.GetBatteryStatusBlock() + "," +
		GetCurrentTimeBlock("15:04") +
		"],"
}

func main() {
	SendHeader()

	f, _ := os.Create("/home/xtal/filename.ext")
	defer f.Close()
	defer os.Remove("/home/xtal/filename.ext")

	ch := make(chan uint64)
	events_ch := make(chan ClickEvent)

	go StartClock(ch, 1, 0)
	go HandleClickEvents(events_ch, f)

	// batteryState.Mode = BATTERY_STATE_MODE_REMAINING_TIME

	for {
		select {
		case <-ch:
			batteryState.Update()
		case event := <-events_ch:
			if event.Name == "power_supply" {
				batteryState.Mode = (batteryState.Mode + 1) % 2
			}
		}
		fmt.Println(Render())
	}
}
