package maxbot

// SendOptions represents message sending options.
type SendOptions struct {
	Text        string
	Format      string
	Attachments []Attachment
	ReplyToMid  string // mid of message to reply to
}

// CallbackResponse represents a response to a callback query.
// Text is the notification toast shown to the user.
type CallbackResponse struct {
	Text string
}

// Attachment represents a message attachment (keyboard, file, etc).
// Latitude/Longitude are only used by the "location" attachment type,
// which per spec carries them as top-level fields rather than in Payload.
type Attachment struct {
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Latitude  *float64               `json:"latitude,omitempty"`
	Longitude *float64               `json:"longitude,omitempty"`
}

// SendMessage represents an outgoing message request.
// Exactly one of UserID or ChatID must be set.
type SendMessage struct {
	UserID      string       // recipient user ID (private chats)
	ChatID      string       // recipient chat/channel ID
	Text        string       `json:"text"`
	Format      string       `json:"format,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Link        *linkedRef   `json:"link,omitempty"`
}

// linkedRef is used to attach a reply/forward link to an outgoing message.
type linkedRef struct {
	Type string `json:"type"`
	Mid  string `json:"mid"`
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
