package maxbot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const maxRetries = 4

// sendMessage sends a message via MAX API, retrying on attachment-not-ready errors.
func (b *Bot) sendMessage(msg *SendMessage) (*Message, error) {
	var recipientParam string
	if msg.ChatID != "" {
		recipientParam = "chat_id=" + msg.ChatID
	} else {
		recipientParam = "user_id=" + msg.UserID
	}
	if msg.DisableLinkPreview {
		recipientParam += "&disable_link_preview=true"
	}
	url := fmt.Sprintf("%s/messages?%s", b.URL, recipientParam)

	// NewMessageBody requires text/attachments/link to be present (each is
	// nullable, but the key itself is required) -- always include them.
	body := map[string]interface{}{
		"text":        msg.Text,
		"attachments": msg.Attachments,
		"link":        msg.Link,
	}
	if msg.Format != "" {
		body["format"] = msg.Format
	}
	if msg.Notify != nil {
		body["notify"] = *msg.Notify
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
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
			lastErr = &NetworkError{Op: "sendMessage", Err: err}
			continue
		}

		respData, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			apiErr := parseAPIError(resp.StatusCode, respData)
			if apiErr.IsAttachmentNotReady() {
				lastErr = apiErr
				continue
			}
			return nil, apiErr
		}

		var wrapper struct {
			Message Message `json:"message"`
		}
		if err := json.Unmarshal(respData, &wrapper); err != nil {
			return nil, err
		}
		return &wrapper.Message, nil
	}
	return nil, lastErr
}

// editMessageByMid edits a message using MAX message ID (mid), retrying on attachment-not-ready errors.
// sendOpts carries the aggregated Format/Attachments/ReplyToMid to apply; nil means text-only.
func (b *Bot) editMessageByMid(mid string, what interface{}, sendOpts *SendOptions) error {
	if mid == "" {
		return fmt.Errorf("message mid is empty")
	}

	// NewMessageBody requires attachments/link to be present (nullable);
	// nil means "leave attachments/link unchanged" per the API's edit
	// semantics -- unless the caller explicitly passed attachments/a
	// keyboard/a reply link, which must replace them.
	body := map[string]interface{}{
		"attachments": nil,
		"link":        nil,
	}
	switch v := what.(type) {
	case string:
		body["text"] = v
	default:
		return fmt.Errorf("unsupported editable type: %T", what)
	}
	if sendOpts != nil {
		if sendOpts.Format != "" {
			body["format"] = sendOpts.Format
		}
		if len(sendOpts.Attachments) > 0 {
			body["attachments"] = sendOpts.Attachments
		}
		if sendOpts.ReplyToMid != "" {
			body["link"] = &linkedRef{Type: "reply", Mid: sendOpts.ReplyToMid}
		}
		if sendOpts.Notify != nil {
			body["notify"] = *sendOpts.Notify
		}
	}

	url := fmt.Sprintf("%s/messages?message_id=%s", b.URL, mid)

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}

		data, err := json.Marshal(body)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", b.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := b.Client.Do(req)
		if err != nil {
			lastErr = &NetworkError{Op: "editMessage", Err: err}
			continue
		}

		respData, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			apiErr := parseAPIError(resp.StatusCode, respData)
			if apiErr.IsAttachmentNotReady() {
				lastErr = apiErr
				continue
			}
			return apiErr
		}

		var result SimpleQueryResult
		if err := json.Unmarshal(respData, &result); err != nil {
			return err
		}
		if !result.Success {
			return errors.New(result.Message)
		}
		return nil
	}
	return lastErr
}

// editMessage edits a message via API using StoredMessage's mid.
func (b *Bot) editMessage(edit *EditMessage) error {
	path := "/messages?message_id=" + edit.MessageID
	// NewMessageBody requires attachments/link to be present (nullable);
	// nil means "leave unchanged" unless the caller supplied attachments/a
	// keyboard/a reply link, which must replace them.
	body := map[string]interface{}{
		"text":        edit.Text,
		"attachments": nil,
		"link":        nil,
	}
	if edit.Format != "" {
		body["format"] = edit.Format
	}
	if edit.Notify != nil {
		body["notify"] = *edit.Notify
	}
	if len(edit.Attachments) > 0 {
		body["attachments"] = edit.Attachments
	}
	if edit.Link != nil {
		body["link"] = edit.Link
	}
	return b.rawSimple("PUT", path, body)
}

