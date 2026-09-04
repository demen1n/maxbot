package maxbot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

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

// Per the MAX API schema (ReplyKeyboardAttachmentRequest), buttons/direct/
// direct_user_id are top-level attachment fields, not nested under "payload"
// like most other attachment types.
func TestReplyKeyboardWireShapeIsTopLevel(t *testing.T) {
	var gotAttachments []map[string]interface{}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Attachments []map[string]interface{} `json:"attachments"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		gotAttachments = body.Attachments
		json.NewEncoder(w).Encode(map[string]interface{}{"message": map[string]interface{}{}})
	}))

	kb := &ReplyKeyboard{Direct: true, DirectUserID: 42}
	kb.Row(kb.Message("Hi", "hi"))
	if _, err := b.Send(&User{ID: 1}, "hello", kb); err != nil {
		t.Fatalf("Send error: %v", err)
	}

	if len(gotAttachments) != 1 {
		t.Fatalf("expected 1 attachment, got %+v", gotAttachments)
	}
	att := gotAttachments[0]
	if _, hasPayload := att["payload"]; hasPayload {
		t.Errorf("reply_keyboard must not wrap fields in payload, got %+v", att)
	}
	if _, ok := att["buttons"]; !ok {
		t.Errorf("expected top-level buttons field, got %+v", att)
	}
	if att["direct"] != true {
		t.Errorf("expected top-level direct=true, got %+v", att["direct"])
	}
	if att["direct_user_id"] != float64(42) {
		t.Errorf("expected top-level direct_user_id=42, got %+v", att["direct_user_id"])
	}
}

func TestMatchCallbackRouting(t *testing.T) {
	b := makeBot(t)

	payloadCalled, fallbackCalled := false, false
	b.Handle("pay-1", func(Context) error { payloadCalled = true; return nil })
	b.Handle(OnCallback, func(Context) error { fallbackCalled = true; return nil })

	// Exact payload match takes priority over the OnCallback fallback.
	h := b.match(Update{CallbackQuery: &CallbackQuery{CallbackID: "cb1", Payload: "pay-1"}})
	if h == nil {
		t.Fatal("expected handler for known payload")
	}
	if err := h(&nativeContext{b: b}); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !payloadCalled || fallbackCalled {
		t.Errorf("expected payload handler only, got payload=%v fallback=%v", payloadCalled, fallbackCalled)
	}

	// Unknown payload falls back to OnCallback.
	h = b.match(Update{CallbackQuery: &CallbackQuery{CallbackID: "cb2", Payload: "unknown"}})
	if h == nil {
		t.Fatal("expected fallback handler")
	}
	if err := h(&nativeContext{b: b}); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !fallbackCalled {
		t.Error("expected OnCallback fallback to run")
	}

	// A CallbackQuery without a CallbackID isn't a real callback.
	if h := b.match(Update{CallbackQuery: &CallbackQuery{}}); h != nil {
		t.Error("expected nil handler for callback without CallbackID")
	}
}

func TestMatchCommandRouting(t *testing.T) {
	b := makeBot(t)
	called := ""
	b.Handle("/start", func(c Context) error { called = "start:" + c.Text(); return nil })

	tests := []struct {
		name string
		text string
		want bool
	}{
		{"plain command", "/start", true},
		{"command with args", "/start hello world", true},
		{"command with bot mention", "/start@mybot", true},
		{"unregistered command falls through", "/unknown", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called = ""
			u := Update{Message: &Message{Body: &MessageBody{Text: tc.text}}}
			h := b.match(u)
			if !tc.want {
				return
			}
			if h == nil {
				t.Fatal("expected handler")
			}
			if err := h(&nativeContext{b: b, update: u}); err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if called == "" {
				t.Error("expected handler to run")
			}
		})
	}
}

func TestMediaEndpointRouting(t *testing.T) {
	tests := []struct {
		attType string
		want    string
	}{
		{"image", OnPhoto},
		{"video", OnVideo},
		{"audio", OnAudio},
		{"file", OnDocument},
	}
	for _, tc := range tests {
		msg := &Message{Body: &MessageBody{
			Attachments: []MessageAttachment{{Type: tc.attType}},
		}}
		key, ok := mediaEndpoint(msg)
		if !ok || key != tc.want {
			t.Errorf("attachment %s: expected endpoint %q, got %q (ok=%v)", tc.attType, tc.want, key, ok)
		}
	}

	if _, ok := mediaEndpoint(&Message{}); ok {
		t.Error("expected no media endpoint for message with nil body")
	}
	if _, ok := mediaEndpoint(&Message{Body: &MessageBody{}}); ok {
		t.Error("expected no media endpoint for message with no attachments")
	}
}

func TestMatchTextAndFallback(t *testing.T) {
	b := makeBot(t)
	b.Handle(OnText, func(Context) error { return nil })
	b.Handle(OnMessage, func(Context) error { return nil })

	// Text present routes to OnText.
	h := b.match(Update{Message: &Message{Body: &MessageBody{Text: "hi"}}})
	if h == nil {
		t.Error("expected OnText handler")
	}

	// No text falls back to OnMessage.
	h = b.match(Update{Message: &Message{Body: &MessageBody{}}})
	if h == nil {
		t.Error("expected OnMessage fallback handler")
	}
}

func TestMatchNoHandlerReturnsNil(t *testing.T) {
	b := makeBot(t)
	if h := b.match(Update{Message: &Message{Body: &MessageBody{Text: "hi"}}}); h != nil {
		t.Error("expected nil handler when nothing registered")
	}
	if h := b.match(Update{UpdateType: UpdateDialogCleared}); h != nil {
		t.Error("expected nil handler for unregistered update type")
	}
}

func TestProcessUpdateDispatchesAndReportsErrors(t *testing.T) {
	b := makeBot(t)
	var gotErr error
	b.onError = func(err error, c Context) { gotErr = err }

	wantErr := fmt.Errorf("boom")
	b.Handle(OnText, func(Context) error { return wantErr })

	b.ProcessUpdate(Update{Message: &Message{Body: &MessageBody{Text: "hi"}}})
	if gotErr != wantErr {
		t.Errorf("expected onError to receive %v, got %v", wantErr, gotErr)
	}
}

func TestProcessUpdateNoHandlerDoesNotPanic(t *testing.T) {
	b := makeBot(t)
	b.ProcessUpdate(Update{Message: &Message{Body: &MessageBody{Text: "hi"}}})
}

// fakePoller lets TestStartStop exercise Bot.Start/Stop's lifecycle without
// LongPoller making real network calls.
type fakePoller struct {
	started chan struct{}
}

func (p *fakePoller) Poll(b *Bot, updates chan Update, stop chan struct{}) {
	close(p.started)
	<-stop
	close(updates)
}

func TestStartStop(t *testing.T) {
	fp := &fakePoller{started: make(chan struct{})}
	b, err := NewBot(Settings{Token: "tok", Poller: fp})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		b.Start()
		close(done)
	}()

	select {
	case <-fp.started:
	case <-time.After(2 * time.Second):
		t.Fatal("Poll was never started")
	}
	b.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after Stop()")
	}
}

func TestBotDeleteMessage(t *testing.T) {
	var gotPath string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	msg := &Message{Body: &MessageBody{Mid: "mid.9"}}
	if err := b.Delete(msg); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if gotPath != "/messages?message_id=mid.9&v="+APIVersion {
		t.Errorf("expected path /messages?message_id=mid.9, got %q", gotPath)
	}
}
