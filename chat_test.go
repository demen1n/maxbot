package maxbot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TASK-7: KickChatMember sends user_id and block as query params, not body.
func TestKickChatMemberQueryParams(t *testing.T) {
	var gotUserID, gotBlock string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.URL.Query().Get("user_id")
		gotBlock = r.URL.Query().Get("block")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.KickChatMember(100, 200, true); err != nil {
		t.Fatalf("KickChatMember error: %v", err)
	}
	if gotUserID != "200" {
		t.Errorf("expected user_id=200, got %q", gotUserID)
	}
	if gotBlock != "true" {
		t.Errorf("expected block=true, got %q", gotBlock)
	}
}

