package widgets

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0B"},
		{1, "1B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1024 * 1024, "1.0M"},
		{1024 * 1024 * 1024, "1.0G"},
		{1024 * 1024 * 1024 * 1024, "1.0T"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.0P"},
		{1024*1024*1024*1024*1024*1024 - 1, "1024.0P"}, // just under 1E
		{1536 * 1024, "1.5M"},
		{1536 * 1024 * 1024, "1.5G"},
		{2048 * 1024 * 1024, "2.0G"},
		{500 * 1024, "500.0K"},
		{7 * 1024 * 1024 * 1024, "7.0G"},     // typical RAM
		{16 * 1024 * 1024 * 1024, "16.0G"},   // typical RAM
		{256 * 1024 * 1024 * 1024, "256.0G"}, // typical SSD
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"zero", 0, "0:00"},
		{"one minute", time.Minute, "0:01"},
		{"thirty minutes", 30 * time.Minute, "0:30"},
		{"one hour", time.Hour, "1:00"},
		{"one hour thirty", time.Hour + 30*time.Minute, "1:30"},
		{"two hours", 2 * time.Hour, "2:00"},
		{"five hours fifteen", 5*time.Hour + 15*time.Minute, "5:15"},
		{"ten hours", 10 * time.Hour, "10:00"},
		{"24 hours", 24 * time.Hour, "24:00"},
		{"100 hours", 100 * time.Hour, "100:00"},

		// rounding tests (rounds to nearest minute)
		{"29 seconds rounds down", 29 * time.Second, "0:00"},
		{"30 seconds rounds up", 30 * time.Second, "0:01"},
		{"1h 30s rounds to 1h", time.Hour + 30*time.Second, "1:01"},
		{"59m 29s rounds to 59m", 59*time.Minute + 29*time.Second, "0:59"},
		{"59m 30s rounds to 1h", 59*time.Minute + 30*time.Second, "1:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReadInt(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		expected    int
		expectError bool
	}{
		{"simple number", "42", 42, false},
		{"with newline", "123\n", 123, false},
		{"with whitespace", "  456  \n", 456, false},
		{"zero", "0", 0, false},
		{"negative", "-100", -100, false},
		{"large number", "2147483647", 2147483647, false},
		{"empty file", "", 0, false},          // strconv returns 0 for empty
		{"invalid content", "abc", 0, false},  // strconv returns 0 for invalid
		{"mixed content", "123abc", 0, false}, // strconv returns 0 for invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with content
			tmpFile := filepath.Join(tmpDir, tt.name)
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			result, err := ReadInt(tmpFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("ReadInt() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ReadInt() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ReadInt() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestReadInt_FileNotFound(t *testing.T) {
	_, err := ReadInt("/nonexistent/path/to/file")
	if err == nil {
		t.Error("ReadInt() expected error for non-existent file, got nil")
	}
}
