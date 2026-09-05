package maxbot

import (
	"encoding/json"
	"fmt"
)

// User represents a MAX user.
type User struct {
	ID             int64  `json:"user_id"`
	Name           string `json:"name"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Username       string `json:"username,omitempty"`
	IsBot          bool   `json:"is_bot"`
	LastActivityAt int64  `json:"last_activity_time"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	FullAvatarURL  string `json:"full_avatar_url,omitempty"`
}

// Recipient returns user ID as recipient identifier.
func (u *User) Recipient() string {
	return fmt.Sprintf("%d", u.ID)
}

// ChatStatus represents the bot's membership state in a chat.
type ChatStatus string

const (
	ChatActive    ChatStatus = "active"
	ChatRemoved   ChatStatus = "removed"
	ChatLeft      ChatStatus = "left"
	ChatClosed    ChatStatus = "closed"
	ChatSuspended ChatStatus = "suspended"
)

// Image holds a URL to an image resource.
type Image struct {
	URL string `json:"url"`
}

// Chat represents a MAX chat.
type Chat struct {
	ID                int64      `json:"chat_id"`
	Type              string     `json:"type"`
	Status            ChatStatus `json:"status,omitempty"`
	Title             string     `json:"title,omitempty"`
	Description       string     `json:"description,omitempty"`
	Icon              *Image     `json:"icon,omitempty"`
	LastEventTime     int64      `json:"last_event_time,omitempty"`
	ParticipantsCount int        `json:"participants_count,omitempty"`
	OwnerID           int64      `json:"owner_id,omitempty"`
	IsPublic          bool       `json:"is_public,omitempty"`
	Link              string     `json:"link,omitempty"`
	MessagesCount     int64      `json:"messages_count,omitempty"`
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
	// Link is the quoted/forwarded message reference at the top-level Message object per spec.
	Link *LinkedMessage `json:"link,omitempty"`

	// ReplyTo is populated automatically from Link when type == "reply".
	ReplyTo *LinkedMessage `json:"-"`
}

// UnmarshalJSON populates ReplyTo from the top-level link field when type is "reply".
func (m *Message) UnmarshalJSON(data []byte) error {
	type plain struct {
		RecipientInfo *RecipientInfo `json:"recipient,omitempty"`
		Sender        *User          `json:"sender,omitempty"`
		Timestamp     int64          `json:"timestamp"`
		Body          *MessageBody   `json:"body,omitempty"`
		Link          *LinkedMessage `json:"link,omitempty"`
	}
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	m.RecipientInfo = p.RecipientInfo
	m.Sender = p.Sender
	m.Timestamp = p.Timestamp
	m.Body = p.Body
	m.Link = p.Link
	if m.Link != nil && m.Link.Type == "reply" {
		m.ReplyTo = m.Link
	}
	return nil
}

// MessageBody represents message content.
type MessageBody struct {
	Mid         string              `json:"mid"`
	Seq         int64               `json:"seq"`
	Text        string              `json:"text"`
	Attachments []MessageAttachment `json:"attachments,omitempty"`
	Markup      []MarkupElement     `json:"markup,omitempty"`
	// ReplyTo is the mid of the message being replied to (used when sending).
	// Do not confuse with Message.ReplyTo which is the full LinkedMessage object.
	ReplyTo string `json:"reply_to,omitempty"`
}

// MarkupElement представляет элемент форматирования текста (bold, italic и т.д.).
type MarkupElement struct {
	From   int    `json:"from"`
	Length int    `json:"length"`
	Type   string `json:"type"` // "emphasized", "strong", "strikethrough", etc.
}

// LinkedMessage представляет цитируемое или пересланное сообщение.
// Type может быть "reply" или "forward".
type LinkedMessage struct {
	Type    string       `json:"type"`
	Sender  *User        `json:"sender,omitempty"`
	ChatID  int64        `json:"chat_id,omitempty"`
	Message *MessageBody `json:"message,omitempty"`
}

