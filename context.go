package maxbot

import (
	"fmt"
	"strings"
)

// Context represents the context of an incoming update.
// It provides convenient methods to access update data and respond to users.
type Context interface {
	Bot() *Bot
	Update() Update
	Message() *Message
	Callback() *CallbackQuery

	Sender() *User
	Chat() *Chat
	Text() string
	Args() []string
	Payload() string

	Send(what interface{}, opts ...interface{}) error
	Reply(what interface{}, opts ...interface{}) error
	Edit(what interface{}, opts ...interface{}) error
	Delete() error
	Respond(opts ...*CallbackResponse) error

	Get(key string) interface{}
	Set(key string, val interface{})
}

type nativeContext struct {
	b      *Bot
	update Update
	store  map[string]interface{}
}

func (c *nativeContext) Bot() *Bot                { return c.b }
func (c *nativeContext) Update() Update           { return c.update }
func (c *nativeContext) Message() *Message        { return c.update.Message }
func (c *nativeContext) Callback() *CallbackQuery { return c.update.CallbackQuery }

// Sender returns the user who sent the update.
func (c *nativeContext) Sender() *User {
	if c.update.CallbackQuery != nil {
		return c.update.CallbackQuery.User
	}
	if c.update.Message != nil {
		return c.update.Message.Sender
	}
	return nil
}

// Chat returns the chat where the update occurred.
// For callbacks, chat information may not be available.
func (c *nativeContext) Chat() *Chat {
	if c.update.Message != nil {
		return c.update.Message.Chat()
	}
	return nil
}

// Text returns the message text.
func (c *nativeContext) Text() string {
	if c.update.Message != nil {
		return c.update.Message.Text()
	}
	return ""
}

// Args returns command arguments as a slice.
// For "/start arg1 arg2", returns ["arg1", "arg2"].
func (c *nativeContext) Args() []string {
	text := c.Text()
	if text == "" || text[0] != '/' {
		return nil
	}

	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return nil
	}

	return parts[1:]
}

// Payload returns everything after the command.
// For "/start hello world", returns "hello world".
func (c *nativeContext) Payload() string {
	text := c.Text()
	if text == "" || text[0] != '/' {
		return text
	}

	idx := strings.Index(text, " ")
	if idx == -1 {
		return ""
	}

	return strings.TrimSpace(text[idx+1:])
}

// Send sends a message to the update sender.
func (c *nativeContext) Send(what interface{}, opts ...interface{}) error {
	sender := c.Sender()
	if sender == nil {
		return fmt.Errorf("sender not found")
	}
	_, err := c.b.Send(sender, what, opts...)
	return err
}

// Reply is an alias for Send.
func (c *nativeContext) Reply(what interface{}, opts ...interface{}) error {
	return c.Send(what, opts...)
}

// Edit edits the message that triggered this update.
func (c *nativeContext) Edit(what interface{}, opts ...interface{}) error {
	msg := c.Message()
	if msg == nil {
		return fmt.Errorf("message not found")
	}
	_, err := c.b.Edit(msg, what, opts...)
	return err
}

// Delete deletes the message that triggered this update.
func (c *nativeContext) Delete() error {
	msg := c.Message()
	if msg == nil {
		return fmt.Errorf("message not found")
	}
	return c.b.Delete(msg)
}

// Respond answers a callback query.
func (c *nativeContext) Respond(opts ...*CallbackResponse) error {
	cb := c.Callback()
	if cb == nil {
		return fmt.Errorf("callback not found")
	}

	resp := &CallbackResponse{}
	if len(opts) > 0 {
		resp = opts[0]
	}

	return c.b.respondCallback(cb.CallbackID, resp)
}

// Get retrieves a value from context storage.
func (c *nativeContext) Get(key string) interface{} {
	if c.store == nil {
		return nil
	}
	return c.store[key]
}

// Set stores a value in context storage.
func (c *nativeContext) Set(key string, val interface{}) {
	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = val
}
