package maxbot

// SendOptions represents message sending options.
type SendOptions struct {
	Text        string
	Format      string
	Attachments []Attachment
	ReplyToMid  string // mid of message to reply to

	// Notify controls whether chat members get a push notification for this
	// message. nil uses the server default (true); for channels the API
	// requires true (or the field omitted) -- posts are always notified.
	Notify *bool

	// DisableLinkPreview suppresses link preview generation for the message
	// text. Only honored by Bot.Send (POST /messages) and Context.Respond
	// (POST /answers); PUT /messages (Bot.Edit) has no such query parameter.
	DisableLinkPreview bool
}

// CallbackResponse represents a response to a callback query.
// Text is the notification toast shown to the user.
type CallbackResponse struct {
	Text string
}

// Attachment represents a message attachment (keyboard, file, etc).
// Latitude/Longitude are only used by the "location" attachment type, and
// Buttons/Direct/DirectUserID only by "reply_keyboard" — per spec both carry
// their fields at the top level of the attachment rather than under Payload.
type Attachment struct {
	Type         string                 `json:"type"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	Latitude     *float64               `json:"latitude,omitempty"`
	Longitude    *float64               `json:"longitude,omitempty"`
	Buttons      [][]ReplyButton        `json:"buttons,omitempty"`
	Direct       bool                   `json:"direct,omitempty"`
	DirectUserID int64                  `json:"direct_user_id,omitempty"`
}

// SendMessage represents an outgoing message request.
// Exactly one of UserID or ChatID must be set.
type SendMessage struct {
	UserID             string       // recipient user ID (private chats)
	ChatID             string       // recipient chat/channel ID
	Text               string       `json:"text"`
	Format             string       `json:"format,omitempty"`
	Attachments        []Attachment `json:"attachments,omitempty"`
	Link               *linkedRef   `json:"link,omitempty"`
	Notify             *bool        `json:"notify,omitempty"`
	DisableLinkPreview bool         // sent as a query param, not a body field
}

// linkedRef is used to attach a reply/forward link to an outgoing message.
type linkedRef struct {
	Type string `json:"type"`
	Mid  string `json:"mid"`
}

// EditMessage represents a message edit request.
type EditMessage struct {
	MessageID   string       `json:"message_id"` // MAX message mid
	ChatID      int64        `json:"chat_id"`
	Text        string       `json:"text"`
	Format      string       `json:"format,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Link        *linkedRef   `json:"link,omitempty"`
	Notify      *bool        `json:"notify,omitempty"`
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
			if v.Notify != nil {
				o.Notify = v.Notify
			}
			o.DisableLinkPreview = v.DisableLinkPreview
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
				o.Attachments = append(o.Attachments, Attachment{
					Type:         "reply_keyboard",
					Buttons:      v.Buttons,
					Direct:       v.Direct,
					DirectUserID: v.DirectUserID,
				})
			}
		}
	}
	return o
}
