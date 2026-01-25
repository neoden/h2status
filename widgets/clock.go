package widgets

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"neoden/h2status/swaybar"

	"golang.org/x/sys/unix"
)

type Clock struct {
	format string
}

func NewClock(format string) *Clock {
	return &Clock{format: format}
}

func (c *Clock) Update() {
	// Clock doesn't need periodic updates - it just reads current time
}

func (c *Clock) GetBlock() string {
	dt := time.Now()
	return swaybar.MakeBlock("time", dt.Format(c.format), false)
}

// StartTicker creates a timerfd-based ticker that fires every second
// Returns a channel that receives tick counts
func StartTicker(seconds int64, nanoseconds int64) (<-chan uint64, error) {
	fd, err := unix.TimerfdCreate(unix.CLOCK_REALTIME, 0)
	if err != nil {
		return nil, err
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
		unix.Close(fd)
		return nil, err
	}

	ch := make(chan uint64)

	go func() {
		file := os.NewFile(uintptr(fd), "timerfd")
		defer file.Close()
		defer close(ch)

		buffer := make([]byte, 8)
		for {
			_, err := file.Read(buffer)
			if err != nil {
				if err != io.EOF {
					fmt.Fprintln(os.Stderr, "timerfd:", err)
				}
				break
			}
			ch <- binary.BigEndian.Uint64(buffer)
		}
	}()

	return ch, nil
}
