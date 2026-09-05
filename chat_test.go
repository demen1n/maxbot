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
	if gotQuery != "user_ids=1,2" {
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
	if gotQuery != "user_ids=42" {
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

func TestGetChatsBuildsQueryAndMarker(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"chats":  []map[string]interface{}{{"chat_id": 1, "type": "dialog"}},
			"marker": 42,
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	marker := int64(10)
	chats, next, err := b.GetChats(5, &marker)
	if err != nil {
		t.Fatalf("GetChats error: %v", err)
	}
	if gotQuery != "count=5&marker=10" {
		t.Errorf("unexpected query: %q", gotQuery)
	}
	if len(chats) != 1 || chats[0].ID != 1 {
		t.Errorf("unexpected chats: %+v", chats)
	}
	if next == nil || *next != 42 {
		t.Errorf("expected next marker 42, got %v", next)
	}
}

func TestGetChatByLink(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"chat_id": 7, "type": "channel"})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	chat, err := b.GetChatByLink("mygroup")
	if err != nil {
		t.Fatalf("GetChatByLink error: %v", err)
	}
	if gotPath != "/chats/mygroup" {
		t.Errorf("expected path /chats/mygroup, got %q", gotPath)
	}
	if chat.ID != 7 {
		t.Errorf("expected chat_id=7, got %d", chat.ID)
	}
}

func TestGetChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/1" {
			t.Errorf("expected path /chats/1, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"chat_id": 1, "title": "Test"})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	chat, err := b.GetChat(1)
	if err != nil {
		t.Fatalf("GetChat error: %v", err)
	}
	if chat.Title != "Test" {
		t.Errorf("expected title Test, got %q", chat.Title)
	}
}

func TestUpdateChat(t *testing.T) {
	var gotMethod string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"chat_id": 1, "title": "New title"})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	chat, err := b.UpdateChat(1, map[string]interface{}{"title": "New title"})
	if err != nil {
		t.Fatalf("UpdateChat error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", gotMethod)
	}
	if gotBody["title"] != "New title" {
		t.Errorf("expected title in body, got %+v", gotBody)
	}
	if chat.Title != "New title" {
		t.Errorf("expected title New title, got %q", chat.Title)
	}
}

