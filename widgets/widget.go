package widgets

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Widget is the interface for status bar widgets
type Widget interface {
	Update()
	GetBlock() string
}

// AsyncWidget is for widgets that have async updates (like bluetooth via dbus)
type AsyncWidget interface {
	Widget
	Init() error
	Updates() <-chan struct{}
}

// ClickHandler is for widgets that handle click events
type ClickHandler interface {
	HandleClick(button int)
}

// Logger for widgets
var Log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// Helper functions

func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(b)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%d:%02d", h, m)
}

func ReadInt(file string) (int, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return 0, err
	}
	value, _ := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 32)
	return int(value), nil
}
