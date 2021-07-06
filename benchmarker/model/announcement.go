package model

type Announcement struct {
	ID      string
	Title   string
	Message string
}

func NewAnnouncement(id, title, message string) *Announcement {
	return &Announcement{
		ID:      id,
		Title:   title,
		Message: message,
	}
}
