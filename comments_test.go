package maxbot

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetCommentsBuildsQuery(t *testing.T) {
	var gotPath, gotQuery string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]interface{}{
				{"timestamp": 1, "body": map[string]interface{}{"mid": "mid.c1", "text": "hi"}},
			},
		})
	}))

	comments, err := b.GetComments("mid.post1", []string{"mid.c1", "mid.c2"}, 100, 200, 10)
	if err != nil {
		t.Fatalf("GetComments error: %v", err)
	}
	if gotPath != "/messages/mid.post1/comments" {
		t.Errorf("expected path /messages/mid.post1/comments, got %q", gotPath)
	}
	if gotQuery != "comment_ids=mid.c1,mid.c2&after=100&before=200&count=10&v="+APIVersion {
		t.Errorf("unexpected query: %q", gotQuery)
	}
	if len(comments) != 1 || comments[0].Text() != "hi" {
		t.Fatalf("unexpected comments: %+v", comments)
	}
}

// CommentMessage is an alias for Message, so it must pick up
// Message.UnmarshalJSON's ReplyTo auto-population, not just Sender/Body.
func TestGetCommentPopulatesReplyTo(t *testing.T) {
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"timestamp": 1,
			"body":      map[string]interface{}{"mid": "mid.c2", "text": "reply"},
			"link": map[string]interface{}{
				"type":    "reply",
				"message": map[string]interface{}{"mid": "mid.c1", "text": "original"},
			},
		})
	}))

	comment, err := b.GetComment("mid.post1", "mid.c2")
	if err != nil {
		t.Fatalf("GetComment error: %v", err)
	}
	if comment.ReplyTo == nil || comment.ReplyTo.Text() != "original" {
		t.Fatalf("expected ReplyTo populated from link, got %+v", comment.ReplyTo)
	}
}

func TestGetCommentSingle(t *testing.T) {
	var gotPath string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sender":    map[string]interface{}{"user_id": 1, "name": "Alice"},
			"timestamp": 123,
			"body":      map[string]interface{}{"mid": "mid.c1", "text": "hello"},
		})
	}))

	comment, err := b.GetComment("mid.post1", "mid.c1")
	if err != nil {
		t.Fatalf("GetComment error: %v", err)
	}
	if gotPath != "/messages/mid.post1/comments/mid.c1" {
		t.Errorf("expected path /messages/mid.post1/comments/mid.c1, got %q", gotPath)
	}
	if comment.Sender == nil || comment.Sender.ID != 1 || comment.Text() != "hello" {
		t.Errorf("unexpected comment: %+v", comment)
	}
}

func TestPostCommentUnwrapsEnvelope(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]interface{}{
				"timestamp": 1,
				"body":      map[string]interface{}{"mid": "mid.c1", "text": "hi there"},
			},
		})
	}))

	comment, err := b.PostComment("mid.post1", "hi there", "markdown")
	if err != nil {
		t.Fatalf("PostComment error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/messages/mid.post1/comments" {
		t.Errorf("expected POST /messages/mid.post1/comments, got %s %s", gotMethod, gotPath)
	}
	if gotBody["text"] != "hi there" || gotBody["format"] != "markdown" {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
	if comment.Text() != "hi there" {
		t.Errorf("expected unwrapped comment text, got %+v", comment)
	}
}

func TestEditCommentQueryAndFailure(t *testing.T) {
	var gotQuery string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "not allowed"})
	}))

	err := b.EditComment("mid.post1", "mid.c1", "edited")
	if err == nil || err.Error() != "not allowed" {
		t.Fatalf("expected 'not allowed' error, got %v", err)
	}
	if gotQuery != "comment_id=mid.c1&v="+APIVersion {
		t.Errorf("unexpected query: %q", gotQuery)
	}
}

func TestDeleteCommentQuery(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))

	if err := b.DeleteComment("mid.post1", "mid.c1"); err != nil {
		t.Fatalf("DeleteComment error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/messages/mid.post1/comments" {
		t.Errorf("expected DELETE /messages/mid.post1/comments, got %s %s", gotMethod, gotPath)
	}
	if gotQuery != "comment_id=mid.c1&v="+APIVersion {
		t.Errorf("unexpected query: %q", gotQuery)
	}
}