// Text возвращает текст цитируемого сообщения.
// Используется как msg.ReplyTo.Text
func (l *LinkedMessage) Text() string {
	if l.Message != nil {
		return l.Message.Text
	}
	return ""
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
func (m *Message) MessageSig() (string, int64) {
	if m.RecipientInfo != nil {
		return m.Mid(), m.RecipientInfo.ChatID
	}
	return m.Mid(), 0
}

// Mid returns MAX message ID as string.
func (m *Message) Mid() string {
	if m.Body != nil {
		return m.Body.Mid
	}
	return ""
}

// Update type constants.
const (
	UpdateMessageCreated   = "message_created"
	UpdateMessageEdited    = "message_edited"
	UpdateMessageRemoved   = "message_removed"
	UpdateMessageCallback  = "message_callback"
	UpdateBotAdded         = "bot_added"
	UpdateBotRemoved       = "bot_removed"
	UpdateBotStarted       = "bot_started"
	UpdateBotStopped       = "bot_stopped"
	UpdateUserAdded        = "user_added"
	UpdateUserRemoved      = "user_removed"
	UpdateChatTitleChanged = "chat_title_changed"
	UpdateDialogRemoved    = "dialog_removed"
	UpdateDialogCleared    = "dialog_cleared"
	UpdateDialogMuted      = "dialog_muted"
	UpdateDialogUnmuted    = "dialog_unmuted"
	UpdateCommentCreated   = "comment_created"
	UpdateCommentEdited    = "comment_edited"
	UpdateCommentRemoved   = "comment_removed"
	// UpdateMessageChatCreated is not documented on dev.max.ru and absent
	// from the discriminator list of update_type in the live spec (0.0.33),
	// but the MessageChatCreatedUpdate schema and ChatButton are still
	// present in components -- kept for chats created via a Chat button.
	UpdateMessageChatCreated = "message_chat_created"
)

// Update represents an incoming update from MAX API.
type Update struct {
	UpdateType    string         `json:"update_type"`
	Timestamp     int64          `json:"timestamp"`
	UserLocale    string         `json:"user_locale,omitempty"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback,omitempty"`

	// Fields for bot_started, bot_added, bot_removed, bot_stopped,
	// user_added, user_removed, chat_title_changed.
	ChatID  int64  `json:"chat_id,omitempty"`
	User    *User  `json:"user,omitempty"`
	Payload string `json:"payload,omitempty"` // bot_started deeplink
	Title   string `json:"title,omitempty"`   // chat_title_changed

	// Fields for message_removed / comment_removed.
	MessageID string `json:"message_id,omitempty"`
	UserID    int64  `json:"user_id,omitempty"`
	// PostID is the channel post a removed comment belonged to
	// (message_removed, comment_removed); empty for a removed chat message.
	PostID string `json:"post_id,omitempty"`

	// Fields for user_added.
	InviterID int64 `json:"inviter_id,omitempty"`

	// AdminID is who removed User from the chat (user_removed); nil if the
	// user left on their own.
	AdminID *int64 `json:"admin_id,omitempty"`

	// Fields for user_added / user_removed.
	IsChannel bool `json:"is_channel,omitempty"`

	// MutedUntil is the Unix ms timestamp until which the dialog is muted
	// (dialog_muted only).
	MutedUntil int64 `json:"muted_until,omitempty"`

	// Fields for message_chat_created (fired when the first user taps a
	// Chat button). MessageID (above) carries the id of the message the
	// button was on.
	Chat         *Chat  `json:"chat,omitempty"`
	StartPayload string `json:"start_payload,omitempty"`
}

// CallbackQuery represents a callback button press. Per the MAX schema this
// object itself carries no message; the originating message (with the
// pressed keyboard) arrives as the top-level "message" field of the
// containing Update, alongside "callback" — see Update.Message.
type CallbackQuery struct {
	CallbackID string `json:"callback_id"`
	Timestamp  int64  `json:"timestamp"`
	User       *User  `json:"user"`
	Payload    string `json:"payload"`
}

// StoredMessage is a lightweight message reference for database storage.
type StoredMessage struct {
	MessageID string `json:"message_id"` // MAX message mid
	ChatID    int64  `json:"chat_id"`
}

func (sm *StoredMessage) MessageSig() (string, int64) {
	return sm.MessageID, sm.ChatID
}

// BotCommand represents a bot command with description.
type BotCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ChatAdminPermission is a named permission that can be granted to a chat admin.
type ChatAdminPermission string

const (
	PermReadAllMessages  ChatAdminPermission = "read_all_messages"
	PermAddRemoveMembers ChatAdminPermission = "add_remove_members"
	PermAddAdmins        ChatAdminPermission = "add_admins"
	PermChangeChatInfo   ChatAdminPermission = "change_chat_info"
	PermPinMessage       ChatAdminPermission = "pin_message"
	PermEditLink         ChatAdminPermission = "edit_link"
	PermWrite            ChatAdminPermission = "write"
	PermEdit             ChatAdminPermission = "edit"
	PermDelete           ChatAdminPermission = "delete"
	// PermCanCall is documented as assigned automatically by MAX and
	// unavailable in channels; granting it explicitly has no effect.
	PermCanCall ChatAdminPermission = "can_call"

	// Legacy permission values: absent from the live ChatAdminPermission
	// enum (0.0.33) and not grantable, but MAX's own docs say a
	// ChatMember.permissions response may still return them for
	// pre-existing admins. Recognize them when reading, never grant them.
	PermPostEditDeleteMessage ChatAdminPermission = "post_edit_delete_message"
	PermEditMessage           ChatAdminPermission = "edit_message"
	PermDeleteMessage         ChatAdminPermission = "delete_message"

	// PermViewStats is not part of the live ChatAdminPermission enum at all
	// (0.0.33); it is documented in prose as an owner-only channel
	// capability that a bot admin can never hold. Kept only so callers who
	// see it in prose/older docs don't hit an undefined identifier; do not
	// grant it via PromoteChatMember.
	PermViewStats ChatAdminPermission = "view_stats"
)

// ChatMember represents a chat participant.
// The MAX API returns user fields flat alongside member-specific fields;
// UnmarshalJSON populates the nested User from those flat fields.
type ChatMember struct {
	User           *User                 `json:"-"`
	IsOwner        bool                  `json:"is_owner"`
	IsAdmin        bool                  `json:"is_admin"`
	JoinTime       int64                 `json:"join_time"`
	LastAccessTime int64                 `json:"last_access_time"`
	Permissions    []ChatAdminPermission `json:"permissions,omitempty"`
	// Alias is the custom role label shown next to the member's name in the
	// chat/channel settings UI; empty if none was set (the client then
	// substitutes "owner"/"admin" on its own).
	Alias string `json:"alias,omitempty"`
}

// UnmarshalJSON reads flat user fields from the API response into the nested User struct.
func (m *ChatMember) UnmarshalJSON(data []byte) error {
	var raw struct {
		// User fields (flat in the API response)
		UserID         int64  `json:"user_id"`
		Name           string `json:"name"`
		FirstName      string `json:"first_name"`
		LastName       string `json:"last_name"`
		Username       string `json:"username"`
		IsBot          bool   `json:"is_bot"`
		LastActivityAt int64  `json:"last_activity_time"`
		AvatarURL      string `json:"avatar_url"`
		FullAvatarURL  string `json:"full_avatar_url"`
		// ChatMember-specific fields
		IsOwner        bool                  `json:"is_owner"`
		IsAdmin        bool                  `json:"is_admin"`
		JoinTime       int64                 `json:"join_time"`
		LastAccessTime int64                 `json:"last_access_time"`
		Permissions    []ChatAdminPermission `json:"permissions"`
		Alias          string                `json:"alias"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.User = &User{
		ID:             raw.UserID,
		Name:           raw.Name,
		FirstName:      raw.FirstName,
		LastName:       raw.LastName,
		Username:       raw.Username,
		IsBot:          raw.IsBot,
		LastActivityAt: raw.LastActivityAt,
		AvatarURL:      raw.AvatarURL,
		FullAvatarURL:  raw.FullAvatarURL,
	}
	m.IsOwner = raw.IsOwner
	m.IsAdmin = raw.IsAdmin
	m.JoinTime = raw.JoinTime
	m.LastAccessTime = raw.LastAccessTime
	m.Permissions = raw.Permissions
	m.Alias = raw.Alias
	return nil
}

// ChatAction represents a bot action in chat (typing, sending media, etc).
type ChatAction string

const (
	ActionTyping       ChatAction = "typing_on"
	ActionSendingPhoto ChatAction = "sending_photo"
	ActionSendingVideo ChatAction = "sending_video"
	ActionSendingAudio ChatAction = "sending_audio"
	ActionSendingFile  ChatAction = "sending_file"

	// ActionMarkSeen is not part of the live SenderAction enum (0.0.33; it
	// only lists typing_on/sending_photo/sending_video/sending_audio/
	// sending_file) -- it existed in the older 0.0.10 schema alongside
	// typing_off, which is also gone.
	//
	// Deprecated: sending it will likely be rejected by the API. Kept only
	// for source compatibility with existing callers.
	ActionMarkSeen ChatAction = "mark_seen"
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

// SimpleQueryResult is the response body for write operations that return only success status.
type SimpleQueryResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// PhotoToken holds the token for a single uploaded photo.
type PhotoToken struct {
	Token string `json:"token"`
}

// PhotoTokens is the response from a photo upload: a map keyed by photo size/index.
type PhotoTokens struct {
	Photos map[string]PhotoToken `json:"photos"`
}

// UploadedInfo is the response from an audio/video/file upload.
type UploadedInfo struct {
	FileID int64  `json:"file_id,omitempty"`
	Token  string `json:"token,omitempty"`
}
