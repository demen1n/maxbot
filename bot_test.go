package maxbot

import "testing"

// A *ReplyKeyboard attachment must survive a later *SendOptions in the same
// Send() call instead of being clobbered by it.
func TestSendKeyboardSurvivesSendOptions(t *testing.T) {
	b, got := captureAttachments(t)
	kb := &ReplyKeyboard{}
	kb.Row(kb.Message("Hi", "hi"))

	if _, err := b.Send(&User{ID: 1}, "hello", kb, &SendOptions{Format: "markdown"}); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "reply_keyboard" {
		t.Fatalf("expected reply_keyboard attachment to survive, got %+v", *got)
	}
}

// A *ReplyKeyboard (or *ReplyMarkup) option must reach the outgoing message
// even when the payload is a Sendable (Photo, Sticker, ...), not just plain text.
func TestSendableHonorsReplyKeyboard(t *testing.T) {
	b, got := captureAttachments(t)
	kb := &ReplyKeyboard{}
	kb.Row(kb.Message("Hi", "hi"))

	if _, err := b.Send(&User{ID: 1}, &Sticker{Code: "smile"}, kb); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 2 {
		t.Fatalf("expected sticker + reply_keyboard attachments, got %+v", *got)
	}
	if (*got)[0].Type != "sticker" || (*got)[1].Type != "reply_keyboard" {
		t.Fatalf("unexpected attachment types: %+v", *got)
	}
}