// deleteMessage deletes a message via API using its string mid.
func (b *Bot) deleteMessage(mid string) error {
	return b.rawSimple("DELETE", "/messages?message_id="+mid, nil)
}

// getUpdates retrieves updates via long polling.
func (b *Bot) getUpdates(marker *int64, limit int, timeout int, types []string) ([]Update, *int64, error) {
	path := fmt.Sprintf("/updates?timeout=%d", timeout)
	if limit > 0 {
		path += fmt.Sprintf("&limit=%d", limit)
	}
	if marker != nil {
		path += fmt.Sprintf("&marker=%d", *marker)
	}
	if len(types) > 0 {
		path += "&types=" + strings.Join(types, ",")
	}
	url := b.URL + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", b.Token)

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, parseAPIError(resp.StatusCode, data)
	}

	var response struct {
		Updates []Update `json:"updates"`
		Marker  *int64   `json:"marker"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal updates: %v", err)
	}

	if len(response.Updates) > 0 {
		b.log("received %d update(s)", len(response.Updates))
	}

	return response.Updates, response.Marker, nil
}

// respondCallback responds to a callback query.
// callback_id is sent as a query parameter per the MAX API spec.
func (b *Bot) respondCallback(callbackID string, resp *CallbackResponse) error {
	body := map[string]interface{}{}
	if resp != nil && resp.Text != "" {
		body["notification"] = resp.Text
	}
	_, err := b.Raw("POST", "/answers?callback_id="+callbackID, body)
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

// BotPatch contains fields to update on the bot via PATCH /me.
// Commands are not part of this: MAX exposes a dedicated PATCH /me/commands
// endpoint for them (see SetCommands).
type BotPatch struct {
	// Name sets the bot's visible name. Deprecated by MAX in favor of
	// FirstName; kept for compatibility with existing callers.
	Name        string `json:"name,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	Description string `json:"description,omitempty"`
	// Commands replaces the bot's command list. Pass an empty (non-nil)
	// slice to remove all commands; prefer SetCommands/DeleteCommands,
	// which use the dedicated PATCH /me/commands endpoint instead.
	Commands []BotCommand `json:"commands,omitempty"`
}

