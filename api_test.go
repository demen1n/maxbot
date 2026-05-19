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

// TASK-11: all requests include v= query param.
func TestRequestsIncludeAPIVersion(t *testing.T) {
	var gotV string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotV = r.URL.Query().Get("v")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]interface{}{
				"body": map[string]interface{}{"mid": "mid.1", "seq": 1, "text": "hi"},
			},
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	b.Send(&Chat{ID: 1}, "hi")
	if gotV != APIVersion {
		t.Errorf("expected v=%s in request, got %q", APIVersion, gotV)
	}
}

// TASK-6: link is a top-level Message field; reply populates ReplyTo.
func TestMessageLinkParsed(t *testing.T) {
	raw := `{
		"timestamp": 1700000000,
		"sender": {"user_id": 1, "name": "Alice"},
		"body": {"mid": "mid.1", "seq": 1, "text": "reply text"},
		"link": {
			"type": "reply",
			"sender": {"user_id": 2, "name": "Bob"},
			"message": {"mid": "mid.0", "seq": 0, "text": "original"}
		}
	}`
	var msg Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if msg.Link == nil {
		t.Fatal("expected Link to be set")
	}
	if msg.ReplyTo == nil {
		t.Fatal("expected ReplyTo to be set")
	}
	if msg.ReplyTo.Text() != "original" {
		t.Errorf("expected ReplyTo text 'original', got %q", msg.ReplyTo.Text())
	}
}

// TASK-4: UploadPhoto uses multipart and parses PhotoTokens response.
func TestUploadPhotoMultipart(t *testing.T) {
	// Two servers: one for GET /uploads (getUploadURL), one for the actual upload.
	var uploadURL string
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct == "" || ct == "application/octet-stream" {
			t.Errorf("expected multipart Content-Type, got %q", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"photos": map[string]interface{}{
				"0": map[string]interface{}{"token": "photo-token-xyz"},
			},
		})
	}))
	defer uploadSrv.Close()
	uploadURL = uploadSrv.URL

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"url": uploadURL})
	}))
	defer apiSrv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: apiSrv.URL, Poller: &LongPoller{}})
	tokens, err := b.UploadPhoto("photo.jpg", []byte("imgdata"))
	if err != nil {
		t.Fatalf("UploadPhoto error: %v", err)
	}
	if tokens.Photos["0"].Token != "photo-token-xyz" {
		t.Errorf("unexpected token: %q", tokens.Photos["0"].Token)
	}
}

// TASK-4: UploadMedia for file type parses UploadedInfo from response body.
func TestUploadMediaFile(t *testing.T) {
	var uploadURL string
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"file_id": 99, "token": "file-token"})
	}))
	defer uploadSrv.Close()
	uploadURL = uploadSrv.URL

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"url": uploadURL})
	}))
	defer apiSrv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: apiSrv.URL, Poller: &LongPoller{}})
	info, err := b.UploadMedia("file", "doc.pdf", []byte("pdfdata"))
	if err != nil {
		t.Fatalf("UploadMedia error: %v", err)
	}
	if info.Token != "file-token" {
		t.Errorf("unexpected token: %q", info.Token)
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
