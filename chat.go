package maxbot

import (
	"encoding/json"
	"fmt"
)

// GetChats returns group chats the bot participates in.
// count limits results (0 = server default); marker is the pagination cursor (*nil = start).
// Returns chats and the next page marker (nil when no more pages).
func (b *Bot) GetChats(count int, marker *int64) ([]Chat, *int64, error) {
	path := "/chats"
	sep := "?"
	if count > 0 {
		path += sep + fmt.Sprintf("count=%d", count)
		sep = "&"
	}
	if marker != nil {
		path += sep + fmt.Sprintf("marker=%d", *marker)
	}
	data, err := b.Raw("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Chats  []Chat `json:"chats"`
		Marker *int64 `json:"marker"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, err
	}

	return response.Chats, response.Marker, nil
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

// GetChatMembers returns members of a chat with optional pagination.
func (b *Bot) GetChatMembers(chatID, count int64, marker *int64) ([]ChatMember, *int64, error) {
	path := fmt.Sprintf("/chats/%d/members", chatID)
	sep := "?"
	if count > 0 {
		path += sep + fmt.Sprintf("count=%d", count)
		sep = "&"
	}
	if marker != nil {
		path += sep + fmt.Sprintf("marker=%d", *marker)
	}
	data, err := b.Raw("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Members []ChatMember `json:"members"`
		Marker  *int64       `json:"marker"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, err
	}

	return response.Members, response.Marker, nil
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
// Returns members and an optional pagination marker.
func (b *Bot) GetChatAdmins(chatID int64) ([]ChatMember, *int64, error) {
	url := fmt.Sprintf("/chats/%d/members/admins", chatID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Members []ChatMember `json:"members"`
		Marker  *int64       `json:"marker"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, err
	}

	return response.Members, response.Marker, nil
}

// PromoteChatMember grants admin rights to a user.
// perms lists the permissions to grant; if empty, all permissions are granted.
func (b *Bot) PromoteChatMember(chatID, userID int64, perms ...ChatAdminPermission) error {
	if len(perms) == 0 {
		perms = []ChatAdminPermission{
			PermReadAllMessages,
			PermAddRemoveMembers,
			PermAddAdmins,
			PermChangeChatInfo,
			PermPinMessage,
			PermWrite,
		}
	}
	endpoint := fmt.Sprintf("/chats/%d/members/admins", chatID)
	payload := map[string]interface{}{
		"admins": []map[string]interface{}{
			{"user_id": userID, "permissions": perms},
		},
	}
	_, err := b.Raw("POST", endpoint, payload)
	return err
}

// DemoteChatMember removes administrator rights from a user.
func (b *Bot) DemoteChatMember(chatID int64, userID int64) error {
	url := fmt.Sprintf("/chats/%d/members/admins/%d", chatID, userID)
	_, err := b.Raw("DELETE", url, nil)
	return err
}

// KickChatMember removes a user from the chat.
// Set block=true to also ban the user from rejoining.
func (b *Bot) KickChatMember(chatID, userID int64, block bool) error {
	endpoint := fmt.Sprintf("/chats/%d/members?user_id=%d", chatID, userID)
	if block {
		endpoint += "&block=true"
	}
	_, err := b.Raw("DELETE", endpoint, nil)
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
