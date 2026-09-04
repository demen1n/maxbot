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
