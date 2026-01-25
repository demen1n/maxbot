package maxbot

import "encoding/json"

// ReplyMarkup represents inline keyboard markup.
type ReplyMarkup struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard,omitempty"`
}

// InlineButton represents an inline keyboard button.
type InlineButton struct {
	Text    string `json:"text"`
	Type    string `json:"type"`
	Payload string `json:"payload,omitempty"`
	URL     string `json:"url,omitempty"`
	Data    string `json:"-"` // используется для роутинга
}

// Row adds a row of buttons to the keyboard.
func (r *ReplyMarkup) Row(buttons ...InlineButton) {
	r.InlineKeyboard = append(r.InlineKeyboard, buttons)
}

// Data creates a callback button.
// If payload is provided as structured data, it will be marshaled to JSON.
func (r *ReplyMarkup) Data(text, data string, payload ...interface{}) InlineButton {
	btn := InlineButton{
		Text:    text,
		Type:    "callback",
		Data:    data,
		Payload: data,
	}
	if len(payload) > 0 {
		if p, err := json.Marshal(payload[0]); err == nil {
			btn.Payload = string(p)
		}
	}
	return btn
}

// URL creates a link button.
func (r *ReplyMarkup) URL(text, url string) InlineButton {
	return InlineButton{
		Text: text,
		Type: "link",
		URL:  url,
	}
}
