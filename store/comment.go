package store

import (
	"database/sql"

	"golog/entity"
)

func CreateComment(c *entity.CommentW) error {
	_, err := db.Exec(`INSERT INTO comments (id, post_id, author_name, author_email, author_url, content, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.PostID, c.AuthorName, c.AuthorEmail, c.AuthorURL, c.Content, c.Status, c.CreatedAt)
	return err
}

func ListCommentsByPost(postID string) ([]*entity.CommentR, error) {
	rows, err := db.Query(`SELECT id, post_id, author_name, author_email, author_url, content, status, created_at FROM comments WHERE post_id = ? AND status = ? ORDER BY created_at ASC`, postID, "approved")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*entity.CommentR
	for rows.Next() {
		var c entity.CommentR
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.AuthorEmail, &c.AuthorURL, &c.Content, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	return comments, rows.Err()
}

func ListCommentsByStatus(status string, offset, limit int) ([]*entity.CommentR, int, error) {
	var (
		rows      *sql.Rows
		err       error
		args      []any
		where     = "WHERE 1 = 1"
	)
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	rows, err = db.Query(`SELECT id, post_id, author_name, author_email, author_url, content, status, created_at FROM comments `+where+` ORDER BY created_at DESC LIMIT ?, ?`, append(args, offset, limit)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*entity.CommentR
	for rows.Next() {
		var c entity.CommentR
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.AuthorEmail, &c.AuthorURL, &c.Content, &c.Status, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		comments = append(comments, &c)
	}

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comments `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return comments, total, rows.Err()
}

func GetComment(id string) (*entity.CommentR, error) {
	var c entity.CommentR
	if err := db.QueryRow(`SELECT id, post_id, author_name, author_email, author_url, content, status, created_at FROM comments WHERE id = ?`, id).Scan(
		&c.ID, &c.PostID, &c.AuthorName, &c.AuthorEmail, &c.AuthorURL, &c.Content, &c.Status, &c.CreatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func UpdateCommentStatus(id string, status string) error {
	_, err := db.Exec(`UPDATE comments SET status = ? WHERE id = ?`, status, id)
	return err
}

func DeleteComment(id string) error {
	_, err := db.Exec(`DELETE FROM comments WHERE id = ?`, id)
	return err
}

func CountCommentsByPost(postID string) (int, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comments WHERE post_id = ? AND status = ?`, postID, "approved").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func CountByStatus(status string) (int, error) {
	var query string
	var args []any
	if status == "" {
		query = `SELECT COUNT(*) FROM comments`
	} else {
		query = `SELECT COUNT(*) FROM comments WHERE status = ?`
		args = append(args, status)
	}
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func DeleteCommentsByPost(postID string) error {
	_, err := db.Exec(`DELETE FROM comments WHERE post_id = ?`, postID)
	return err
}
