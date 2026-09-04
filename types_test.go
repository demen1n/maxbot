package maxbot

import "testing"

func TestMessageFrom(t *testing.T) {
	sender := &User{ID: 1, Name: "Alice"}
	msg := &Message{Sender: sender}
	if msg.From() != sender {
		t.Error("expected From() to return Sender")
	}
}

func TestMessageChat(t *testing.T) {
	if (&Message{}).Chat() != nil {
		t.Error("expected nil Chat() when RecipientInfo is nil")
	}

	msg := &Message{RecipientInfo: &RecipientInfo{ChatID: 42, ChatType: "dialog"}}
	chat := msg.Chat()
	if chat == nil || chat.ID != 42 || chat.Type != "dialog" {
		t.Errorf("unexpected Chat(): %+v", chat)
	}
}

func TestMessageMessageSig(t *testing.T) {
	id, chatID := (&Message{}).MessageSig()
	if id != "" || chatID != 0 {
		t.Errorf("expected zero values with no RecipientInfo/Body, got (%q, %d)", id, chatID)
	}

	msg := &Message{
		RecipientInfo: &RecipientInfo{ChatID: 7},
		Body:          &MessageBody{Mid: "mid.1"},
	}
	id, chatID = msg.MessageSig()
	if id != "mid.1" || chatID != 7 {
		t.Errorf("expected (mid.1, 7), got (%q, %d)", id, chatID)
	}
}

func TestStoredMessageMessageSig(t *testing.T) {
	sm := &StoredMessage{MessageID: "mid.5", ChatID: 42}
	id, chatID := sm.MessageSig()
	if id != "mid.5" || chatID != 42 {
		t.Errorf("expected (mid.5, 42), got (%s, %d)", id, chatID)
	}
}
