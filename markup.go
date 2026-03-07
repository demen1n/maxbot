package maxbot

import "encoding/json"

// ReplyMarkup represents inline keyboard markup.
type ReplyMarkup struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard,omitempty"`
}

// Intent controls the visual style of a button.
type Intent string

const (
	IntentDefault  Intent = "default"
	IntentPositive Intent = "positive"
	IntentNegative Intent = "negative"
)

// InlineButton represents an inline keyboard button.
type InlineButton struct {
	Text    string `json:"text"`
	Type    string `json:"type"`
	Intent  Intent `json:"intent,omitempty"`
	Payload string `json:"payload,omitempty"`
	URL     string `json:"url,omitempty"`
	// OpenApp fields
	App        string `json:"app,omitempty"`
	AppPayload string `json:"app_payload,omitempty"`
	ContactID  int64  `json:"contact_id,omitempty"`
	// Geolocation fields
	Quick bool `json:"quick,omitempty"`

	Data string `json:"-"` // used for routing only
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

// Contact creates a button that requests the user's phone number.
func (r *ReplyMarkup) Contact(text string) InlineButton {
	return InlineButton{
		Text: text,
		Type: "request_contact",
	}
}

// Geolocation creates a button that requests the user's location.
// If quick is true, the location is sent immediately without a confirmation dialog.
func (r *ReplyMarkup) Geolocation(text string, quick bool) InlineButton {
	return InlineButton{
		Text:  text,
		Type:  "request_geo_location",
		Quick: quick,
	}
}

// OpenApp creates a button that opens a MAX mini-app.
// app is the app identifier, payload is passed to the app on launch,
// contactID optionally pins the launch to a specific contact.
func (r *ReplyMarkup) OpenApp(text, app, payload string, contactID int64) InlineButton {
	btn := InlineButton{
		Text:       text,
		Type:       "open_app",
		App:        app,
		AppPayload: payload,
	}
	if contactID != 0 {
		btn.ContactID = contactID
	}
	return btn
}
