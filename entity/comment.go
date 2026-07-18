package entity

import "time"

type CommentW struct {
	ID           string
	PostID       string
	AuthorName   string
	AuthorEmail  string
	AuthorURL    string
	Content      string
	Status       string // pending | approved | rejected
	CreatedAt    int64
}

type CommentR struct {
	ID           string
	PostID       string
	AuthorName   string
	AuthorEmail  string
	AuthorURL    string
	Content      string
	Status       string
	CreatedAt    int64
}

func (c *CommentR) CreatedDate() string {
	return time.Unix(c.CreatedAt, 0).Format("2006-01-02 15:04")
}
