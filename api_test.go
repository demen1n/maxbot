package maxbot

import (
	"encoding/json"
	"errors"
	"fmt"
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

// SetCommands must use the dedicated PATCH /me/commands endpoint, not PATCH /me.
func TestSetCommandsUsesDedicatedEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"commands": gotBody["commands"]})
	}))

	err := b.SetCommands([]BotCommand{{Name: "start", Description: "Start the bot"}})
	if err != nil {
		t.Fatalf("SetCommands error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/me/commands" {
		t.Errorf("expected path /me/commands, got %q", gotPath)
	}
	cmds, ok := gotBody["commands"].([]interface{})
	if !ok || len(cmds) != 1 {
		t.Fatalf("expected 1 command in body, got %+v", gotBody["commands"])
	}
}

// DeleteCommands must send an empty (not nil/omitted) commands array.
func TestDeleteCommandsSendsEmptyArray(t *testing.T) {
	var gotBody map[string]interface{}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"commands": []interface{}{}})
	}))

	if err := b.DeleteCommands(); err != nil {
		t.Fatalf("DeleteCommands error: %v", err)
	}
	cmds, ok := gotBody["commands"].([]interface{})
	if !ok {
		t.Fatalf("expected commands field to be an array, got %+v", gotBody["commands"])
	}
	if len(cmds) != 0 {
		t.Errorf("expected empty commands array, got %+v", cmds)
	}
}

// Edit on a StoredMessage (not *Message) must go through editMessage,
// using message_id/chat_id instead of mid.
func TestEditStoredMessage(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))

	sm := &StoredMessage{MessageID: 5, ChatID: 42}
	if err := b.Edit(sm, "updated text"); err != nil {
		t.Fatalf("Edit error: %v", err)
	}
	if gotPath != "/messages?message_id=5&v="+APIVersion {
		t.Errorf("expected path /messages?message_id=5, got %q", gotPath)
	}
	if gotBody["text"] != "updated text" {
		t.Errorf("expected text=updated text, got %+v", gotBody)
	}
}

func TestEditStoredMessageUnsupportedType(t *testing.T) {
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	sm := &StoredMessage{MessageID: 5, ChatID: 42}
	if err := b.Edit(sm, 123); err == nil {
		t.Fatal("expected error for unsupported editable payload type, got nil")
	}
}

// Delete on a StoredMessage must go through deleteMessage using its int ID.
func TestDeleteStoredMessage(t *testing.T) {
	var gotPath string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))

	sm := &StoredMessage{MessageID: 7, ChatID: 42}
	if err := b.Delete(sm); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if gotPath != "/messages?message_id=7&v="+APIVersion {
		t.Errorf("expected path /messages?message_id=7, got %q", gotPath)
	}
}

func TestDeleteMessageEmptyMid(t *testing.T) {
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be made when mid is empty")
	}))
	if err := b.Delete(&Message{Body: &MessageBody{}}); err == nil {
		t.Fatal("expected error for empty mid, got nil")
	}
}

func TestGetUpdatesBuildsQueryAndParses(t *testing.T) {
	var gotQuery string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"updates": []map[string]interface{}{
				{"update_type": "message_created", "timestamp": 1},
			},
			"marker": 123,
		})
	}))

	marker := int64(10)
	updates, next, err := b.getUpdates(&marker, 5, 30, []string{"message_created", "bot_started"})
	if err != nil {
		t.Fatalf("getUpdates error: %v", err)
	}
	if gotQuery != fmt.Sprintf("timeout=30&limit=5&marker=10&types[]=message_created&types[]=bot_started&v=%s", APIVersion) {
		t.Errorf("unexpected query: %q", gotQuery)
	}
	if len(updates) != 1 || updates[0].UpdateType != "message_created" {
		t.Fatalf("unexpected updates: %+v", updates)
	}
	if next == nil || *next != 123 {
		t.Errorf("expected marker 123, got %v", next)
	}
}

func TestGetUpdatesAPIError(t *testing.T) {
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "boom", "code": "internal"})
	}))
	if _, _, err := b.getUpdates(nil, 0, 30, nil); err == nil {
		t.Fatal("expected error on non-200 response, got nil")
	}
}

func TestMe(t *testing.T) {
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me" {
			t.Errorf("expected path /me, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"user_id": 1, "name": "Bot"})
	}))
	user, err := b.Me()
	if err != nil {
		t.Fatalf("Me error: %v", err)
	}
	if user.Name != "Bot" {
		t.Errorf("expected name=Bot, got %q", user.Name)
	}
}