func TestDeleteChat(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.DeleteChat(1); err != nil {
		t.Fatalf("DeleteChat error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/chats/1" {
		t.Errorf("expected DELETE /chats/1, got %s %s", gotMethod, gotPath)
	}
}

func TestGetChatMemberMe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/1/members/me" {
			t.Errorf("expected path /chats/1/members/me, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"user_id": 5, "name": "Bot"})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	member, err := b.GetChatMemberMe(1)
	if err != nil {
		t.Fatalf("GetChatMemberMe error: %v", err)
	}
	if member.User == nil || member.User.ID != 5 {
		t.Errorf("expected user_id=5, got %+v", member)
	}
}

func TestGetChatMembersPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"members": []map[string]interface{}{{"user_id": 1, "name": "Alice"}},
			"marker":  99,
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	marker := int64(3)
	members, next, err := b.GetChatMembers(1, 20, &marker)
	if err != nil {
		t.Fatalf("GetChatMembers error: %v", err)
	}
	if gotQuery != "count=20&marker=3" {
		t.Errorf("unexpected query: %q", gotQuery)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if next == nil || *next != 99 {
		t.Errorf("expected next marker 99, got %v", next)
	}
}

// B1: the default permission set must exclude view_stats (not part of the
// ChatAdminPermission enum -- an owner-only capability the request would
// reject) and can_call (assigned automatically by MAX; explicit grants have
// no effect).
func TestPromoteChatMemberDefaultsSafePermissions(t *testing.T) {
	var gotBody struct {
		Admins []struct {
			UserID      int64    `json:"user_id"`
			Permissions []string `json:"permissions"`
		} `json:"admins"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.PromoteChatMember(1, 2); err != nil {
		t.Fatalf("PromoteChatMember error: %v", err)
	}
	if len(gotBody.Admins) != 1 || gotBody.Admins[0].UserID != 2 {
		t.Fatalf("unexpected admins body: %+v", gotBody.Admins)
	}
	for _, p := range gotBody.Admins[0].Permissions {
		if p == string(PermViewStats) || p == string(PermCanCall) {
			t.Errorf("default permissions must not include %q", p)
		}
	}
	if len(gotBody.Admins[0].Permissions) != len(defaultAdminPermissions) {
		t.Errorf("expected %d default permissions, got %v", len(defaultAdminPermissions), gotBody.Admins[0].Permissions)
	}
}

// PromoteChatMemberWithAlias must include the alias field when set.
func TestPromoteChatMemberWithAlias(t *testing.T) {
	var gotBody struct {
		Admins []struct {
			Alias string `json:"alias"`
		} `json:"admins"`
	}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))

	if err := b.PromoteChatMemberWithAlias(1, 2, "Moderator", PermPinMessage); err != nil {
		t.Fatalf("PromoteChatMemberWithAlias error: %v", err)
	}
	if len(gotBody.Admins) != 1 || gotBody.Admins[0].Alias != "Moderator" {
		t.Errorf("expected alias=Moderator, got %+v", gotBody.Admins)
	}
}

func TestPromoteChatMemberExplicitPermissions(t *testing.T) {
	var gotBody struct {
		Admins []struct {
			Permissions []string `json:"permissions"`
		} `json:"admins"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.PromoteChatMember(1, 2, PermPinMessage); err != nil {
		t.Fatalf("PromoteChatMember error: %v", err)
	}
	if len(gotBody.Admins[0].Permissions) != 1 || gotBody.Admins[0].Permissions[0] != string(PermPinMessage) {
		t.Errorf("expected only PermPinMessage, got %v", gotBody.Admins[0].Permissions)
	}
}

func TestDemoteChatMember(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.DemoteChatMember(1, 2); err != nil {
		t.Fatalf("DemoteChatMember error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/chats/1/members/admins/2" {
		t.Errorf("expected DELETE /chats/1/members/admins/2, got %s %s", gotMethod, gotPath)
	}
}

func TestInviteChatMembers(t *testing.T) {
	var gotBody struct {
		UserIDs []int64 `json:"user_ids"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.InviteChatMembers(1, []int64{2, 3}); err != nil {
		t.Fatalf("InviteChatMembers error: %v", err)
	}
	if len(gotBody.UserIDs) != 2 || gotBody.UserIDs[0] != 2 || gotBody.UserIDs[1] != 3 {
		t.Errorf("unexpected user_ids: %v", gotBody.UserIDs)
	}
}

func TestLeaveChat(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.LeaveChat(1); err != nil {
		t.Fatalf("LeaveChat error: %v", err)
	}
	if gotPath != "/chats/1/members/me" {
		t.Errorf("expected path /chats/1/members/me, got %q", gotPath)
	}
}

func TestPinMessageWithNotify(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	notify := false
	if err := b.PinMessage(1, "mid.1", &notify); err != nil {
		t.Fatalf("PinMessage error: %v", err)
	}
	if gotBody["message_id"] != "mid.1" {
		t.Errorf("expected message_id=mid.1, got %+v", gotBody)
	}
	if gotBody["notify"] != false {
		t.Errorf("expected notify=false, got %+v", gotBody["notify"])
	}
}

func TestUnpinMessage(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.UnpinMessage(1); err != nil {
		t.Fatalf("UnpinMessage error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/chats/1/pin" {
		t.Errorf("expected DELETE /chats/1/pin, got %s %s", gotMethod, gotPath)
	}
}

func TestGetPinnedMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chats/1/pin" {
			t.Errorf("expected path /chats/1/pin, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]interface{}{
				"timestamp": 1,
				"body":      map[string]interface{}{"mid": "mid.1", "text": "pinned"},
			},
		})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	msg, err := b.GetPinnedMessage(1)
	if err != nil {
		t.Fatalf("GetPinnedMessage error: %v", err)
	}
	if msg == nil || msg.Text() != "pinned" {
		t.Errorf("expected text=pinned, got %+v", msg)
	}
}

func TestGetPinnedMessageNone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"message": nil})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	msg, err := b.GetPinnedMessage(1)
	if err != nil {
		t.Fatalf("GetPinnedMessage error: %v", err)
	}
	if msg != nil {
		t.Errorf("expected nil message when nothing is pinned, got %+v", msg)
	}
}

func TestSendChatAction(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	b, _ := NewBot(Settings{Token: "tok", URL: srv.URL, Poller: &LongPoller{}})
	if err := b.SendChatAction(1, ActionTyping); err != nil {
		t.Fatalf("SendChatAction error: %v", err)
	}
	if gotBody["action"] != string(ActionTyping) {
		t.Errorf("expected action=%s, got %+v", ActionTyping, gotBody["action"])
	}
}
