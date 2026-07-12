package store

import (
	"golog/entity"
)

func CreatePostRevision(rev *entity.PostRevision) error {
	_, err := db.Exec(`INSERT INTO post_revisions (id, post_id, type, title, slug, excerpt, password, visibility, content, published_at, pinned_at, tags, created_at, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rev.ID, rev.PostID, rev.Type, rev.Title, rev.Slug, rev.Excerpt, rev.Password, rev.Visibility, rev.Content, rev.PublishedAt, rev.PinnedAt, rev.Tags, rev.CreatedAt, rev.CreatedBy)
	return err
}

func ListPostRevisions(postID string) ([]*entity.PostRevision, error) {
	rows, err := db.Query(`SELECT id, post_id, type, title, slug, excerpt, password, visibility, content, published_at, pinned_at, tags, created_at, created_by FROM post_revisions WHERE post_id = ? ORDER BY created_at DESC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revs []*entity.PostRevision
	for rows.Next() {
		var r entity.PostRevision
		if err := rows.Scan(&r.ID, &r.PostID, &r.Type, &r.Title, &r.Slug, &r.Excerpt, &r.Password, &r.Visibility, &r.Content, &r.PublishedAt, &r.PinnedAt, &r.Tags, &r.CreatedAt, &r.CreatedBy); err != nil {
			return nil, err
		}
		revs = append(revs, &r)
	}
	return revs, rows.Err()
}

func GetPostRevision(id string) (*entity.PostRevision, error) {
	var r entity.PostRevision
	if err := db.QueryRow(`SELECT id, post_id, type, title, slug, excerpt, password, visibility, content, published_at, pinned_at, tags, created_at, created_by FROM post_revisions WHERE id = ?`, id).Scan(
		&r.ID, &r.PostID, &r.Type, &r.Title, &r.Slug, &r.Excerpt, &r.Password, &r.Visibility, &r.Content, &r.PublishedAt, &r.PinnedAt, &r.Tags, &r.CreatedAt, &r.CreatedBy); err != nil {
		return nil, err
	}
	return &r, nil
}

func DeletePostRevision(id string) error {
	_, err := db.Exec(`DELETE FROM post_revisions WHERE id = ?`, id)
	return err
}

func DeletePostRevisionsByPost(postID string) error {
	_, err := db.Exec(`DELETE FROM post_revisions WHERE post_id = ?`, postID)
	return err
}

func TrimPostRevisions(postID string, max int) error {
	_, err := db.Exec(`DELETE FROM post_revisions WHERE post_id = ? AND id NOT IN (SELECT id FROM post_revisions WHERE post_id = ? ORDER BY created_at DESC LIMIT ?)`, postID, postID, max)
	return err
}
