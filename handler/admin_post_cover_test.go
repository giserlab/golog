package handler

import "testing"

// TestNormalizeCoverURL 验证后台“封面图片链接”的校验与规范化规则。
func TestNormalizeCoverURL(t *testing.T) {
	cases := []struct {
		in      string
		wantOK  bool
		wantOut string
	}{
		// 留空合法：表示不使用外链封面
		{"", true, ""},
		{"   ", true, ""},
		// 完整地址
		{"https://example.com/cover.jpg", true, "https://example.com/cover.jpg"},
		{"http://example.com/cover.jpg", true, "http://example.com/cover.jpg"},
		{"  https://example.com/cover.jpg  ", true, "https://example.com/cover.jpg"},
		// 站内根相对 / 协议相对
		{"/uploads/covers/a.jpg", true, "/uploads/covers/a.jpg"},
		{"//cdn.example.com/cover.jpg", true, "//cdn.example.com/cover.jpg"},
		// 无 scheme：按 https 外链补全
		{"cdn.example.com/cover.jpg", true, "https://cdn.example.com/cover.jpg"},
		// 危险或非法输入
		{"javascript:alert(1)", false, ""},
		{"data:image/png;base64,AAAA", false, ""},
		{"file:///etc/passwd", false, ""},
		{"ftp://example.com/cover.jpg", false, ""},
		{"https://", false, ""},
	}
	for _, tc := range cases {
		got, ok := normalizeCoverURL(tc.in)
		if ok != tc.wantOK || got != tc.wantOut {
			t.Errorf("normalizeCoverURL(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.wantOut, tc.wantOK)
		}
	}
}