func TestPatchBot(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"user_id": 1, "name": "New name"})
	}))
	user, err := b.PatchBot(BotPatch{Name: "New name"})
	if err != nil {
		t.Fatalf("PatchBot error: %v", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/me" {
		t.Errorf("expected PATCH /me, got %s %s", gotMethod, gotPath)
	}
	if gotBody["name"] != "New name" {
		t.Errorf("expected name in body, got %+v", gotBody)
	}
	if user.Name != "New name" {
		t.Errorf("expected name=New name, got %q", user.Name)
	}
}

func TestUploadFileImageDelegatesToUploadPhoto(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"photos": map[string]interface{}{"0": map[string]interface{}{"token": "photo-tok"}},
		})
	}))
	defer uploadSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"url": uploadSrv.URL})
	}))
	defer apiSrv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: apiSrv.URL, Poller: &LongPoller{}})
	token, err := b.UploadFile("image", "photo.jpg", []byte("data"))
	if err != nil {
		t.Fatalf("UploadFile error: %v", err)
	}
	if token != "photo-tok" {
		t.Errorf("expected token=photo-tok, got %q", token)
	}
}

func TestUploadFileNonImageDelegatesToUploadMedia(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"token": "file-tok"})
	}))
	defer uploadSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"url": uploadSrv.URL})
	}))
	defer apiSrv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: apiSrv.URL, Poller: &LongPoller{}})
	token, err := b.UploadFile("file", "doc.pdf", []byte("data"))
	if err != nil {
		t.Fatalf("UploadFile error: %v", err)
	}
	if token != "file-tok" {
		t.Errorf("expected token=file-tok, got %q", token)
	}
}

func TestGetMessagesBuildsQuery(t *testing.T) {
	var gotQuery string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]interface{}{
				{"timestamp": 1, "body": map[string]interface{}{"mid": "mid.1", "text": "hi"}},
			},
		})
	}))
	msgs, _, err := b.GetMessages(42, 10, 100, 200)
	if err != nil {
		t.Fatalf("GetMessages error: %v", err)
	}
	if gotQuery != fmt.Sprintf("chat_id=42&count=10&from=100&to=200&v=%s", APIVersion) {
		t.Errorf("unexpected query: %q", gotQuery)
	}
	if len(msgs) != 1 || msgs[0].Text() != "hi" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
}

func TestGetMessage(t *testing.T) {
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages/mid.1" {
			t.Errorf("expected path /messages/mid.1, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"timestamp": 1,
			"body":      map[string]interface{}{"mid": "mid.1", "text": "hello"},
		})
	}))
	msg, err := b.GetMessage("mid.1")
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}
	if msg.Text() != "hello" {
		t.Errorf("expected text=hello, got %q", msg.Text())
	}
}

func TestGetVideoInfo(t *testing.T) {
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/videos/tok123" {
			t.Errorf("expected path /videos/tok123, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"width": 1920, "height": 1080})
	}))
	info, err := b.GetVideoInfo("tok123")
	if err != nil {
		t.Fatalf("GetVideoInfo error: %v", err)
	}
	if info["width"] != float64(1920) {
		t.Errorf("expected width=1920, got %v", info["width"])
	}
}

func TestParseAPIErrorStructured(t *testing.T) {
	body := []byte(`{"error":"bad_request","code":"invalid.chat_id","message":"chat not found"}`)
	err := parseAPIError(http.StatusBadRequest, body)
	if err.Code != http.StatusBadRequest {
		t.Errorf("expected Code=400, got %d", err.Code)
	}
	if err.ErrorText != "bad_request" || err.Message != "invalid.chat_id" || err.Details != "chat not found" {
		t.Errorf("unexpected parsed error: %+v", err)
	}
}

func TestParseAPIErrorUnstructuredBody(t *testing.T) {
	body := []byte("plain text error")
	err := parseAPIError(http.StatusInternalServerError, body)
	if err.ErrorText != "plain text error" {
		t.Errorf("expected raw body as ErrorText, got %q", err.ErrorText)
	}
}

func TestIsAPIError(t *testing.T) {
	apiErr := &APIError{Code: 404}
	if !IsAPIError(apiErr) {
		t.Error("expected IsAPIError(apiErr) to be true")
	}
	if !IsAPIError(apiErr, 404, 500) {
		t.Error("expected IsAPIError to match code 404")
	}
	if IsAPIError(apiErr, 500) {
		t.Error("expected IsAPIError to not match unrelated code")
	}
	if IsAPIError(errors.New("plain error")) {
		t.Error("expected IsAPIError(non-APIError) to be false")
	}
}
