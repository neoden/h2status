package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

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

func PrintHeader() {
	header_str, err := json.Marshal(Header{
		Version: 1,
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(string(header_str))
	fmt.Println("[")
}

func MakeBlock(text string, urgent bool) string {
	block := Body{
		FullText: text,
		Urgent:   urgent,
	}
	block_str, err := json.Marshal(block)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return string(block_str)
}

func GetCurrentTimeBlock(format string) string {
	dt := time.Now()
	return MakeBlock(dt.Format(format), false)
}

func ReadInt64(file string) (int64, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return 0, err
	}
	value, _ := strconv.ParseInt(strings.Trim(string(content), "\n"), 10, 64)
	return value, nil
}

func GetBatteryStatusBlock() string {
	var symbol [5]string = [5]string{"\uf244", "\uf243", "\uf242", "\uf241", "\uf240"}

	percentage, err := ReadInt64("/sys/class/power_supply/BAT0/capacity")
	if err != nil {
		return "??"
	}

	energy_now, err := ReadInt64("/sys/class/power_supply/BAT0/energy_now")
	if err != nil {
		return "??"
	}

	power_now, err := ReadInt64("/sys/class/power_supply/BAT0/power_now")
	if err != nil {
		return "??"
	}

	hours := float64(energy_now) / float64(power_now)
	hours_int := int64(hours)
	minutes := int64((hours - float64(hours_int)) * 60)

	text := fmt.Sprintf("%s %d:%02d %d%%", symbol[percentage/20], hours_int, minutes, percentage)

	return MakeBlock(text, percentage < 10)
}

func main() {
	PrintHeader()

	ch := make(chan uint64)
	go StartClock(ch, 1, 0)

	for {
		select {
		case <-ch:
			fmt.Println("[" +
				GetBatteryStatusBlock() + "," +
				GetCurrentTimeBlock("15:04") +
				"],")
		}
	}
}
