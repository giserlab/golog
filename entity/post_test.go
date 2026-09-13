package entity

import "testing"

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
