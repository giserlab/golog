package entity

import (
	"strings"
	"testing"
)

// TestPostCoverPrefersURL 验证外链封面优先于本地上传文件：
// 填写了 cover_url 时 Cover() 直接返回该链接。
func TestPostCoverPrefersURL(t *testing.T) {
	const url = "https://cdn.example.com/cover.jpg"
	p := &PostR{ID: "no-such-post", CoverURL: url}
	if got := p.Cover(); got != url {
		t.Fatalf("Cover() = %q, want %q", got, url)
	}

	// 未填写链接且本地没有对应文件时返回空串
	p2 := &PostR{ID: "no-such-post"}
	if got := p2.Cover(); got != "" {
		t.Fatalf("Cover() = %q, want empty", got)
	}
}

// TestPostExcerptStripsHighlight 验证摘要不会泄漏 ==高亮== 语法标记。
func TestPostExcerptStripsHighlight(t *testing.T) {
	p := &PostR{Content: "今天 ==发布了新版本==，详见 **更新日志**。"}
	got := p.Excerpt()
	if strings.Contains(got, "==") {
		t.Errorf("Excerpt() still contains highlight markers: %q", got)
	}
	if !strings.Contains(got, "发布了新版本") {
		t.Errorf("Excerpt() dropped highlighted text: %q", got)
	}
}
