package maxbot

import (
	"testing"
)

func makeBot(t *testing.T) *Bot {
	t.Helper()
	b, err := NewBot(Settings{Token: "tok", Poller: &LongPoller{}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TASK-10: match routes non-message update types to their handlers.
func TestMatchNonMessageUpdateTypes(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   string
		update     Update
		wantRouted bool
	}{
		{
			name:     "bot_started",
			endpoint: OnBotStarted,
			update:   Update{UpdateType: UpdateBotStarted, ChatID: 1, User: &User{ID: 2}},
			wantRouted: true,
		},
		{
			name:     "bot_added",
			endpoint: OnBotAdded,
			update:   Update{UpdateType: UpdateBotAdded, ChatID: 1, User: &User{ID: 3}},
			wantRouted: true,
		},
		{
			name:     "user_added",
			endpoint: OnUserAdded,
			update:   Update{UpdateType: UpdateUserAdded, ChatID: 1, User: &User{ID: 4}},
			wantRouted: true,
		},
		{
			name:     "chat_title_changed",
			endpoint: OnChatTitleChanged,
			update:   Update{UpdateType: UpdateChatTitleChanged, ChatID: 1, Title: "New Title"},
			wantRouted: true,
		},
		{
			name:     "unregistered type returns nil",
			endpoint: OnBotStarted,
			update:   Update{UpdateType: UpdateBotStopped},
			wantRouted: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := makeBot(t)
			called := false
			b.Handle(tc.endpoint, func(Context) error {
				called = true
				return nil
			})
			h := b.match(tc.update)
			if tc.wantRouted && h == nil {
				t.Errorf("expected handler for %s, got nil", tc.name)
			}
			if !tc.wantRouted && h != nil {
				t.Errorf("expected no handler for %s, got one", tc.name)
			}
			_ = called
		})
	}
}

// TASK-10: Sender() returns User from non-message updates.
func TestSenderFromNonMessageUpdate(t *testing.T) {
	b := makeBot(t)
	u := Update{UpdateType: UpdateBotStarted, User: &User{ID: 42, Name: "Bot"}}
	c := &nativeContext{b: b, update: u}
	sender := c.Sender()
	if sender == nil || sender.ID != 42 {
		t.Errorf("expected sender ID=42, got %+v", sender)
	}
}

// TASK-10: Chat() returns chat from non-message updates with chat_id.
func TestChatFromNonMessageUpdate(t *testing.T) {
	b := makeBot(t)
	u := Update{UpdateType: UpdateBotStarted, ChatID: 99}
	c := &nativeContext{b: b, update: u}
	chat := c.Chat()
	if chat == nil || chat.ID != 99 {
		t.Errorf("expected chat ID=99, got %+v", chat)
	}
}

// TASK-10: message_edited routes to OnMessageEdited before OnMessage.
func TestMessageEditedRoutesToOnMessageEdited(t *testing.T) {
	b := makeBot(t)
	editedCalled := false
	b.Handle(OnMessageEdited, func(Context) error { editedCalled = true; return nil })
	b.Handle(OnMessage, func(Context) error { return nil })

	u := Update{
		UpdateType: UpdateMessageEdited,
		Message:    &Message{Body: &MessageBody{Mid: "mid.1", Text: "edited"}},
	}
	h := b.match(u)
	if h == nil {
		t.Fatal("expected handler")
	}
	_ = editedCalled
}

// message_chat_created (fired when a Chat button creates a chat) routes to
// OnMessageChatCreated, and Context.Chat() resolves the created chat.
func TestMessageChatCreatedRouting(t *testing.T) {
	b := makeBot(t)
	called := false
	b.Handle(OnMessageChatCreated, func(Context) error { called = true; return nil })

	u := Update{
		UpdateType:   UpdateMessageChatCreated,
		MessageID:    "mid.1",
		StartPayload: "ref-42",
		Chat:         &Chat{ID: 777, Title: "New chat"},
	}
	h := b.match(u)
	if h == nil {
		t.Fatal("expected handler for message_chat_created")
	}
	if err := h(&nativeContext{b: b, update: u}); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !called {
		t.Error("expected handler to be invoked")
	}

	c := &nativeContext{b: b, update: u}
	chat := c.Chat()
	if chat == nil || chat.ID != 777 {
		t.Errorf("expected chat ID=777, got %+v", chat)
	}
}
