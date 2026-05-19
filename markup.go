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
// The button type is determined automatically by MarshalJSON based on which fields are set.
type InlineButton struct {
	Text   string `json:"text"`
	Intent Intent `json:"intent,omitempty"`

	// Callback button
	Payload string `json:"payload,omitempty"`

	// Link button
	URL string `json:"url,omitempty"`

	// OpenApp button (type: "open_app")
	WebApp    string `json:"web_app,omitempty"`
	ContactID int64  `json:"contact_id,omitempty"`

	// Geolocation button (type: "request_geo_location")
	Quick bool `json:"quick,omitempty"`

	// Clipboard button (type: "clipboard")
	ClipboardPayload string `json:"clipboard_payload,omitempty"`

	// Chat button (type: "chat")
	ChatTitle        string `json:"chat_title,omitempty"`
	ChatDescription  string `json:"chat_description,omitempty"`
	ChatStartPayload string `json:"start_payload,omitempty"`
	ChatUUID         string `json:"uuid,omitempty"`

	// Internal routing hint; not serialised.
	Data string `json:"-"`

	// Internal type selectors; not serialised.
	Contact  bool `json:"-"`
	Location bool `json:"-"`
	Message  bool `json:"-"` // forces type:"message"
}

// MarshalJSON serialises the button with an auto-computed "type" field.
func (b *InlineButton) MarshalJSON() ([]byte, error) {
	type Alias InlineButton
	var btnType string
	switch {
	case b.WebApp != "":
		btnType = "open_app"
	case b.URL != "":
		btnType = "link"
	case b.Contact:
		btnType = "request_contact"
	case b.Location:
		btnType = "request_geo_location"
	case b.ClipboardPayload != "":
		btnType = "clipboard"
	case b.ChatTitle != "":
		btnType = "chat"
	case b.Message:
		btnType = "message"
	default:
		btnType = "callback"
	}
	return json.Marshal(struct {
		Type string `json:"type"`
		*Alias
	}{Type: btnType, Alias: (*Alias)(b)})
}

// Row adds a row of buttons to the keyboard.
func (r *ReplyMarkup) Row(buttons ...InlineButton) {
	r.InlineKeyboard = append(r.InlineKeyboard, buttons)
}

// Data creates a callback button with the given payload.
func (r *ReplyMarkup) Data(text, data string, payload ...interface{}) InlineButton {
	btn := InlineButton{
		Text:    text,
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
		URL:  url,
	}
}

// Contact creates a button that requests the user's phone number.
func (r *ReplyMarkup) Contact(text string) InlineButton {
	return InlineButton{
		Text:    text,
		Contact: true,
	}
}

// Geolocation creates a button that requests the user's location.
// If quick is true, the location is sent immediately without a confirmation dialog.
func (r *ReplyMarkup) Geolocation(text string, quick bool) InlineButton {
	return InlineButton{
		Text:     text,
		Location: true,
		Quick:    quick,
	}
}

// OpenApp creates a button that opens a MAX mini-app.
// webApp is the app URL/identifier, payload is passed to the app on launch,
// contactID optionally pins the launch to a specific contact.
func (r *ReplyMarkup) OpenApp(text, webApp, payload string, contactID int64) InlineButton {
	btn := InlineButton{
		Text:    text,
		WebApp:  webApp,
		Payload: payload,
	}
	if contactID != 0 {
		btn.ContactID = contactID
	}
	return btn
}

// Clipboard creates a button that copies text to the clipboard when pressed.
func (r *ReplyMarkup) Clipboard(text, payload string) InlineButton {
	return InlineButton{
		Text:             text,
		ClipboardPayload: payload,
	}
}

// Chat creates a button that initiates a new chat creation flow.
func (r *ReplyMarkup) Chat(text, title, description, startPayload string) InlineButton {
	return InlineButton{
		Text:             text,
		ChatTitle:        title,
		ChatDescription:  description,
		ChatStartPayload: startPayload,
	}
}

// MessageBtn creates a template message button.
func (r *ReplyMarkup) MessageBtn(text string) InlineButton {
	return InlineButton{
		Text:    text,
		Message: true,
	}
}
