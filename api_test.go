package maxbot

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestBot(t *testing.T, handler http.Handler) *Bot {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	b, err := NewBot(Settings{
		Token:  "test-token",
		URL:    srv.URL,
		Poller: &LongPoller{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TASK-1: sendMessage must unwrap {"message":{...}}.
func TestSendMessageUnwrapsEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]interface{}{
				"body":      map[string]interface{}{"mid": "mid.123", "seq": 1, "text": "hello"},
				"sender":    map[string]interface{}{"user_id": 1, "name": "Alice"},
				"timestamp": 1700000000,
				"recipient": map[string]interface{}{"chat_id": 42, "chat_type": "dialog", "user_id": 0},
			},
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	msg, err := b.Send(&Chat{ID: 42}, "hello")
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if msg.Mid() != "mid.123" {
		t.Errorf("expected mid.123, got %q", msg.Mid())
	}
}

// TASK-1: editMessageByMid must return nil on success:true.
func TestEditMessageByMidSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	msg := &Message{Body: &MessageBody{Mid: "mid.abc"}}
	if err := b.Edit(msg, "new text"); err != nil {
		t.Errorf("Edit error: %v", err)
	}
}

// TASK-3: respondCallback puts callback_id in query and notification in body.
func TestRespondCallbackQueryParam(t *testing.T) {
	var gotCallbackID string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCallbackID = r.URL.Query().Get("callback_id")
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	err := b.respondCallback("cbid-42", &CallbackResponse{Text: "toast!"})
	if err != nil {
		t.Fatalf("respondCallback error: %v", err)
	}
	if gotCallbackID != "cbid-42" {
		t.Errorf("expected callback_id=cbid-42 in query, got %q", gotCallbackID)
	}
	if gotBody["notification"] != "toast!" {
		t.Errorf("expected notification=toast! in body, got %v", gotBody["notification"])
	}
	if _, hasID := gotBody["callback_id"]; hasID {
		t.Error("callback_id must not be in request body")
	}
}

// TASK-1: editMessageByMid must return error on success:false.
func TestEditMessageByMidFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "not allowed"})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	msg := &Message{Body: &MessageBody{Mid: "mid.abc"}}
	err := b.Edit(msg, "new text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "not allowed" {
		t.Errorf("unexpected error message: %v", err)
	}
}