// PatchBot updates bot properties via PATCH /me.
func (b *Bot) PatchBot(patch BotPatch) (*User, error) {
	data, err := b.Raw("PATCH", "/me", patch)
	if err != nil {
		return nil, err
	}
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// SetCommands sets the bot's command list via the dedicated
// PATCH /me/commands endpoint. Pass an empty slice to remove all commands.
func (b *Bot) SetCommands(commands []BotCommand) error {
	if commands == nil {
		commands = []BotCommand{}
	}
	payload := map[string]interface{}{
		"commands": commands,
	}
	// PATCH /me/commands returns BotCommandsInfo (the resulting command
	// list), not a SimpleQueryResult -- failures surface as non-200 status
	// codes, which Raw already turns into an error.
	_, err := b.Raw("PATCH", "/me/commands", payload)
	return err
}

// DeleteCommands removes all bot commands.
func (b *Bot) DeleteCommands() error {
	return b.SetCommands([]BotCommand{})
}

// GetUploadURL gets a URL for uploading files.
// fileType can be: "image", "video", "audio", "file"
func (b *Bot) GetUploadURL(fileType string) (*UploadInfo, error) {
	data, err := b.Raw("POST", "/uploads?type="+fileType, nil)
	if err != nil {
		return nil, err
	}

	var info UploadInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// UploadPhoto uploads an image file via multipart/form-data.
// Returns PhotoTokens containing the uploaded photo tokens.
func (b *Bot) UploadPhoto(fileName string, data []byte) (*PhotoTokens, error) {
	info, err := b.GetUploadURL("image")
	if err != nil {
		return nil, err
	}

	body, contentType, err := buildMultipart(fileName, data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", info.URL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, &NetworkError{Op: "UploadPhoto", Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed: %d - %s", resp.StatusCode, string(raw))
	}

	var result PhotoTokens
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UploadMedia uploads an audio, video or file via multipart/form-data.
// fileType must be one of: "audio", "video", "file".
// For audio/video the token comes from the upload URL response (info.Token).
// For file the token comes from the upload response body.
func (b *Bot) UploadMedia(fileType, fileName string, data []byte) (*UploadedInfo, error) {
	info, err := b.GetUploadURL(fileType)
	if err != nil {
		return nil, err
	}

	body, contentType, err := buildMultipart(fileName, data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", info.URL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, &NetworkError{Op: "UploadMedia", Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed: %d - %s", resp.StatusCode, string(raw))
	}

	if fileType == "audio" || fileType == "video" {
		// Token provided by the upload URL endpoint, response body is irrelevant.
		return &UploadedInfo{Token: info.Token}, nil
	}

	var result UploadedInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UploadFile is a compatibility wrapper around UploadPhoto/UploadMedia.
// Deprecated: use UploadPhoto for images and UploadMedia for other types.
func (b *Bot) UploadFile(fileType string, fileName string, fileData []byte) (string, error) {
	if fileType == "image" {
		tokens, err := b.UploadPhoto(fileName, fileData)
		if err != nil {
			return "", err
		}
		for _, t := range tokens.Photos {
			return t.Token, nil
		}
		return "", nil
	}
	info, err := b.UploadMedia(fileType, fileName, fileData)
	if err != nil {
		return "", err
	}
	return info.Token, nil
}

// buildMultipart creates a multipart/form-data body with a single "data" field.
func buildMultipart(fileName string, data []byte) (*bytes.Buffer, string, error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	part, err := w.CreateFormFile("data", fileName)
	if err != nil {
		return nil, "", err
	}
	if _, err = part.Write(data); err != nil {
		return nil, "", err
	}
	if err = w.Close(); err != nil {
		return nil, "", err
	}
	return buf, w.FormDataContentType(), nil
}

// GetMessages retrieves messages in a chat.
// from/to are optional timestamp boundaries (pass 0 to omit); count limits results.
// The returned marker may always be nil: MAX's own MessageList response
// schema declares only "messages", even though the endpoint's own
// description mentions marker-based pagination -- this passes it through
// if the server does send one, without assuming it will.
func (b *Bot) GetMessages(chatID int64, count int, from, to int64) ([]Message, *int64, error) {
	path := fmt.Sprintf("/messages?chat_id=%d", chatID)
	if count > 0 {
		path += fmt.Sprintf("&count=%d", count)
	}
	if from > 0 {
		path += fmt.Sprintf("&from=%d", from)
	}
	if to > 0 {
		path += fmt.Sprintf("&to=%d", to)
	}

	data, err := b.Raw("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Messages []Message `json:"messages"`
		Marker   *int64    `json:"marker"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, err
	}

	return response.Messages, response.Marker, nil
}

// GetMessage retrieves a single message by its mid.
func (b *Bot) GetMessage(mid string) (*Message, error) {
	data, err := b.Raw("GET", "/messages/"+mid, nil)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// GetVideoInfo returns video metadata by its token.
func (b *Bot) GetVideoInfo(videoToken string) (map[string]interface{}, error) {
	data, err := b.Raw("GET", "/videos/"+videoToken, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// rawSimple performs a request whose response body is a SimpleQueryResult
// (or an extension of it, like ModifyMembersResult) and turns a 200 OK
// {"success": false} response into an error -- the MAX API reports failures
// like "not found" or "no permission" this way instead of via HTTP status.
func (b *Bot) rawSimple(method, endpoint string, payload interface{}) error {
	data, err := b.Raw(method, endpoint, payload)
	if err != nil {
		return err
	}
	var result SimpleQueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if !result.Success {
		return errors.New(result.Message)
	}
	return nil
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
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, data)
	}

	return data, nil
}

// parseAPIError parses an error response body into an *APIError.
// MAX API error body: {"error": "short", "code": "dot.separated.key", "message": "human text"}
func parseAPIError(statusCode int, body []byte) *APIError {
	var resp struct {
		Error   string `json:"error"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	apiErr := &APIError{Code: statusCode}
	if json.Unmarshal(body, &resp) == nil && (resp.Code != "" || resp.Error != "") {
		apiErr.ErrorText = resp.Error
		apiErr.Message = resp.Code
		apiErr.Details = resp.Message
	} else {
		apiErr.ErrorText = string(body)
	}
	return apiErr
}

// IsAPIError reports whether err is an *APIError and optionally checks HTTP status codes.
func IsAPIError(err error, codes ...int) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if len(codes) == 0 {
		return true
	}
	for _, c := range codes {
		if apiErr.Code == c {
			return true
		}
	}
	return false
}
