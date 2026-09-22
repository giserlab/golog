package store

import (
	"path/filepath"
	"testing"
	"time"

	"golog/entity"
)

// TestCommentReplyRoundTrip 验证 parent_id 在迁移、写入与各查询中被正确往返，
// 并且后台列表能通过自连接拿到父留言作者名。
func TestCommentReplyRoundTrip(t *testing.T) {
	if err := Open(filepath.Join(t.TempDir(), "test.sqlite")); err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer Open("golog.sqlite")
	if err := AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	if err := CreateUser(&entity.UserW{
		ID: "u1", Email: "u1@example.com", Nickname: "u1",
		Password: "x", Role: "admin", CreatedAt: 1,
	}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	now := time.Now().Unix()
	if err := CreatePost(&entity.PostW{
		ID: "p1", Type: "blog", Title: "标题", Slug: "slug", AuthorID: "u1",
		Visibility: entity.VisibilityPublic, Content: "正文",
		PublishedAt: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	root := &entity.CommentW{
		ID: "c-root", PostID: "p1", AuthorName: "楼主", AuthorEmail: "root@example.com",
		Content: "顶层留言", Status: "approved", CreatedAt: now,
	}
	reply := &entity.CommentW{
		ID: "c-reply", PostID: "p1", ParentID: "c-root", AuthorName: "访客",
		AuthorEmail: "visitor@example.com", Content: "回复内容", Status: "pending", CreatedAt: now + 1,
	}
	for _, c := range []*entity.CommentW{root, reply} {
		if err := CreateComment(c); err != nil {
			t.Fatalf("CreateComment(%s): %v", c.ID, err)
		}
	}

	got, err := GetComment("c-reply")
	if err != nil {
		t.Fatalf("GetComment: %v", err)
	}
	if got.ParentID != "c-root" {
		t.Fatalf("ParentID = %q, want c-root", got.ParentID)
	}
	if got.ParentAuthor != "楼主" {
		t.Fatalf("ParentAuthor = %q, want 楼主", got.ParentAuthor)
	}

	count, err := CountReplies("c-root")
	if err != nil {
		t.Fatalf("CountReplies: %v", err)
	}
	if count != 1 {
		t.Fatalf("CountReplies = %d, want 1", count)
	}

	// 公开列表只返回已通过审核的留言。
	approved, err := ListCommentsByPost("p1")
	if err != nil {
		t.Fatalf("ListCommentsByPost: %v", err)
	}
	if len(approved) != 1 || approved[0].ID != "c-root" {
		t.Fatalf("ListCommentsByPost = %#v, want only c-root", approved)
	}

	// 后台按状态列出时包含父留言作者，且回复通过后公开列表出现两条。
	all, total, err := ListCommentsByStatus("", 0, 10)
	if err != nil {
		t.Fatalf("ListCommentsByStatus: %v", err)
	}
	if total != 2 || len(all) != 2 {
		t.Fatalf("ListCommentsByStatus total = %d, len = %d, want 2/2", total, len(all))
	}
	if err := UpdateCommentStatus("c-reply", "approved"); err != nil {
		t.Fatalf("UpdateCommentStatus: %v", err)
	}
	approved, err = ListCommentsByPost("p1")
	if err != nil {
		t.Fatalf("ListCommentsByPost after approve: %v", err)
	}
	if len(approved) != 2 {
		t.Fatalf("ListCommentsByPost after approve = %d, want 2", len(approved))
	}

	tree := entity.BuildCommentTree(approved)
	if len(tree) != 1 || len(tree[0].Replies) != 1 {
		t.Fatalf("tree = %#v, want one root with one reply", tree)
	}

	if err := DeleteCommentsByPost("p1"); err != nil {
		t.Fatalf("DeleteCommentsByPost: %v", err)
	}
	remaining, _, err := ListCommentsByStatus("", 0, 10)
	if err != nil {
		t.Fatalf("ListCommentsByStatus after delete: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("comments were not deleted: %#v", remaining)
	}
}
