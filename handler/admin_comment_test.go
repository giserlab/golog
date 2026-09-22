package handler

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"golog/entity"
	"golog/system"
	"golog/view"

	"github.com/gin-gonic/gin"
)

// TestAdminCommentsTemplateRendersReplyContext 校验后台留言列表能渲染回复上下文
// （“回复 @某某”）与跳转文章的链接，避免模板在执行期才报错。
func TestAdminCommentsTemplateRendersReplyContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restoreSystemState(t)

	system.Config = &entity.Config{
		Name: "测试站", Theme: "default", Locale: "zh-cn",
		DateFormat: "2006-01-02", TimeFormat: "15:04", PostsPerPage: 10,
	}
	system.ReloadLocale("zh-cn")

	tmpl, err := template.New("admin_base.html").Funcs(funcs).ParseFS(
		view.Templates,
		"templates/admin_base.html",
		"templates/admin_pagination.html",
		"templates/admin_comments.html",
	)
	if err != nil {
		t.Fatalf("parse admin comments templates: %v", err)
	}

	comments := []*entity.CommentR{
		{
			ID: "c-root", PostID: "p1", AuthorName: "楼主", AuthorEmail: "root@example.com",
			Content: "顶层留言", Status: "pending", CreatedAt: 1700000000,
		},
		{
			ID: "c-reply", PostID: "p1", ParentID: "c-root", ParentAuthor: "楼主",
			AuthorName: "访客", AuthorEmail: "visitor@example.com",
			Content: "回复内容", Status: "pending", CreatedAt: 1700000100,
		},
	}

	stats := map[[2]string]int{}
	momentStats := map[string]int{}
	tagMap := map[[2]string]int{}
	data := map[string]any{
		"Config": system.Config,
		"Self":   &entity.UserR{ID: "u1", Nickname: "管理员", Role: "admin", Email: "admin@example.com"},
		"URL": map[string]string{
			"Root": "http://localhost/", "Absolute": "http://localhost/admin/comments",
			"AbsoluteHost": "http://localhost/", "RelativeRoot": "../../", "PageType": "comment",
		},
		"Message": "", "CSRF": "csrf-token", "PendingCommentCount": 2,
		"Stats": &stats, "MomentStats": &momentStats, "TagMap": &tagMap,
		"Comments": comments,
		"Status":   "pending",
		"Pagination": &entity.Pagination{
			CurrentPage: 1, TotalCount: 2, TotalPages: 1, Query: "",
		},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "admin_base.html", data); err != nil {
		t.Fatalf("execute admin comments template: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"回复 楼主",
		"回复内容",
		"../../blog/p1",
		"admin/comment/approve",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered comments page missing %q", want)
		}
	}
}
