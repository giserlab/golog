package system

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"golog/entity"
)

// setupTheme 以指定主题加载全部模板，供留言渲染测试复用。
func setupTheme(t *testing.T, theme string) {
	t.Helper()

	original := Config
	Config = &entity.Config{
		Name:            "测试站",
		Description:     "描述",
		Theme:           theme,
		Locale:          "zh-cn",
		DateFormat:      "2006-01-02",
		TimeFormat:      "15:04",
		FontSize:        "16px",
		CommentsEnabled: true,
	}
	SetConfigWriter(func(*entity.Config) error { return nil })
	if err := SaveConfig(); err != nil {
		t.Fatalf("SaveConfig(%s): %v", theme, err)
	}
	t.Cleanup(func() { Config = original })
}

// singularData 构造与 handler.renderSingular 一致的数据形态。
func singularData(comments []*entity.CommentNode) map[string]any {
	tagMap := map[[2]string]int{}
	stats := map[[2]string]int{}
	momentStats := map[string]int{}
	post := &entity.PostR{
		Type:        "blog",
		ID:          "p1",
		Title:       "文章标题",
		Slug:        "post-title",
		Content:     "正文内容。",
		Visibility:  entity.VisibilityPublic,
		PublishedAt: 1750000000,
	}
	post.Author = entity.UserR{ID: "u1", Nickname: "作者", Email: "author@example.com"}

	return map[string]any{
		"Config": Config,
		"URL": map[string]string{
			"Root":         "http://localhost/",
			"Absolute":     "http://localhost/post/post-title",
			"AbsoluteHost": "http://localhost/",
			"RelativeRoot": "/",
			"PageType":     "post",
		},
		"Self":         nil,
		"Message":      "",
		"CSRF":         "csrf-token",
		"Stats":        &stats,
		"MomentStats":  &momentStats,
		"TagMap":       &tagMap,
		"Post":         post,
		"Navigations":  []*entity.NavigationR{},
		"Routes":       []entity.Route{{Name: "首页", Path: "/"}},
		"PreviousPost": nil,
		"NextPost":     nil,
		"IsUnlocked":   true,
		"Comments":     comments,
	}
}

func commentTree() []*entity.CommentNode {
	root := &entity.CommentR{
		ID: "c-root", PostID: "p1", AuthorName: "楼主", AuthorURL: "https://root.example.com",
		Content: "顶层留言内容", Status: "approved", CreatedAt: 1700000000,
	}
	reply := &entity.CommentR{
		ID: "c-reply", PostID: "p1", ParentID: "c-root", AuthorName: "访客",
		Content: "回复内容", Status: "approved", CreatedAt: 1700000100, ParentAuthor: "楼主",
	}
	return entity.BuildCommentTree([]*entity.CommentR{root, reply})
}

// TestThemeCommentReplyRendering 校验三个内置主题都能渲染留言回复：
// 回复挂在顶层留言下、带锚点、并暴露回复按钮与表单的 parent_id 字段。
func TestThemeCommentReplyRendering(t *testing.T) {
	for _, theme := range []string{"default", "note", "corporate"} {
		t.Run(theme, func(t *testing.T) {
			setupTheme(t, theme)

			var buf bytes.Buffer
			if err := SingularTmpl.Execute(&buf, singularData(commentTree())); err != nil {
				t.Fatalf("execute singular template: %v", err)
			}
			out := buf.String()

			for _, want := range []string{
				`id="comment-c-root"`,
				`id="comment-c-reply"`,
				`class="comment-replies"`,
				`顶层留言内容`,
				`回复内容`,
				`data-comment-id="c-root"`,
				`data-comment-id="c-reply"`,
				`name="parent_id"`,
				`class="comment-reply-hint"`,
				`/assets/comment.js`,
				`回复 楼主`, // 回复上下文（_f "comments_reply_to"）
			} {
				if !strings.Contains(out, want) {
					t.Errorf("rendered singular page missing %q", want)
				}
			}
		})
	}
}

// TestThemeCommentEmptyList 校验没有留言时只渲染表单，不出现空列表容器。
func TestThemeCommentEmptyList(t *testing.T) {
	for _, theme := range []string{"default", "note", "corporate"} {
		t.Run(theme, func(t *testing.T) {
			setupTheme(t, theme)

			var buf bytes.Buffer
			if err := SingularTmpl.Execute(&buf, singularData(nil)); err != nil {
				t.Fatalf("execute singular template: %v", err)
			}
			out := buf.String()
			if strings.Contains(out, `class="comments-list"`) {
				t.Error("empty comment list should not render the list container")
			}
			if !strings.Contains(out, `name="parent_id"`) {
				t.Error("comment form should always be rendered")
			}
		})
	}
}

// TestSharedCommentScriptLocation 回归测试：共享资源必须平铺在 themes/shared/ 下
// （AssetView 的回退路径是 themes/shared/<asset>，没有 assets/ 子目录）。
// 该脚本曾被放到 themes/shared/assets/comment.js，导致 /assets/comment.js 404，
// 页面点击「回复」没有任何反应。
func TestSharedCommentScriptLocation(t *testing.T) {
	if _, err := fs.Stat(ThemesFS, "themes/shared/comment.js"); err != nil {
		t.Fatalf("shared comment.js must live at themes/shared/comment.js: %v", err)
	}
	if _, err := fs.Stat(ThemesFS, "themes/shared/assets/comment.js"); err == nil {
		t.Error("comment.js must not be nested under themes/shared/assets/: AssetView would not serve it")
	}
}

// TestThemeCommentReplyHint 校验回复提示条（提示当前正在回复谁）在三个主题中
// 都带有本地化占位模板，且模板引用了共享脚本。
func TestThemeCommentReplyHint(t *testing.T) {
	for _, theme := range []string{"default", "note", "corporate"} {
		t.Run(theme, func(t *testing.T) {
			setupTheme(t, theme)

			var buf bytes.Buffer
			if err := SingularTmpl.Execute(&buf, singularData(commentTree())); err != nil {
				t.Fatalf("execute singular template: %v", err)
			}
			out := buf.String()
			for _, want := range []string{
				`class="comment-reply-hint"`,
				`role="status"`,
				`data-reply-placeholder="回复 @%s："`,
				`class="comment-reply-name"`,
				`class="comment-reply-cancel"`,
			} {
				if !strings.Contains(out, want) {
					t.Errorf("rendered singular page missing %q", want)
				}
			}
		})
	}
}
