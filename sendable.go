package maxbot

// sendAttachment builds and sends a single-attachment message, applying
// Text/Format/ReplyToMid and any extra attachments (e.g. a keyboard) from
// opts. Shared by every Sendable implementation below.
func (b *Bot) sendAttachment(to Recipient, attachment Attachment, opts *SendOptions) (*Message, error) {
	msg := newSendMessage(to)
	msg.Attachments = []Attachment{attachment}

	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
		msg.Attachments = append(msg.Attachments, opts.Attachments...)
		if opts.ReplyToMid != "" {
			msg.Link = &linkedRef{Type: "reply", Mid: opts.ReplyToMid}
		}
	}

	return b.sendMessage(msg)
}

// Photo represents an uploaded image ready to send.
// Obtain via Bot.UploadPhoto.
type Photo struct {
	PhotoTokens
}

// Send implements Sendable interface for Photo.
func (p *Photo) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	return b.sendAttachment(to, Attachment{
		Type: "image",
		Payload: map[string]interface{}{
			"photos": p.Photos,
		},
	}, opts)
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
	}, opts)
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
	}, opts)
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
	}, opts)
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
	}, opts)
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
	return b.sendAttachment(to, Attachment{Type: "contact", Payload: payload}, opts)
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
	return b.sendAttachment(to, Attachment{Type: "location", Latitude: &lat, Longitude: &lon}, opts)
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
	}, opts)
}
