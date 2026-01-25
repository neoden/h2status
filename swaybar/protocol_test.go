package swaybar

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestMakeBlock(t *testing.T) {
	tests := []struct {
		name     string
		blockName string
		fullText string
		urgent   bool
	}{
		{"simple block", "cpu", "CPU 50%", false},
		{"urgent block", "battery", "BAT 5%", true},
		{"empty text", "test", "", false},
		{"special chars", "disk", "/ 10.5G", false},
		{"unicode", "temp", "🌡 75°", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MakeBlock(tt.blockName, tt.fullText, tt.urgent)

			// Should be valid JSON
			var block Block
			if err := json.Unmarshal([]byte(result), &block); err != nil {
				t.Fatalf("MakeBlock() returned invalid JSON: %v", err)
			}

			if block.Name != tt.blockName {
				t.Errorf("Name = %q, want %q", block.Name, tt.blockName)
			}
			if block.FullText != tt.fullText {
				t.Errorf("FullText = %q, want %q", block.FullText, tt.fullText)
			}
			if block.Urgent != tt.urgent {
				t.Errorf("Urgent = %v, want %v", block.Urgent, tt.urgent)
			}
		})
	}
}

func TestMakeBlock_JSONFormat(t *testing.T) {
	result := MakeBlock("test", "hello", true)

	// Check specific JSON structure
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check required fields present
	if _, ok := data["full_text"]; !ok {
		t.Error("Missing full_text field")
	}
	if _, ok := data["name"]; !ok {
		t.Error("Missing name field")
	}
}

func TestBlock_JSONMarshal(t *testing.T) {
	block := Block{
		FullText: "test text",
		Name:     "test",
		Urgent:   true,
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal back
	var decoded Block
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.FullText != block.FullText {
		t.Errorf("FullText = %q, want %q", decoded.FullText, block.FullText)
	}
	if decoded.Name != block.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, block.Name)
	}
	if decoded.Urgent != block.Urgent {
		t.Errorf("Urgent = %v, want %v", decoded.Urgent, block.Urgent)
	}
}

func TestHeader_JSONMarshal(t *testing.T) {
	header := Header{
		Version:     1,
		ClickEvents: true,
	}

	data, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded["version"].(float64) != 1 {
		t.Errorf("version = %v, want 1", decoded["version"])
	}
	if decoded["click_events"].(bool) != true {
		t.Errorf("click_events = %v, want true", decoded["click_events"])
	}
}

func TestClickEvent_JSONUnmarshal(t *testing.T) {
	jsonStr := `{"name":"cpu","instance":"","button":1,"x":100,"y":50}`

	var event ClickEvent
	if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if event.Name != "cpu" {
		t.Errorf("Name = %q, want cpu", event.Name)
	}
	if event.Button != 1 {
		t.Errorf("Button = %d, want 1", event.Button)
	}
	if event.X != 100 {
		t.Errorf("X = %d, want 100", event.X)
	}
	if event.Y != 50 {
		t.Errorf("Y = %d, want 50", event.Y)
	}
}

func TestBlock_OmitEmpty(t *testing.T) {
	block := Block{
		FullText: "test",
		Name:     "test",
		// All other fields empty/zero
	}

	data, _ := json.Marshal(block)
	str := string(data)

	// These should be omitted
	if contains(str, "short_text") {
		t.Error("short_text should be omitted when empty")
	}
	if contains(str, "color") {
		t.Error("color should be omitted when empty")
	}
	if contains(str, "background") {
		t.Error("background should be omitted when empty")
	}
	// urgent:false should be omitted too
	if contains(str, "urgent") {
		t.Error("urgent should be omitted when false")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSendHeaderTo(t *testing.T) {
	var buf bytes.Buffer
	SendHeaderTo(&buf)
	output := buf.String()

	// Should contain valid JSON header
	if !contains(output, `"version":1`) {
		t.Errorf("output should contain version:1: %s", output)
	}
	if !contains(output, `"click_events":true`) {
		t.Errorf("output should contain click_events:true: %s", output)
	}
	// Should end with opening bracket for array
	if !contains(output, "[") {
		t.Errorf("output should contain '[' for array start: %s", output)
	}
}

func TestSendHeaderTo_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	SendHeaderTo(&buf)

	// First line should be valid JSON
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	var header Header
	if err := json.Unmarshal(lines[0], &header); err != nil {
		t.Fatalf("first line is not valid JSON: %v", err)
	}

	if header.Version != 1 {
		t.Errorf("Version = %d, want 1", header.Version)
	}
	if !header.ClickEvents {
		t.Error("ClickEvents = false, want true")
	}
}
