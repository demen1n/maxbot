package maxbot

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultAPIURL  = "https://platform-api.max.ru"
	DefaultTimeout = 10 * time.Second
)

// Common endpoint constants for message routing.
const (
	OnMessage  = "\amessage"  // любое входящее сообщение
	OnText     = "\atext"     // текстовое сообщение (без команд)
	OnCallback = "\acallback" // нажатие inline-кнопки
	OnPhoto    = "\aphoto"
	OnVideo    = "\avideo"
	OnAudio    = "\aaudio"
	OnDocument = "\adocument"
)

// Bot represents a MAX bot instance.
type Bot struct {
	Token  string
	URL    string
	Poller Poller
	Client *http.Client
	Logger *log.Logger

	handlers map[string]HandlerFunc
	onError  func(error, Context)
}

// Settings represents bot configuration.
type Settings struct {
	URL     string
	Token   string
	Poller  Poller
	Logger  *log.Logger
	OnError func(error, Context)
}

// NewBot creates a new bot instance with the given settings.
func NewBot(s Settings) (*Bot, error) {
	if s.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	if s.URL == "" {
		s.URL = DefaultAPIURL
	}

	if s.Poller == nil {
		s.Poller = &LongPoller{Timeout: DefaultTimeout}
	}

	return &Bot{
		Token:    s.Token,
		URL:      s.URL,
		Poller:   s.Poller,
		Client:   &http.Client{Timeout: 30 * time.Second},
		Logger:   s.Logger,
		handlers: make(map[string]HandlerFunc),
		onError:  s.OnError,
	}, nil
}

func (b *Bot) log(format string, v ...interface{}) {
	if b.Logger != nil {
		b.Logger.Printf(format, v...)
	}
}

// Start begins the bot polling loop and blocks until stopped.
func (b *Bot) Start() {
	b.log("Bot started")
	updates := make(chan Update)
	stop := make(chan struct{})

	go b.Poller.Poll(b, updates, stop)

	for upd := range updates {
		b.ProcessUpdate(upd)
	}
}

// ProcessUpdate processes a single update by finding and executing the appropriate handler.
func (b *Bot) ProcessUpdate(u Update) {
	c := &nativeContext{
		b:      b,
		update: u,
	}

	handler := b.match(u)
	if handler == nil {
		b.log("no handler for update type=%q", u.UpdateType)
		return
	}

	if err := handler(c); err != nil {
		b.log("handler error: %v", err)
		if b.onError != nil {
			b.onError(err, c)
		}
	}
}

// Handle registers a handler for the specified endpoint.
// Endpoint can be a string (command or endpoint constant) or *InlineButton.
// Middleware is applied in the order provided.
func (b *Bot) Handle(endpoint interface{}, handler HandlerFunc, m ...MiddlewareFunc) {
	var key string

	switch end := endpoint.(type) {
	case string:
		key = end
	case *InlineButton:
		key = end.Data
	default:
		panic(fmt.Sprintf("maxbot: unsupported endpoint type %T", endpoint))
	}

	for i := len(m) - 1; i >= 0; i-- {
		handler = m[i](handler)
	}

	b.handlers[key] = handler
	b.log("Registered handler for: %q", key)
}

// match finds the appropriate handler for an update.
// Priority order: callback > command > media type > OnText > OnMessage
func (b *Bot) match(u Update) HandlerFunc {
	// callback идёт первым, но только если это реальный callback (есть CallbackID)
	if u.CallbackQuery != nil && u.CallbackQuery.CallbackID != "" {
		if handler, ok := b.handlers[u.CallbackQuery.Payload]; ok {
			return handler
		}
		if handler, ok := b.handlers[OnCallback]; ok {
			return handler
		}
		return nil
	}

	if u.Message != nil {
		text := u.Message.Text()

		// команды имеют наивысший приоритет среди сообщений
		if text != "" && text[0] == '/' {
			cmd := text
			if idx := strings.Index(text, " "); idx > 0 {
				cmd = text[:idx]
			}
			if idx := strings.Index(cmd, "@"); idx > 0 {
				cmd = cmd[:idx]
			}
			if handler, ok := b.handlers[cmd]; ok {
				return handler
			}
		}

		// роутинг по типу вложения
		if key, ok := mediaEndpoint(u.Message); ok {
			if handler, ok := b.handlers[key]; ok {
				return handler
			}
		}

		// текстовое сообщение
		if text != "" {
			if handler, ok := b.handlers[OnText]; ok {
				return handler
			}
		}

		// OnMessage — ловит всё что не поймали выше
		if handler, ok := b.handlers[OnMessage]; ok {
			return handler
		}
	}

	return nil
}

