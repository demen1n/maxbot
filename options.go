package maxbot

// SendOptions represents message sending options.
type SendOptions struct {
	Text        string
	Format      string
	Attachments []Attachment
}

// CallbackResponse represents a response to callback query.
type CallbackResponse struct {
	Text      string `json:"text,omitempty"`
	ShowAlert bool   `json:"show_alert,omitempty"`
	URL       string `json:"url,omitempty"`
}

// Attachment represents a message attachment (keyboard, file, etc).
type Attachment struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// SendMessage represents an outgoing message request.
type SendMessage struct {
	ChatID      string       `json:"chat_id"`
	Text        string       `json:"text"`
	Format      string       `json:"format,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// EditMessage represents a message edit request.
type EditMessage struct {
	MessageID int    `json:"message_id"`
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
}

func parseSendOptions(opts []interface{}) *SendOptions {
	for _, opt := range opts {
		if o, ok := opt.(*SendOptions); ok {
			return o
		}
	}
	return &SendOptions{}
}
