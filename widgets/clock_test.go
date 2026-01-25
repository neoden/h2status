package widgets

import (
	"testing"
)

func TestNewClock_DefaultFormat(t *testing.T) {
	c := NewClock(nil)

	if len(c.formats) != 1 {
		t.Fatalf("formats length = %d, want 1", len(c.formats))
	}
	if c.formats[0] != "%H:%M" {
		t.Errorf("default format = %q, want %%H:%%M", c.formats[0])
	}
}

func TestNewClock_EmptyFormats(t *testing.T) {
	c := NewClock([]string{})

	if len(c.formats) != 1 {
		t.Fatalf("formats length = %d, want 1", len(c.formats))
	}
	if c.formats[0] != "%H:%M" {
		t.Errorf("default format = %q, want %%H:%%M", c.formats[0])
	}
}

func TestNewClock_CustomFormats(t *testing.T) {
	formats := []string{"%H:%M", "%Y-%m-%d", "%A"}
	c := NewClock(formats)

	if len(c.formats) != 3 {
		t.Fatalf("formats length = %d, want 3", len(c.formats))
	}
	if c.formats[0] != "%H:%M" {
		t.Errorf("formats[0] = %q, want %%H:%%M", c.formats[0])
	}
	if c.index != 0 {
		t.Errorf("initial index = %d, want 0", c.index)
	}
}

func TestClock_Update(t *testing.T) {
	c := NewClock(nil)
	c.Update()
}

func TestClock_GetBlock(t *testing.T) {
	c := NewClock([]string{"%H:%M"})
	block := c.GetBlock()

	if block == "" {
		t.Error("GetBlock() = empty, want non-empty")
	}
	if !contains(block, "clock") {
		t.Errorf("GetBlock() should contain 'clock' name: %s", block)
	}
	// Should contain time in HH:MM format (we can't predict exact value)
	if !contains(block, ":") {
		t.Errorf("GetBlock() should contain time with colon: %s", block)
	}
}

func TestClock_HandleClick(t *testing.T) {
	c := NewClock([]string{"%H:%M", "%Y-%m-%d", "%A"})

	if c.index != 0 {
		t.Errorf("initial index = %d, want 0", c.index)
	}

	c.HandleClick(1)
	if c.index != 1 {
		t.Errorf("after first click index = %d, want 1", c.index)
	}

	c.HandleClick(1)
	if c.index != 2 {
		t.Errorf("after second click index = %d, want 2", c.index)
	}

	c.HandleClick(1)
	if c.index != 0 {
		t.Errorf("after third click index = %d, want 0 (wrap around)", c.index)
	}
}

func TestClock_HandleClick_SingleFormat(t *testing.T) {
	c := NewClock([]string{"%H:%M"})

	c.HandleClick(1)
	if c.index != 0 {
		t.Errorf("with single format, index should stay 0, got %d", c.index)
	}
}

func TestClock_GetBlock_DifferentFormats(t *testing.T) {
	c := NewClock([]string{"%H:%M", "%Y"})

	block1 := c.GetBlock()
	c.HandleClick(1)
	block2 := c.GetBlock()

	// Both should be valid blocks, but with different content
	if block1 == "" || block2 == "" {
		t.Error("GetBlock() should return non-empty for both formats")
	}
	// Year format should contain 4-digit year
	if !contains(block2, "202") {
		t.Errorf("Year format should contain '202x': %s", block2)
	}
}
