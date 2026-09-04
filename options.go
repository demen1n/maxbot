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

// buildSendOptions aggregates every recognized option (*SendOptions,
// *ReplyMarkup, *ReplyKeyboard) into a single *SendOptions, converting
// keyboards into attachments the same way Bot.Send does for plain-text
// messages. This lets Sendable payloads (Photo, Sticker, ...) honor
// keyboards passed alongside them.
func buildSendOptions(opts []interface{}) *SendOptions {
	o := &SendOptions{}
	for _, opt := range opts {
		switch v := opt.(type) {
		case *SendOptions:
			o.Text = v.Text
			o.Format = v.Format
			o.Attachments = append(o.Attachments, v.Attachments...)
			if v.ReplyToMid != "" {
				o.ReplyToMid = v.ReplyToMid
			}
		case *ReplyMarkup:
			if len(v.InlineKeyboard) > 0 {
				o.Attachments = append(o.Attachments, Attachment{
					Type: "inline_keyboard",
					Payload: map[string]interface{}{
						"buttons": v.InlineKeyboard,
					},
				})
			}
		case *ReplyKeyboard:
			if len(v.Buttons) > 0 {
				payload := map[string]interface{}{"buttons": v.Buttons}
				if v.Direct {
					payload["direct"] = true
				}
				if v.DirectUserID != 0 {
					payload["direct_user_id"] = v.DirectUserID
				}
				o.Attachments = append(o.Attachments, Attachment{
					Type:    "reply_keyboard",
					Payload: payload,
				})
			}
		}
	}
	return o
}
