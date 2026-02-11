package maxbot

import "fmt"

// User represents a MAX user.
type User struct {
	ID             int64  `json:"user_id"`
	Name           string `json:"name"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Username       string `json:"username,omitempty"`
	IsBot          bool   `json:"is_bot"`
	LastActivityAt int64  `json:"last_activity_time"`
}

// Recipient returns user ID as recipient identifier.
func (u *User) Recipient() string {
	return fmt.Sprintf("%d", u.ID)
}

// Chat represents a MAX chat.
type Chat struct {
	ID          int64  `json:"chat_id"`
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// Recipient returns chat ID as recipient identifier.
func (c *Chat) Recipient() string {
	return fmt.Sprintf("%d", c.ID)
}

// Message represents a MAX message.
type Message struct {
	RecipientInfo *RecipientInfo `json:"recipient,omitempty"`
	Sender        *User          `json:"sender,omitempty"`
	Timestamp     int64          `json:"timestamp"`
	Body          *MessageBody   `json:"body,omitempty"`
}

// MessageBody represents message content.
type MessageBody struct {
	Mid         string              `json:"mid"`
	Seq         int64               `json:"seq"`
	Text        string              `json:"text"`
	Attachments []MessageAttachment `json:"attachments,omitempty"`
	Markup      []MarkupElement     `json:"markup,omitempty"`
	// link содержит информацию о цитируемом сообщении (reply/forward).
	// в MAX API это поле называется "link".
	Link *LinkedMessage `json:"link,omitempty"`
}

// MarkupElement представляет элемент форматирования текста (bold, italic и т.д.).
type MarkupElement struct {
	From   int    `json:"from"`
	Length int    `json:"length"`
	Type   string `json:"type"` // "emphasized", "strong", "strikethrough", etc.
}

// LinkedMessage представляет цитируемое или пересланное сообщение.
type LinkedMessage struct {
	// type может быть "reply" или "forward"
	Type    string       `json:"type"`
	Sender  *User        `json:"sender,omitempty"`
	ChatID  int64        `json:"chat_id,omitempty"`
	Message *MessageBody `json:"message,omitempty"`
}

// MessageAttachment представляет вложение в полученном сообщении.
type MessageAttachment struct {
	Type       string                 `json:"type"`
	CallbackID string                 `json:"callback_id,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// RecipientInfo contains message recipient information.
type RecipientInfo struct {
	ChatID   int64  `json:"chat_id"`
	ChatType string `json:"chat_type"`
	UserID   int64  `json:"user_id"`
}

// Text returns message text content.
func (m *Message) Text() string {
	if m.Body != nil {
		return m.Body.Text
	}
	return ""
}

// ReplyTo returns the message this message is a reply to, or nil.
func (m *Message) ReplyTo() *LinkedMessage {
	if m.Body == nil {
		return nil
	}
	if m.Body.Link != nil && m.Body.Link.Type == "reply" {
		return m.Body.Link
	}
	return nil
}

// IsReply reports whether the message is a reply to another message.
func (m *Message) IsReply() bool {
	return m.ReplyTo() != nil
}

// From returns message sender.
func (m *Message) From() *User {
	return m.Sender
}

// Chat converts recipient info to Chat object.
func (m *Message) Chat() *Chat {
	if m.RecipientInfo == nil {
		return nil
	}
	return &Chat{
		ID:   m.RecipientInfo.ChatID,
		Type: m.RecipientInfo.ChatType,
	}
}

// MessageSig returns message signature for compatibility with Editable interface.
// Note: MAX API uses string mid, so message_id is always 0.
func (m *Message) MessageSig() (int, int64) {
	if m.RecipientInfo != nil {
		return 0, m.RecipientInfo.ChatID
	}
	return 0, 0
}

// Mid returns MAX message ID as string.
func (m *Message) Mid() string {
	if m.Body != nil {
		return m.Body.Mid
	}
	return ""
}

// Update represents an incoming update from MAX API.
type Update struct {
	UpdateType    string         `json:"update_type"`
	Timestamp     int64          `json:"timestamp"`
	UserLocale    string         `json:"user_locale,omitempty"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback,omitempty"`
}

// CallbackQuery represents a callback button press.
type CallbackQuery struct {
	CallbackID string `json:"callback_id"`
	Timestamp  int64  `json:"timestamp"`
	User       *User  `json:"user"`
	Payload    string `json:"payload"`
}

// StoredMessage is a lightweight message reference for database storage.
type StoredMessage struct {
	MessageID int   `json:"message_id"`
	ChatID    int64 `json:"chat_id"`
}

func (sm *StoredMessage) MessageSig() (int, int64) {
	return sm.MessageID, sm.ChatID
}

// BotCommand represents a bot command with description.
type BotCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ChatMember represents a chat member with their status.
type ChatMember struct {
	User   *User  `json:"user"`
	Status string `json:"status"`
}

// ChatAction represents a bot action in chat (typing, sending media, etc).
type ChatAction string

const (
	ActionTyping       ChatAction = "typing_on"
	ActionSendingPhoto ChatAction = "sending_photo"
	ActionSendingVideo ChatAction = "sending_video"
	ActionSendingAudio ChatAction = "sending_audio"
	ActionSendingFile  ChatAction = "sending_file"
	ActionMarkSeen     ChatAction = "mark_seen"
)

// WebhookInfo represents webhook subscription information.
type WebhookInfo struct {
	URL         string   `json:"url"`
	UpdateTypes []string `json:"update_types,omitempty"`
	Secret      string   `json:"secret,omitempty"`
}

// UploadInfo represents upload URL information from MAX API.
type UploadInfo struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}
