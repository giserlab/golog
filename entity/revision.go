package entity

type PostRevision struct {
	ID          string
	PostID      string
	Type        string
	Title       string
	Slug        string
	Excerpt     string
	Password    string
	Visibility  Visibility
	Content     string
	PublishedAt int64
	PinnedAt    int64
	Tags        string
	CreatedAt   int64
	CreatedBy   string
}
