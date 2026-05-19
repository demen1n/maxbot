package maxbot

// Photo represents an uploaded image ready to send.
// Obtain via Bot.UploadPhoto.
type Photo struct {
	PhotoTokens
}

// Send implements Sendable interface for Photo.
func (p *Photo) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type: "image",
		Payload: map[string]interface{}{
			"photos": p.Photos,
		},
	}

	msg := newSendMessage(to)
	msg.Attachments = []Attachment{attachment}

	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}

	return b.sendMessage(msg)
}

// Video represents an uploaded video ready to send.
// Obtain via Bot.UploadMedia("video", ...).
type Video struct {
	UploadedInfo
}

// Send implements Sendable interface for Video.
func (v *Video) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type:    "video",
		Payload: map[string]interface{}{"token": v.Token},
	}
	msg := newSendMessage(to)
	msg.Attachments = []Attachment{attachment}
	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}
	return b.sendMessage(msg)
}

// Audio represents an uploaded audio file ready to send.
// Obtain via Bot.UploadMedia("audio", ...).
type Audio struct {
	UploadedInfo
}

// Send implements Sendable interface for Audio.
func (a *Audio) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type:    "audio",
		Payload: map[string]interface{}{"token": a.Token},
	}
	msg := newSendMessage(to)
	msg.Attachments = []Attachment{attachment}
	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}
	return b.sendMessage(msg)
}

// Document represents an uploaded file ready to send.
// Obtain via Bot.UploadMedia("file", ...).
type Document struct {
	UploadedInfo
}

// Send implements Sendable interface for Document.
func (d *Document) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type:    "file",
		Payload: map[string]interface{}{"token": d.Token},
	}
	msg := newSendMessage(to)
	msg.Attachments = []Attachment{attachment}
	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}
	return b.sendMessage(msg)
}
