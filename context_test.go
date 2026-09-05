package maxbot

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestContextBasicAccessors(t *testing.T) {
	b := makeBot(t)
	msg := &Message{Body: &MessageBody{Mid: "mid.1", Text: "hi"}}
	u := Update{Message: msg}
	c := &nativeContext{b: b, update: u}

	if c.Bot() != b {
		t.Error("Bot() should return the owning bot")
	}
	if c.Update().Message != msg {
		t.Error("Update() should return the stored update")
	}
	if c.Message() != msg {
		t.Error("Message() should return the update's message")
	}
	if c.Callback() != nil {
		t.Error("Callback() should be nil for a message update")
	}
}

func TestContextCallbackAccessor(t *testing.T) {
	b := makeBot(t)
	cb := &CallbackQuery{CallbackID: "cb1"}
	c := &nativeContext{b: b, update: Update{CallbackQuery: cb}}
	if c.Callback() != cb {
		t.Error("Callback() should return the update's callback query")
	}
	if c.Message() != nil {
		t.Error("Message() should be nil for a callback-only update")
	}
}

func TestContextTextArgsPayload(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantText    string
		wantArgs    []string
		wantPayload string
	}{
		{"plain text", "hello world", "hello world", nil, "hello world"},
		{"bare command", "/start", "/start", nil, ""},
		{"command with args", "/start foo bar", "/start foo bar", []string{"foo", "bar"}, "foo bar"},
		{"empty text", "", "", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := makeBot(t)
			c := &nativeContext{b: b, update: Update{Message: &Message{Body: &MessageBody{Text: tc.text}}}}
			if c.Text() != tc.wantText {
				t.Errorf("Text() = %q, want %q", c.Text(), tc.wantText)
			}
			args := c.Args()
			if len(args) != len(tc.wantArgs) {
				t.Errorf("Args() = %v, want %v", args, tc.wantArgs)
			}
			if c.Payload() != tc.wantPayload {
				t.Errorf("Payload() = %q, want %q", c.Payload(), tc.wantPayload)
			}
		})
	}

	// No message at all.
	c := &nativeContext{b: makeBot(t), update: Update{}}
	if c.Text() != "" {
		t.Errorf("expected empty Text() with no message, got %q", c.Text())
	}
}

func TestContextSendFallsBackToSender(t *testing.T) {
	var gotUserID string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.URL.Query().Get("user_id")
		json.NewEncoder(w).Encode(map[string]interface{}{"message": map[string]interface{}{}})
	}))

	// A callback query carries no chat, only a user -- Send must fall back to it.
	c := &nativeContext{b: b, update: Update{CallbackQuery: &CallbackQuery{CallbackID: "cb1", User: &User{ID: 7}}}}
	if err := c.Send("hi"); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if gotUserID != "7" {
		t.Errorf("expected user_id=7, got %q", gotUserID)
	}
}

func TestContextSendNoRecipient(t *testing.T) {
	c := &nativeContext{b: makeBot(t), update: Update{}}
	if err := c.Send("hi"); err == nil {
		t.Fatal("expected error when neither chat nor sender is available")
	}
}

func TestContextReplySetsReplyToMid(t *testing.T) {
	var gotLink *linkedRef
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Link *linkedRef `json:"link"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		gotLink = body.Link
		json.NewEncoder(w).Encode(map[string]interface{}{"message": map[string]interface{}{}})
	}))

	msg := &Message{
		RecipientInfo: &RecipientInfo{ChatID: 1},
		Body:          &MessageBody{Mid: "mid.5"},
	}
	c := &nativeContext{b: b, update: Update{Message: msg}}
	if err := c.Reply("hi"); err != nil {
		t.Fatalf("Reply error: %v", err)
	}
	if gotLink == nil || gotLink.Type != "reply" || gotLink.Mid != "mid.5" {
		t.Errorf("expected reply link to mid.5, got %+v", gotLink)
	}
}

func TestContextReplyFallsBackToSendWithoutMid(t *testing.T) {
	var gotLink *linkedRef
	sawRequest := false
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		var body struct {
			Link *linkedRef `json:"link"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		gotLink = body.Link
		json.NewEncoder(w).Encode(map[string]interface{}{"message": map[string]interface{}{}})
	}))

	// No message at all -- Reply must behave exactly like Send.
	c := &nativeContext{b: b, update: Update{}}
	if err := c.Reply("hi"); err == nil {
		t.Fatal("expected error: no recipient available")
	}

	// Message present but with no mid -- still no reply link.
	msg := &Message{RecipientInfo: &RecipientInfo{ChatID: 1}, Body: &MessageBody{}}
	c = &nativeContext{b: b, update: Update{Message: msg}}
	if err := c.Reply("hi"); err != nil {
		t.Fatalf("Reply error: %v", err)
	}
	if !sawRequest {
		t.Fatal("expected a request to be sent")
	}
	if gotLink != nil {
		t.Errorf("expected no reply link when mid is empty, got %+v", gotLink)
	}
}

func TestContextEditAndDelete(t *testing.T) {
	var gotPath string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))

	msg := &Message{Body: &MessageBody{Mid: "mid.9"}}
	c := &nativeContext{b: b, update: Update{Message: msg}}

	if err := c.Edit("new text"); err != nil {
		t.Fatalf("Edit error: %v", err)
	}
	if gotPath != "/messages?message_id=mid.9" {
		t.Errorf("expected edit path, got %q", gotPath)
	}

	if err := c.Delete(); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if gotPath != "/messages?message_id=mid.9" {
		t.Errorf("expected delete path, got %q", gotPath)
	}
}

func TestContextEditDeleteNoMessage(t *testing.T) {
	c := &nativeContext{b: makeBot(t), update: Update{}}
	if err := c.Edit("x"); err == nil {
		t.Error("expected error from Edit with no message")
	}
	if err := c.Delete(); err == nil {
		t.Error("expected error from Delete with no message")
	}
}

func TestContextRespond(t *testing.T) {
	var gotCallbackID string
	var gotBody map[string]interface{}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCallbackID = r.URL.Query().Get("callback_id")
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))

	c := &nativeContext{b: b, update: Update{CallbackQuery: &CallbackQuery{CallbackID: "cb42"}}}
	if err := c.Respond(&CallbackResponse{Text: "ok"}); err != nil {
		t.Fatalf("Respond error: %v", err)
	}
	if gotCallbackID != "cb42" {
		t.Errorf("expected callback_id=cb42, got %q", gotCallbackID)
	}
	if gotBody["notification"] != "ok" {
		t.Errorf("expected notification=ok, got %+v", gotBody)
	}

	// No opts -- still succeeds with an empty CallbackResponse.
	if err := c.Respond(); err != nil {
		t.Fatalf("Respond() with no opts error: %v", err)
	}
}

func TestContextRespondNoCallback(t *testing.T) {
	c := &nativeContext{b: makeBot(t), update: Update{}}
	if err := c.Respond(); err == nil {
		t.Fatal("expected error responding without a callback query")
	}
}

func TestContextGetSet(t *testing.T) {
	c := &nativeContext{b: makeBot(t)}
	if c.Get("missing") != nil {
		t.Error("expected nil for missing key on empty store")
	}
	c.Set("key", "value")
	if c.Get("key") != "value" {
		t.Errorf("expected value, got %v", c.Get("key"))
	}
}
