package swaybar

import (
	"encoding/json"
	"fmt"
	"os"
)

type Header struct {
	Version     int  `json:"version"`
	ClickEvents bool `json:"click_events,omitempty"`
	ContSignal  int  `json:"cont_signal,omitempty"`
	StopSignal  int  `json:"stop_signal,omitempty"`
}

type Block struct {
	FullText            string `json:"full_text"`
	ShortText           string `json:"short_text,omitempty"`
	Color               string `json:"color,omitempty"`
	Background          string `json:"background,omitempty"`
	Border              string `json:"border,omitempty"`
	BorderTop           int    `json:"border_top,omitempty"`
	BorderBottom        int    `json:"border_bottom,omitempty"`
	BorderLeft          int    `json:"border_left,omitempty"`
	BorderRight         int    `json:"border_right,omitempty"`
	MinWidthInt         int    `json:"min_width,omitempty"`
	MinWidthString      string `json:"min_width,omitempty"`
	Align               string `json:"align,omitempty"`
	Name                string `json:"name,omitempty"`
	Instance            string `json:"instance,omitempty"`
	Urgent              bool   `json:"urgent,omitempty"`
	Separator           bool   `json:"separator,omitempty"`
	SeparatorBlockWidth int    `json:"separator_block_width,omitempty"`
	Markup              string `json:"markup,omitempty"`
}

type ClickEvent struct {
	Name      string `json:"name"`
	Instance  string `json:"instance"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Button    int    `json:"button"`
	Event     int    `json:"event"`
	RelativeX int    `json:"relative_x"`
	RelativeY int    `json:"relative_y"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

func SendHeader() {
	headerStr, _ := json.Marshal(Header{
		Version:     1,
		ClickEvents: true,
	})

	fmt.Println(string(headerStr))
	fmt.Println("[")
}

func MakeBlock(name string, fullText string, urgent bool) string {
	block := Block{
		FullText: fullText,
		Name:     name,
		Urgent:   urgent,
	}
	blockStr, err := json.Marshal(block)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return string(blockStr)
}
