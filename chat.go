package maxbot

import (
	"encoding/json"
	"fmt"
)

// GetChats returns all group chats the bot participates in.
func (b *Bot) GetChats() ([]Chat, error) {
	data, err := b.Raw("GET", "/chats", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Chats []Chat `json:"chats"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Chats, nil
}

// GetChat retrieves chat information by ID.
func (b *Bot) GetChat(chatID int64) (*Chat, error) {
	url := fmt.Sprintf("/chats/%d", chatID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var chat Chat
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, err
	}

	return &chat, nil
}

// UpdateChat modifies a group chat (title, description, icon, etc).
// fields is a map of fields to update, e.g. {"title": "New title"}.
func (b *Bot) UpdateChat(chatID int64, fields map[string]interface{}) (*Chat, error) {
	url := fmt.Sprintf("/chats/%d", chatID)
	data, err := b.Raw("PATCH", url, fields)
	if err != nil {
		return nil, err
	}

	var chat Chat
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, err
	}

	return &chat, nil
}

// DeleteChat removes a group chat.
func (b *Bot) DeleteChat(chatID int64) error {
	url := fmt.Sprintf("/chats/%d", chatID)
	_, err := b.Raw("DELETE", url, nil)
	return err
}

// GetChatMemberMe returns the bot's own membership info in the chat.
func (b *Bot) GetChatMemberMe(chatID int64) (*ChatMember, error) {
	url := fmt.Sprintf("/chats/%d/members/me", chatID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var member ChatMember
	if err := json.Unmarshal(data, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

// GetChatMembers returns all members of a chat.
func (b *Bot) GetChatMembers(chatID int64) ([]ChatMember, error) {
	url := fmt.Sprintf("/chats/%d/members", chatID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Members []ChatMember `json:"members"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Members, nil
}

// GetChatMember gets information about a specific chat member.
func (b *Bot) GetChatMember(chatID int64, userID int64) (*ChatMember, error) {
	url := fmt.Sprintf("/chats/%d/members/%d", chatID, userID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var member ChatMember
	if err := json.Unmarshal(data, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

// GetChatAdmins gets the list of chat administrators.
func (b *Bot) GetChatAdmins(chatID int64) ([]ChatMember, error) {
	url := fmt.Sprintf("/chats/%d/members/admins", chatID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Admins []ChatMember `json:"admins"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Admins, nil
}

// PromoteChatMember promotes a user to administrator.
func (b *Bot) PromoteChatMember(chatID int64, userID int64) error {
	url := fmt.Sprintf("/chats/%d/members/admins", chatID)
	payload := map[string]interface{}{
		"user_id": userID,
	}

	_, err := b.Raw("POST", url, payload)
	return err
}

// DemoteChatMember removes administrator rights from a user.
func (b *Bot) DemoteChatMember(chatID int64, userID int64) error {
	url := fmt.Sprintf("/chats/%d/members/admins/%d", chatID, userID)
	_, err := b.Raw("DELETE", url, nil)
	return err
}

// KickChatMember removes a user from the chat.
func (b *Bot) KickChatMember(chatID int64, userID int64) error {
	url := fmt.Sprintf("/chats/%d/members", chatID)
	payload := map[string]interface{}{
		"user_id": userID,
	}

	_, err := b.Raw("DELETE", url, payload)
	return err
}

// InviteChatMembers adds users to the chat.
func (b *Bot) InviteChatMembers(chatID int64, userIDs []int64) error {
	url := fmt.Sprintf("/chats/%d/members", chatID)
	payload := map[string]interface{}{
		"user_ids": userIDs,
	}

	_, err := b.Raw("POST", url, payload)
	return err
}

// LeaveChat makes the bot leave the chat.
func (b *Bot) LeaveChat(chatID int64) error {
	url := fmt.Sprintf("/chats/%d/members/me", chatID)
	_, err := b.Raw("DELETE", url, nil)
	return err
}

// PinMessage pins a message in the chat.
func (b *Bot) PinMessage(chatID int64, messageID string) error {
	url := fmt.Sprintf("/chats/%d/pin", chatID)
	payload := map[string]interface{}{
		"message_id": messageID,
	}

	_, err := b.Raw("PUT", url, payload)
	return err
}

// UnpinMessage unpins the pinned message.
func (b *Bot) UnpinMessage(chatID int64) error {
	url := fmt.Sprintf("/chats/%d/pin", chatID)
	_, err := b.Raw("DELETE", url, nil)
	return err
}

// GetPinnedMessage retrieves the pinned message.
func (b *Bot) GetPinnedMessage(chatID int64) (*Message, error) {
	url := fmt.Sprintf("/chats/%d/pin", chatID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// SendChatAction sends a chat action (typing, sending photo, etc).
func (b *Bot) SendChatAction(chatID int64, action ChatAction) error {
	url := fmt.Sprintf("/chats/%d/actions", chatID)
	payload := map[string]interface{}{
		"action": string(action),
	}

	_, err := b.Raw("POST", url, payload)
	return err
}
