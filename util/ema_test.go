package util

import (
	"math"
	"testing"
)

func TestNewEMA(t *testing.T) {
	tests := []struct {
		period        int
		expectedAlpha float64
	}{
		{1, 1.0},          // α = 2/(1+1) = 1
		{2, 2.0 / 3.0},    // α = 2/(2+1) = 0.667
		{5, 2.0 / 6.0},    // α = 2/(5+1) = 0.333
		{10, 2.0 / 11.0},  // α = 2/(10+1) = 0.182
		{0, 1.0},          // period < 1 should default to 1
		{-1, 1.0},         // period < 1 should default to 1
	}

	for _, tt := range tests {
		ema := NewEMA(tt.period)
		if math.Abs(ema.alpha-tt.expectedAlpha) > 0.001 {
			t.Errorf("NewEMA(%d).alpha = %f, want %f", tt.period, ema.alpha, tt.expectedAlpha)
		}
	}
}

func TestEMA_Update(t *testing.T) {
	ema := NewEMA(1) // α = 1, so EMA = current value

	// With α = 1, EMA should equal current value
	result := ema.Update(10)
	if result != 10 {
		t.Errorf("Update(10) = %f, want 10", result)
	}

	result = ema.Update(20)
	if result != 20 {
		t.Errorf("Update(20) = %f, want 20", result)
	}
}

func TestEMA_Smoothing(t *testing.T) {
	ema := NewEMA(5) // α = 0.333

	// First value sets the baseline
	ema.Update(100)
	if ema.Value() != 100 {
		t.Errorf("First update should set value directly, got %f", ema.Value())
	}

	// Sudden spike should be smoothed
	ema.Update(200)
	if ema.Value() >= 200 || ema.Value() <= 100 {
		t.Errorf("Spike should be smoothed, got %f", ema.Value())
	}

	// Continue updating with high value - should approach it
	for i := 0; i < 20; i++ {
		ema.Update(200)
	}
	if ema.Value() < 195 {
		t.Errorf("After many updates, should approach 200, got %f", ema.Value())
	}
}

func TestEMA_Ready(t *testing.T) {
	ema := NewEMA(5)

	if ema.Ready() {
		t.Error("Should not be ready before any updates")
	}

	ema.Update(10)
	if ema.Ready() {
		t.Error("Should not be ready after 1 update (need 2 for first smoothing)")
	}

	ema.Update(20)
	if !ema.Ready() {
		t.Error("Should be ready after 2 updates (smoothing has occurred)")
	}

	ema.Update(30)
	if !ema.Ready() {
		t.Error("Should still be ready after 3 updates")
	}
}

func TestEMA_Value(t *testing.T) {
	ema := NewEMA(2)

	// Before any updates, value should be 0
	if ema.Value() != 0 {
		t.Errorf("Value before updates should be 0, got %f", ema.Value())
	}

	ema.Update(50)
	if ema.Value() != 50 {
		t.Errorf("Value after first update should be 50, got %f", ema.Value())
	}
}
