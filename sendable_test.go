package maxbot

import (
	"encoding/json"
	"net/http"
	"testing"
)

func captureAttachments(t *testing.T) (*Bot, *[]Attachment) {
	t.Helper()
	var got []Attachment
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Attachments []Attachment `json:"attachments"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		got = body.Attachments
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"message": map[string]interface{}{}})
	}))
	return b, &got
}

// Context.Reply's *SendOptions{ReplyToMid: ...} must reach the outgoing
// message for Sendable payloads (Sticker, Contact, Location, Share), not
// just plain text.
func TestSendableHonorsReplyToMid(t *testing.T) {
	var gotLink *linkedRef
	b := newTestBot(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Link *linkedRef `json:"link"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		gotLink = body.Link
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"message": map[string]interface{}{}})
	}))

	if _, err := b.Send(&User{ID: 1}, &Sticker{Code: "smile"}, &SendOptions{ReplyToMid: "mid.1"}); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if gotLink == nil || gotLink.Type != "reply" || gotLink.Mid != "mid.1" {
		t.Fatalf("expected reply link to mid.1, got %+v", gotLink)
	}
}

func TestPhotoSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	p := &Photo{PhotoTokens: PhotoTokens{Photos: map[string]PhotoToken{"0": {Token: "photo-tok"}}}}
	if _, err := b.Send(&User{ID: 1}, p); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "image" {
		t.Fatalf("expected 1 image attachment, got %+v", *got)
	}
}

// C10: Photo must also support the url/token attachment sources, not just
// freshly uploaded photo tokens.
func TestPhotoFromURLSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	if _, err := b.Send(&User{ID: 1}, PhotoFromURL("https://example.com/pic.jpg")); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Payload["url"] != "https://example.com/pic.jpg" {
		t.Fatalf("expected image attachment with url payload, got %+v", *got)
	}
}

func TestPhotoFromTokenSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	if _, err := b.Send(&User{ID: 1}, PhotoFromToken("reused-tok")); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Payload["token"] != "reused-tok" {
		t.Fatalf("expected image attachment with token payload, got %+v", *got)
	}
}

func TestVideoSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	v := &Video{UploadedInfo: UploadedInfo{Token: "video-tok"}}
	if _, err := b.Send(&User{ID: 1}, v); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "video" {
		t.Fatalf("expected 1 video attachment, got %+v", *got)
	}
	if (*got)[0].Payload["token"] != "video-tok" {
		t.Errorf("expected token=video-tok, got %v", (*got)[0].Payload["token"])
	}
}

func TestAudioSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	a := &Audio{UploadedInfo: UploadedInfo{Token: "audio-tok"}}
	if _, err := b.Send(&User{ID: 1}, a); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "audio" {
		t.Fatalf("expected 1 audio attachment, got %+v", *got)
	}
}

func TestDocumentSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	d := &Document{UploadedInfo: UploadedInfo{Token: "doc-tok"}}
	if _, err := b.Send(&User{ID: 1}, d); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "file" {
		t.Fatalf("expected 1 file attachment, got %+v", *got)
	}
}

func TestStickerSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	if _, err := b.Send(&User{ID: 1}, &Sticker{Code: "smile"}); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "sticker" {
		t.Fatalf("expected 1 sticker attachment, got %+v", *got)
	}
	if (*got)[0].Payload["code"] != "smile" {
		t.Errorf("expected code=smile, got %v", (*got)[0].Payload["code"])
	}
}

func TestContactSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	c := &Contact{Name: "Alice", ContactID: 7, VCFPhone: "+100"}
	if _, err := b.Send(&User{ID: 1}, c); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "contact" {
		t.Fatalf("expected 1 contact attachment, got %+v", *got)
	}
	p := (*got)[0].Payload
	if p["name"] != "Alice" || p["vcf_phone"] != "+100" {
		t.Errorf("unexpected contact payload: %+v", p)
	}
	if int64(p["contact_id"].(float64)) != 7 {
		t.Errorf("expected contact_id=7, got %v", p["contact_id"])
	}
}

func TestLocationSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	loc := &Location{Latitude: 55.75, Longitude: 37.61}
	if _, err := b.Send(&User{ID: 1}, loc); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "location" {
		t.Fatalf("expected 1 location attachment, got %+v", *got)
	}
	a := (*got)[0]
	if a.Latitude == nil || *a.Latitude != 55.75 || a.Longitude == nil || *a.Longitude != 37.61 {
		t.Errorf("expected top-level lat/long, got %+v", a)
	}
	if a.Payload != nil {
		t.Errorf("expected no payload wrapper for location, got %+v", a.Payload)
	}
}

func TestShareSendPayload(t *testing.T) {
	b, got := captureAttachments(t)
	if _, err := b.Send(&User{ID: 1}, &Share{URL: "https://example.com"}); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Type != "share" {
		t.Fatalf("expected 1 share attachment, got %+v", *got)
	}
	if (*got)[0].Payload["url"] != "https://example.com" {
		t.Errorf("expected url=https://example.com, got %v", (*got)[0].Payload["url"])
	}
}
