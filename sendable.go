package maxbot

import "fmt"

// exclusivity constrains which extra attachments sendAttachment permits
// alongside the primary one, per spec constraints on certain attachment types.
type exclusivity int

const (
	// exclusivityNone allows any combination (image, video, audio, location, share).
	exclusivityNone exclusivity = iota
	// exclusivitySolo requires the attachment to be the only one in the
	// message (sticker, contact) -- not even a keyboard.
	exclusivitySolo
	// exclusivityKeyboardOnly allows exactly one inline_keyboard attachment
	// alongside (file: "можно отправить только в комбинации с вложением с
	// кнопками").
	exclusivityKeyboardOnly
)

// sendAttachment builds and sends a single-attachment message, applying
// Text/Format/ReplyToMid and any extra attachments (e.g. a keyboard) from
// opts. Shared by every Sendable implementation below.
func (b *Bot) sendAttachment(to Recipient, attachment Attachment, opts *SendOptions, excl exclusivity) (*Message, error) {
	if err := checkExclusivity(attachment.Type, opts, excl); err != nil {
		return nil, err
	}

	msg := newSendMessage(to)
	msg.Attachments = []Attachment{attachment}

	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
		msg.Attachments = append(msg.Attachments, opts.Attachments...)
		if opts.ReplyToMid != "" {
			msg.Link = &linkedRef{Type: "reply", Mid: opts.ReplyToMid}
		}
		msg.Notify = opts.Notify
		msg.DisableLinkPreview = opts.DisableLinkPreview
	}

	return b.sendMessage(msg)
}

func checkExclusivity(attachmentType string, opts *SendOptions, excl exclusivity) error {
	if opts == nil || len(opts.Attachments) == 0 {
		return nil
	}
	switch excl {
	case exclusivitySolo:
		return fmt.Errorf("maxbot: %q must be the only attachment in the message, got %d extra", attachmentType, len(opts.Attachments))
	case exclusivityKeyboardOnly:
		if len(opts.Attachments) > 1 || opts.Attachments[0].Type != "inline_keyboard" {
			return fmt.Errorf("maxbot: %q can only be combined with a single inline_keyboard attachment", attachmentType)
		}
	}
	return nil
}

// Photo represents an image to send, via one of three mutually exclusive
// sources: freshly uploaded tokens (the zero value plus PhotoTokens, as
// returned by Bot.UploadPhoto), an external URL (PhotoFromURL), or a
// previously uploaded attachment's reusable token (PhotoFromToken).
type Photo struct {
	PhotoTokens
	url   string
	token string
}

// PhotoFromURL creates a Photo that attaches an external image URL without
// uploading it first.
func PhotoFromURL(url string) *Photo {
	return &Photo{url: url}
}

// PhotoFromToken creates a Photo that reuses a previously uploaded image's
// token (see the /uploads docs on reusing tokens for frequently sent files).
func PhotoFromToken(token string) *Photo {
	return &Photo{token: token}
}

// Send implements Sendable interface for Photo.
func (p *Photo) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	payload := map[string]interface{}{}
	switch {
	case p.url != "":
		payload["url"] = p.url
	case p.token != "":
		payload["token"] = p.token
	default:
		payload["photos"] = p.Photos
	}
	return b.sendAttachment(to, Attachment{Type: "image", Payload: payload}, opts, exclusivityNone)
}

// Video represents an uploaded video ready to send.
// Obtain via Bot.UploadMedia("video", ...).
type Video struct {
	UploadedInfo
}

// Send implements Sendable interface for Video.
func (v *Video) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	return b.sendAttachment(to, Attachment{
		Type:    "video",
		Payload: map[string]interface{}{"token": v.Token},
	}, opts, exclusivityNone)
}

// Audio represents an uploaded audio file ready to send.
// Obtain via Bot.UploadMedia("audio", ...).
type Audio struct {
	UploadedInfo
}

// Send implements Sendable interface for Audio.
func (a *Audio) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	return b.sendAttachment(to, Attachment{
		Type:    "audio",
		Payload: map[string]interface{}{"token": a.Token},
	}, opts, exclusivitySolo)
}

// Document represents an uploaded file ready to send.
// Obtain via Bot.UploadMedia("file", ...).
type Document struct {
	UploadedInfo
}

// Send implements Sendable interface for Document.
func (d *Document) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	return b.sendAttachment(to, Attachment{
		Type:    "file",
		Payload: map[string]interface{}{"token": d.Token},
	}, opts, exclusivityKeyboardOnly)
}

// Sticker represents a sticker to send, identified by its code.
// Per spec it must be the only attachment in the message.
type Sticker struct {
	Code string
}

// Send implements Sendable interface for Sticker.
func (s *Sticker) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	return b.sendAttachment(to, Attachment{
		Type:    "sticker",
		Payload: map[string]interface{}{"code": s.Code},
	}, opts, exclusivitySolo)
}

// Contact represents a contact card to send.
// Per spec it must be the only attachment in the message.
type Contact struct {
	Name      string
	ContactID int64  // MAX user id, if the contact is a registered user
	VCFInfo   string // full contact info in VCF format
	VCFPhone  string // contact phone in VCF format
}

// Send implements Sendable interface for Contact.
func (c *Contact) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	payload := map[string]interface{}{"name": c.Name}
	if c.ContactID != 0 {
		payload["contact_id"] = c.ContactID
	}
	if c.VCFInfo != "" {
		payload["vcf_info"] = c.VCFInfo
	}
	if c.VCFPhone != "" {
		payload["vcf_phone"] = c.VCFPhone
	}
	return b.sendAttachment(to, Attachment{Type: "contact", Payload: payload}, opts, exclusivitySolo)
}

// Location represents a geographic point to send.
type Location struct {
	Latitude  float64
	Longitude float64
}

// Send implements Sendable interface for Location.
// Per spec latitude/longitude are top-level attachment fields, not payload.
func (l *Location) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	lat, lon := l.Latitude, l.Longitude
	return b.sendAttachment(to, Attachment{Type: "location", Latitude: &lat, Longitude: &lon}, opts, exclusivityNone)
}

// Share attaches a media preview of an external URL to a message.
type Share struct {
	URL string
}

// Send implements Sendable interface for Share.
func (s *Share) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	return b.sendAttachment(to, Attachment{
		Type:    "share",
		Payload: map[string]interface{}{"url": s.URL},
	}, opts, exclusivityNone)
}
