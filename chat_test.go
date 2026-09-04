package maxbot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TASK-9: GetChatAdmins reads members field, not admins.
func TestGetChatAdminsReadsMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"members": []map[string]interface{}{
				{"user_id": 1, "name": "Alice", "is_owner": true},
			},
			"marker": nil,
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	members, _, err := b.GetChatAdmins(1)
	if err != nil {
		t.Fatalf("GetChatAdmins error: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].User == nil || members[0].User.ID != 1 {
		t.Errorf("expected member user_id=1, got %+v", members[0])
	}
}

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

// GetSpecificChatMembers must send user_ids as a single comma-separated
// query parameter, not a repeated key.
func TestGetSpecificChatMembersCommaSeparated(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"members": []map[string]interface{}{
				{"user_id": 1, "name": "Alice"},
				{"user_id": 2, "name": "Bob"},
			},
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	members, err := b.GetSpecificChatMembers(1, []int64{1, 2})
	if err != nil {
		t.Fatalf("GetSpecificChatMembers error: %v", err)
	}
	if gotQuery != "user_ids=1,2&v="+APIVersion {
		t.Errorf("expected query user_ids=1,2, got %q", gotQuery)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

// GetChatMember has no dedicated path on the API; it must go through the
// user_ids filter instead of a nonexistent /members/{userId} endpoint.
func TestGetChatMemberUsesFilterEndpoint(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"members": []map[string]interface{}{
				{"user_id": 42, "name": "Carl"},
			},
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	member, err := b.GetChatMember(1, 42)
	if err != nil {
		t.Fatalf("GetChatMember error: %v", err)
	}
	if gotPath != "/chats/1/members" {
		t.Errorf("expected path /chats/1/members, got %q", gotPath)
	}
	if gotQuery != "user_ids=42&v="+APIVersion {
		t.Errorf("expected query user_ids=42, got %q", gotQuery)
	}
	if member.User == nil || member.User.ID != 42 {
		t.Errorf("expected user_id=42, got %+v", member)
	}
}

// GetChatMember must return an error when the API returns no matching member.
func TestGetChatMemberNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"members": []map[string]interface{}{}})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if _, err := b.GetChatMember(1, 99); err == nil {
		t.Fatal("expected error for missing member, got nil")
	}
}
