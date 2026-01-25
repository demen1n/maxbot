package maxbot

// Photo represents a photo attachment.
type Photo struct {
	FileID string `json:"file_id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url,omitempty"`
}

// Send implements Sendable interface for Photo.
func (p *Photo) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type: "image",
		Payload: map[string]interface{}{
			"file_id": p.FileID,
		},
	}

	msg := &SendMessage{
		ChatID:      to.Recipient(),
		Attachments: []Attachment{attachment},
	}

	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}

	return b.sendMessage(msg)
}

// Video represents a video attachment.
type Video struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
	Token    string `json:"token,omitempty"`
}

// Send implements Sendable interface for Video.
func (v *Video) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type: "video",
		Payload: map[string]interface{}{
			"token": v.Token,
		},
	}

	msg := &SendMessage{
		ChatID:      to.Recipient(),
		Attachments: []Attachment{attachment},
	}

	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}

	return b.sendMessage(msg)
}

// Audio represents an audio file.
type Audio struct {
	FileID    string `json:"file_id"`
	Duration  int    `json:"duration"`
	Title     string `json:"title,omitempty"`
	Performer string `json:"performer,omitempty"`
	Token     string `json:"token,omitempty"`
}

// Send implements Sendable interface for Audio.
func (a *Audio) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type: "audio",
		Payload: map[string]interface{}{
			"token": a.Token,
		},
	}

	msg := &SendMessage{
		ChatID:      to.Recipient(),
		Attachments: []Attachment{attachment},
	}

	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}

	return b.sendMessage(msg)
}

// Document represents a document file.
type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int    `json:"file_size"`
	Token    string `json:"token,omitempty"`
}

// Send implements Sendable interface for Document.
func (d *Document) Send(b *Bot, to Recipient, opts *SendOptions) (*Message, error) {
	attachment := Attachment{
		Type: "file",
		Payload: map[string]interface{}{
			"token": d.Token,
		},
	}

	msg := &SendMessage{
		ChatID:      to.Recipient(),
		Attachments: []Attachment{attachment},
	}

	if opts != nil {
		msg.Text = opts.Text
		msg.Format = opts.Format
	}

	return b.sendMessage(msg)
}
