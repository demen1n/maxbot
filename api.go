package maxbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// sendMessage sends a message via MAX API.
func (b *Bot) sendMessage(msg *SendMessage) (*Message, error) {
	url := fmt.Sprintf("%s/messages?user_id=%s", b.URL, msg.ChatID)

	body := map[string]interface{}{
		"text": msg.Text,
	}

	if msg.Format != "" {
		body["format"] = msg.Format
	}

	if len(msg.Attachments) > 0 {
		body["attachments"] = msg.Attachments
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", b.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respData))
	}

	var result Message
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// editMessageByMid edits a message using MAX message ID (mid).
func (b *Bot) editMessageByMid(mid string, what interface{}, opts ...interface{}) (*Message, error) {
	if mid == "" {
		return nil, fmt.Errorf("message mid is empty")
	}

	body := map[string]interface{}{}

	switch v := what.(type) {
	case string:
		body["text"] = v
	default:
		return nil, fmt.Errorf("unsupported editable type: %T", what)
	}

	for _, opt := range opts {
		if o, ok := opt.(*SendOptions); ok {
			if o.Format != "" {
				body["format"] = o.Format
			}
		}
	}

	url := fmt.Sprintf("%s/messages?message_id=%s", b.URL, mid)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", b.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respData))
	}

	var result Message
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// editMessage edits a message via API (legacy method).
func (b *Bot) editMessage(edit *EditMessage) (*Message, error) {
	path := fmt.Sprintf("/messages/%d", edit.MessageID)
	data, err := b.Raw("PUT", path, edit)
	if err != nil {
		return nil, err
	}

	var result Message
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// deleteMessage deletes a message via API.
func (b *Bot) deleteMessage(msgID int, chatID int64) error {
	path := fmt.Sprintf("/messages/%d", msgID)
	_, err := b.Raw("DELETE", path, nil)
	return err
}

// getUpdates retrieves updates via long polling.
func (b *Bot) getUpdates(marker *int64, limit int, timeout int) ([]Update, *int64, error) {
	url := fmt.Sprintf("%s/updates?timeout=%d", b.URL, timeout)

	if limit > 0 {
		url += fmt.Sprintf("&limit=%d", limit)
	}

	if marker != nil {
		url += fmt.Sprintf("&marker=%d", *marker)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", b.Token)

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(data))
	}

	var response struct {
		Updates []Update `json:"updates"`
		Marker  *int64   `json:"marker"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal updates: %v", err)
	}

	return response.Updates, response.Marker, nil
}

// respondCallback responds to a callback query.
func (b *Bot) respondCallback(callbackID string, resp *CallbackResponse) error {
	payload := map[string]interface{}{
		"callback_id": callbackID,
	}

	if resp != nil && resp.Text != "" {
		payload["notification"] = resp.Text
	}

	_, err := b.Raw("POST", "/answers", payload)
	return err
}

// Me returns information about the bot.
func (b *Bot) Me() (*User, error) {
	data, err := b.Raw("GET", "/me", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// SetCommands sets the bot's command list.
func (b *Bot) SetCommands(commands []BotCommand) error {
	patch := map[string]interface{}{
		"commands": commands,
	}

	data, err := b.Raw("PATCH", "/me", patch)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("failed to set commands: %s", result.Message)
	}

	return nil
}

// DeleteCommands removes all bot commands.
func (b *Bot) DeleteCommands() error {
	return b.SetCommands([]BotCommand{})
}

// GetUploadURL gets a URL for uploading files.
// fileType can be: "image", "video", "audio", "file"
func (b *Bot) GetUploadURL(fileType string) (*UploadInfo, error) {
	url := fmt.Sprintf("%s/uploads?type=%s", b.URL, fileType)

	data, err := b.Raw("POST", url, nil)
	if err != nil {
		return nil, err
	}

	var info UploadInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// UploadFile uploads a file to MAX servers.
func (b *Bot) UploadFile(fileType string, fileName string, fileData []byte) (string, error) {
	info, err := b.GetUploadURL(fileType)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", info.URL, bytes.NewReader(fileData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := b.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed: %d - %s", resp.StatusCode, string(body))
	}

	if fileType == "image" || fileType == "file" {
		var result struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}
		return result.Token, nil
	}

	return info.Token, nil
}

// Raw makes a raw API request.
func (b *Bot) Raw(method, endpoint string, payload interface{}) ([]byte, error) {
	url := b.URL + endpoint

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", b.Token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(data))
	}

	return data, nil
}
