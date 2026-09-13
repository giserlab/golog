package store

import (
	"path/filepath"
	"testing"
	"time"

	"golog/entity"
)

// TestPostCoverURLRoundTrip 验证 posts.cover_url 在创建、读取、更新、列表
// 各条 SQL 中都被正确往返，避免列顺序错位导致外链封面丢失。
func TestPostCoverURLRoundTrip(t *testing.T) {
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
	const coverURL = "https://cdn.example.com/cover.jpg"
	if err := CreatePost(&entity.PostW{
		ID: "p1", Type: "blog", Title: "标题", Slug: "slug", AuthorID: "u1",
		Visibility: entity.VisibilityPublic, Content: "正文",
		CoverURL:    coverURL,
		PublishedAt: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	got, err := GetPost("p1")
	if err != nil {
		t.Fatalf("GetPost: %v", err)
	}
	if got.CoverURL != coverURL {
		t.Fatalf("GetPost cover_url = %q, want %q", got.CoverURL, coverURL)
	}
	if got.Cover() != coverURL {
		t.Fatalf("Cover() = %q, want %q", got.Cover(), coverURL)
	}

	bySlug, err := GetPostBySlug("slug")
	if err != nil {
		t.Fatalf("GetPostBySlug: %v", err)
	}
	if bySlug.CoverURL != coverURL {
		t.Fatalf("GetPostBySlug cover_url = %q, want %q", bySlug.CoverURL, coverURL)
	}

	byID, err := GetPostByID("p1")
	if err != nil {
		t.Fatalf("GetPostByID: %v", err)
	}
	if byID.CoverURL != coverURL {
		t.Fatalf("GetPostByID cover_url = %q, want %q", byID.CoverURL, coverURL)
	}

	list, err := ListPosts(&ListPostsQuery{Type: "blog", Limit: 10})
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(list) != 1 || list[0].CoverURL != coverURL {
		t.Fatalf("ListPosts cover_url = %#v, want %q", list, coverURL)
	}

	listAll, err := ListallPosts(&ListPostsQuery{Type: "blog", Limit: 10})
	if err != nil {
		t.Fatalf("ListallPosts: %v", err)
	}
	if len(listAll) != 1 || listAll[0].CoverURL != coverURL {
		t.Fatalf("ListallPosts cover_url = %#v, want %q", listAll, coverURL)
	}

	// 更新为站内相对路径
	const nextURL = "/uploads/covers/p1.png"
	if err := UpdatePost(&entity.PostW{
		ID: "p1", Type: "blog", Title: "标题2", Slug: "slug", AuthorID: "u1",
		Visibility: entity.VisibilityPublic, Content: "正文2",
		CoverURL:    nextURL,
		PublishedAt: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("UpdatePost: %v", err)
	}
	updated, err := GetPost("p1")
	if err != nil {
		t.Fatalf("GetPost after update: %v", err)
	}
	if updated.CoverURL != nextURL {
		t.Fatalf("after UpdatePost cover_url = %q, want %q", updated.CoverURL, nextURL)
	}
}