// mediaEndpoint возвращает endpoint-константу для сообщения с вложением.
func mediaEndpoint(msg *Message) (string, bool) {
	if msg.Body == nil {
		return "", false
	}
	for _, a := range msg.Body.Attachments {
		switch a.Type {
		case "image":
			return OnPhoto, true
		case "video":
			return OnVideo, true
		case "audio":
			return OnAudio, true
		case "file":
			return OnDocument, true
		}
	}
	return "", false
}

// Send sends a message to the specified recipient.
// Returns the sent message or an error.
func (b *Bot) Send(to Recipient, what interface{}, opts ...interface{}) (*Message, error) {
	msg := newSendMessage(to)

	switch v := what.(type) {
	case string:
		msg.Text = v
	case Sendable:
		return v.Send(b, to, parseSendOptions(opts))
	default:
		return nil, fmt.Errorf("unsupported sendable type: %T", what)
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case *SendOptions:
			msg.Format = o.Format
			msg.Attachments = o.Attachments
		case *ReplyMarkup:
			if len(o.InlineKeyboard) > 0 {
				msg.Attachments = append(msg.Attachments, Attachment{
					Type: "inline_keyboard",
					Payload: map[string]interface{}{
						"buttons": o.InlineKeyboard,
					},
				})
			}
		}
	}

	return b.sendMessage(msg)
}

// Edit edits an existing message.
// For MAX API, uses message mid for editing.
func (b *Bot) Edit(msg Editable, what interface{}, opts ...interface{}) (*Message, error) {
	if m, ok := msg.(*Message); ok {
		mid := m.Mid()
		return b.editMessageByMid(mid, what, opts...)
	}

	msgID, chatID := msg.MessageSig()
	edit := &EditMessage{
		MessageID: msgID,
		ChatID:    chatID,
	}

	switch v := what.(type) {
	case string:
		edit.Text = v
	default:
		return nil, fmt.Errorf("unsupported editable type: %T", what)
	}

	return b.editMessage(edit)
}

// Delete deletes a message.
func (b *Bot) Delete(msg Editable) error {
	if m, ok := msg.(*Message); ok {
		mid := m.Mid()
		if mid == "" {
			return fmt.Errorf("message mid is empty")
		}
		return b.deleteMessage(mid)
	}
	msgID, _ := msg.MessageSig()
	return b.deleteMessage(fmt.Sprintf("%d", msgID))
}

// newSendMessage creates a SendMessage with the correct recipient field set.
func newSendMessage(to Recipient) *SendMessage {
	msg := &SendMessage{}
	switch to.(type) {
	case *Chat:
		msg.ChatID = to.Recipient()
	default:
		msg.UserID = to.Recipient()
	}
	return msg
}

// HandlerFunc represents a handler function for processing updates.
type HandlerFunc func(Context) error

// MiddlewareFunc represents middleware that wraps a handler.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Recipient is any object that can receive messages.
type Recipient interface {
	Recipient() string
}

// Sendable is any object that can send itself (photos, videos, etc).
type Sendable interface {
	Send(*Bot, Recipient, *SendOptions) (*Message, error)
}

// Editable is any object that provides message signature for editing.
type Editable interface {
	MessageSig() (messageID int, chatID int64)
}
