package store

import (
	"database/sql"

	"golog/entity"
)

// commentSelect 是所有留言查询共用的投影：除 comments 表自身的字段外，
// 通过自连接取出父留言的作者名（ParentAuthor），供后台展示“回复 @某某”。
const commentSelect = `SELECT c.id, c.post_id, c.parent_id, c.author_name, c.author_email, c.author_url, c.content, c.status, c.created_at, COALESCE(p.author_name, '') FROM comments c LEFT JOIN comments p ON p.id = c.parent_id`

// commentScanner 同时被 *sql.Row 与 *sql.Rows 满足，便于复用扫描逻辑。
type commentScanner interface {
	Scan(dest ...any) error
}

func scanComment(s commentScanner) (*entity.CommentR, error) {
	var c entity.CommentR
	if err := s.Scan(&c.ID, &c.PostID, &c.ParentID, &c.AuthorName, &c.AuthorEmail, &c.AuthorURL, &c.Content, &c.Status, &c.CreatedAt, &c.ParentAuthor); err != nil {
		return nil, err
	}
	return &c, nil
}

func CreateComment(c *entity.CommentW) error {
	_, err := db.Exec(`INSERT INTO comments (id, post_id, parent_id, author_name, author_email, author_url, content, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.PostID, c.ParentID, c.AuthorName, c.AuthorEmail, c.AuthorURL, c.Content, c.Status, c.CreatedAt)
	return err
}

func ListCommentsByPost(postID string) ([]*entity.CommentR, error) {
	rows, err := db.Query(commentSelect+` WHERE c.post_id = ? AND c.status = ? ORDER BY c.created_at ASC`, postID, "approved")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*entity.CommentR
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func ListCommentsByStatus(status string, offset, limit int) ([]*entity.CommentR, int, error) {
	var (
		rows  *sql.Rows
		err   error
		args  []any
		where = "WHERE 1 = 1"
	)
	if status != "" {
		where += " AND c.status = ?"
		args = append(args, status)
	}

	rows, err = db.Query(commentSelect+` `+where+` ORDER BY c.created_at DESC LIMIT ?, ?`, append(args, offset, limit)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*entity.CommentR
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comments c `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return comments, total, rows.Err()
}

func GetComment(id string) (*entity.CommentR, error) {
	return scanComment(db.QueryRow(commentSelect+` WHERE c.id = ?`, id))
}

func UpdateCommentStatus(id string, status string) error {
	_, err := db.Exec(`UPDATE comments SET status = ? WHERE id = ?`, status, id)
	return err
}

func DeleteComment(id string) error {
	_, err := db.Exec(`DELETE FROM comments WHERE id = ?`, id)
	return err
}

// CountReplies 统计某条留言下的回复数量（含待审/已驳回），用于删除父留言时
// 决定是否需要连带处理回复。
func CountReplies(parentID string) (int, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comments WHERE parent_id = ?`, parentID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
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

// DeleteCommentsByPost 删除文章下的全部留言（含回复）。
func DeleteCommentsByPost(postID string) error {
	_, err := db.Exec(`DELETE FROM comments WHERE post_id = ?`, postID)
	return err
}
