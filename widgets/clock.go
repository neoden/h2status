package widgets

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"neoden/h2status/config"
	"neoden/h2status/swaybar"

	"github.com/klauspost/lctime"
	"golang.org/x/sys/unix"
)

func init() {
	// Fallback to en_US if current locale is minimal (C, POSIX, etc)
	loc := lctime.GetLocale()
	if loc == "" || loc == "C" || loc == "POSIX" || strings.HasPrefix(loc, "C.") {
		lctime.SetLocale("en_US")
	}
}

type Clock struct {
	formats []string
	index   int
}

func NewClock(cfg config.ClockConfig) *Clock {
	formats := cfg.Formats
	if len(formats) == 0 {
		formats = []string{"%H:%M"}
	}
	return &Clock{formats: formats}
}

func (c *Clock) Update() {
	// Clock doesn't need periodic updates - it just reads current time
}

func (c *Clock) GetBlock() string {
	text := lctime.Strftime(c.formats[c.index], time.Now())
	return swaybar.MakeBlock("clock", text, false)
}

func (c *Clock) ClickName() string {
	return "clock"
}

func (c *Clock) HandleClick(button int) {
	c.index = (c.index + 1) % len(c.formats)
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
			ch <- binary.NativeEndian.Uint64(buffer)
		}
	}()

	return ch, nil
}
