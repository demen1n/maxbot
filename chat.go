package maxbot

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// GetChats returns group chats the bot participates in.
// count limits results (0 = server default); marker is the pagination cursor (*nil = start).
// Returns chats and the next page marker (nil when no more pages).
//
// Deprecated: as of June 2026 MAX no longer supports GET /chats and provides
// no replacement for listing all chats the bot is in. Track chat_id yourself
// from incoming updates (bot_added, bot_started, message_created, etc.) and
// use GetChat for lookups instead.
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

// GetChatByLink retrieves chat information by its public link (e.g. "mygroup").
//
// Deprecated: the live MAX Bot API spec (0.0.33) only declares
// /chats/{chatId} with an integer chatId; a link-based path is
// undocumented and left over from the older 0.0.10 schema.
func (b *Bot) GetChatByLink(link string) (*Chat, error) {
	data, err := b.Raw("GET", "/chats/"+link, nil)
	if err != nil {
		return nil, err
	}
	var chat Chat
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, err
	}
	return &chat, nil
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
//
// Deprecated: the live MAX Bot API spec (0.0.33) only declares get/patch
// operations on /chats/{chatId}; delete is undocumented and may not work.
func (b *Bot) DeleteChat(chatID int64) error {
	url := fmt.Sprintf("/chats/%d", chatID)
	return b.rawSimple("DELETE", url, nil)
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

// GetSpecificChatMembers retrieves info for a specific set of users in the chat.
// Per spec, user_ids is a single comma-separated query parameter, not a repeated key.
func (b *Bot) GetSpecificChatMembers(chatID int64, userIDs []int64) ([]ChatMember, error) {
	ids := make([]string, len(userIDs))
	for i, id := range userIDs {
		ids[i] = strconv.FormatInt(id, 10)
	}
	path := fmt.Sprintf("/chats/%d/members?user_ids=%s", chatID, strings.Join(ids, ","))
	data, err := b.Raw("GET", path, nil)
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
// The MAX API has no dedicated /members/{userId} path; it is fetched via
// the user_ids filter on GET /chats/{id}/members.
func (b *Bot) GetChatMember(chatID int64, userID int64) (*ChatMember, error) {
	members, err := b.GetSpecificChatMembers(chatID, []int64{userID})
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, fmt.Errorf("chat member %d not found in chat %d", userID, chatID)
	}
	return &members[0], nil
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

// defaultAdminPermissions are granted by PromoteChatMember when no
// permissions are explicitly requested. It excludes view_stats (not part of
// the ChatAdminPermission enum -- an owner-only capability the request would
// reject) and can_call (assigned automatically by MAX; explicit grants have
// no effect and it is unavailable in channels).
var defaultAdminPermissions = []ChatAdminPermission{
	PermReadAllMessages,
	PermAddRemoveMembers,
	PermAddAdmins,
	PermChangeChatInfo,
	PermPinMessage,
	PermWrite,
}

// PromoteChatMember grants admin rights to a user.
// perms lists the permissions to grant; if empty, defaultAdminPermissions are granted.
func (b *Bot) PromoteChatMember(chatID, userID int64, perms ...ChatAdminPermission) error {
	return b.PromoteChatMemberWithAlias(chatID, userID, "", perms...)
}

// PromoteChatMemberWithAlias grants admin rights to a user, optionally
// labeling the role with a custom alias shown in the chat UI (ChatAdmin.alias).
// perms lists the permissions to grant; if empty, defaultAdminPermissions are granted.
func (b *Bot) PromoteChatMemberWithAlias(chatID, userID int64, alias string, perms ...ChatAdminPermission) error {
	if len(perms) == 0 {
		perms = defaultAdminPermissions
	}
	admin := map[string]interface{}{"user_id": userID, "permissions": perms}
	if alias != "" {
		admin["alias"] = alias
	}
	endpoint := fmt.Sprintf("/chats/%d/members/admins", chatID)
	payload := map[string]interface{}{
		"admins": []map[string]interface{}{admin},
	}
	return b.rawSimple("POST", endpoint, payload)
}

// DemoteChatMember removes administrator rights from a user.
func (b *Bot) DemoteChatMember(chatID int64, userID int64) error {
	url := fmt.Sprintf("/chats/%d/members/admins/%d", chatID, userID)
	return b.rawSimple("DELETE", url, nil)
}

// KickChatMember removes a user from the chat.
// Set block=true to also ban the user from rejoining.
func (b *Bot) KickChatMember(chatID, userID int64, block bool) error {
	endpoint := fmt.Sprintf("/chats/%d/members?user_id=%d", chatID, userID)
	if block {
		endpoint += "&block=true"
	}
	return b.rawSimple("DELETE", endpoint, nil)
}

// InviteChatMembers adds users to the chat. The result reports success as a
// whole and, if MAX could add only some of userIDs, which ones failed and
// why -- a bare error would look like full success or full failure.
func (b *Bot) InviteChatMembers(chatID int64, userIDs []int64) (*ModifyMembersResult, error) {
	url := fmt.Sprintf("/chats/%d/members", chatID)
	payload := map[string]interface{}{
		"user_ids": userIDs,
	}

	data, err := b.Raw("POST", url, payload)
	if err != nil {
		return nil, err
	}
	var result ModifyMembersResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		return &result, errors.New(result.Message)
	}
	return &result, nil
}

// LeaveChat makes the bot leave the chat.
func (b *Bot) LeaveChat(chatID int64) error {
	url := fmt.Sprintf("/chats/%d/members/me", chatID)
	return b.rawSimple("DELETE", url, nil)
}

// PinMessage pins a message in the chat.
// notify controls whether members are notified; pass nil to use server default.
func (b *Bot) PinMessage(chatID int64, messageID string, notify *bool) error {
	endpoint := fmt.Sprintf("/chats/%d/pin", chatID)
	payload := map[string]interface{}{
		"message_id": messageID,
	}
	if notify != nil {
		payload["notify"] = *notify
	}
	return b.rawSimple("PUT", endpoint, payload)
}

// UnpinMessage unpins the pinned message.
func (b *Bot) UnpinMessage(chatID int64) error {
	url := fmt.Sprintf("/chats/%d/pin", chatID)
	return b.rawSimple("DELETE", url, nil)
}

// GetPinnedMessage retrieves the pinned message.
// Returns (nil, nil) if the chat has no pinned message.
func (b *Bot) GetPinnedMessage(chatID int64) (*Message, error) {
	url := fmt.Sprintf("/chats/%d/pin", chatID)
	data, err := b.Raw("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Message *Message `json:"message"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Message, nil
}

// SendChatAction sends a chat action (typing, sending photo, etc).
func (b *Bot) SendChatAction(chatID int64, action ChatAction) error {
	url := fmt.Sprintf("/chats/%d/actions", chatID)
	payload := map[string]interface{}{
		"action": string(action),
	}

	return b.rawSimple("POST", url, payload)
}
