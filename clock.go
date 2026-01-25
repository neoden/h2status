package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
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

func GetCurrentTimeBlock(format string) string {
	dt := time.Now()
	return MakeBlock("time", dt.Format(format), false)
}
