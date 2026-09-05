package maxbot

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CommentMessage represents a comment posted on a channel post.
// It has the same JSON shape as Message (comments never carry attachments
// or forward another message, but otherwise parse identically, including
// ReplyTo auto-population from a "reply"-type link).
type CommentMessage = Message

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
	return b.PostCommentReply(messageID, text, format, "")
}

// PostCommentReply publishes a comment on a channel post as a reply to an
// existing comment (NewCommentBody.link; comments only support link type
// "reply", never "forward"). Pass replyToCommentMid = "" for a plain
// top-level comment, equivalent to PostComment.
func (b *Bot) PostCommentReply(messageID, text, format, replyToCommentMid string) (*CommentMessage, error) {
	path := fmt.Sprintf("/messages/%s/comments", url.PathEscape(messageID))
	payload := commentBody(text, format, replyToCommentMid)

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
	return b.EditCommentReply(messageID, commentID, text, "")
}

// EditCommentReply updates an existing comment's text and its reply target
// (NewCommentBody.link). Pass replyToCommentMid = "" to leave/clear it,
// equivalent to EditComment.
func (b *Bot) EditCommentReply(messageID, commentID, text, replyToCommentMid string) error {
	path := fmt.Sprintf("/messages/%s/comments?comment_id=%s", url.PathEscape(messageID), url.QueryEscape(commentID))
	payload := commentBody(text, "", replyToCommentMid)
	return b.rawSimple("PUT", path, payload)
}

// commentBody builds a NewCommentBody payload (text/format/link) shared by
// PostCommentReply and EditCommentReply.
func commentBody(text, format, replyToCommentMid string) map[string]interface{} {
	payload := map[string]interface{}{"text": text}
	if format != "" {
		payload["format"] = format
	}
	if replyToCommentMid != "" {
		payload["link"] = &linkedRef{Type: "reply", Mid: replyToCommentMid}
	}
	return payload
}

// DeleteComment removes a comment from a channel post.
// The bot must be a channel administrator with read_all_messages and delete permissions.
func (b *Bot) DeleteComment(messageID, commentID string) error {
	path := fmt.Sprintf("/messages/%s/comments?comment_id=%s", url.PathEscape(messageID), url.QueryEscape(commentID))
	return b.rawSimple("DELETE", path, nil)
}
