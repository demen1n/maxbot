package maxbot

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// CommentMessage represents a comment posted on a channel post.
// Comments never carry attachments and never forward another message.
type CommentMessage struct {
	Sender    *User          `json:"sender,omitempty"`
	Recipient *RecipientInfo `json:"recipient,omitempty"`
	Timestamp int64          `json:"timestamp"`
	Link      *LinkedMessage `json:"link,omitempty"`
	Body      *MessageBody   `json:"body,omitempty"`
}

// Text returns the comment's text content.
func (c *CommentMessage) Text() string {
	if c.Body != nil {
		return c.Body.Text
	}
	return ""
}

// GetComments retrieves comments left on a channel post.
// The bot must be a channel administrator with the read_all_messages permission.
// commentIDs optionally filters to specific comments (nil/empty = no filter);
// after/before are millisecond timestamps (pass 0 to omit either bound);
// count limits results (0 = server default, max 100).
func (b *Bot) GetComments(messageID string, commentIDs []string, after, before int64, count int) ([]CommentMessage, error) {
	path := fmt.Sprintf("/messages/%s/comments", url.PathEscape(messageID))
	sep := "?"
	if len(commentIDs) > 0 {
		path += sep + "comment_ids=" + strings.Join(commentIDs, ",")
		sep = "&"
	}
	if after > 0 {
		path += fmt.Sprintf("%safter=%d", sep, after)
		sep = "&"
	}
	if before > 0 {
		path += fmt.Sprintf("%sbefore=%d", sep, before)
		sep = "&"
	}
	if count > 0 {
		path += fmt.Sprintf("%scount=%d", sep, count)
	}

	data, err := b.Raw("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Messages []CommentMessage `json:"messages"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

// GetComment retrieves a single comment by its id.
func (b *Bot) GetComment(messageID, commentID string) (*CommentMessage, error) {
	path := fmt.Sprintf("/messages/%s/comments/%s", url.PathEscape(messageID), url.PathEscape(commentID))
	data, err := b.Raw("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var comment CommentMessage
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// PostComment publishes a comment on a channel post.
// The bot must be a channel administrator with read_all_messages and write
// permissions. format is optional ("markdown" or "html"); pass "" for plain text.
func (b *Bot) PostComment(messageID, text, format string) (*CommentMessage, error) {
	path := fmt.Sprintf("/messages/%s/comments", url.PathEscape(messageID))
	payload := map[string]interface{}{"text": text}
	if format != "" {
		payload["format"] = format
	}

	data, err := b.Raw("POST", path, payload)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Message CommentMessage `json:"message"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Message, nil
}

// EditComment updates the text of an existing comment.
func (b *Bot) EditComment(messageID, commentID, text string) error {
	path := fmt.Sprintf("/messages/%s/comments?comment_id=%s", url.PathEscape(messageID), url.QueryEscape(commentID))
	payload := map[string]interface{}{"text": text}

	data, err := b.Raw("PUT", path, payload)
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

// DeleteComment removes a comment from a channel post.
// The bot must be a channel administrator with read_all_messages and delete permissions.
func (b *Bot) DeleteComment(messageID, commentID string) error {
	path := fmt.Sprintf("/messages/%s/comments?comment_id=%s", url.PathEscape(messageID), url.QueryEscape(commentID))

	data, err := b.Raw("DELETE", path, nil)
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
