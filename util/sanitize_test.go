package util

import (
	"strings"
	"testing"
)

func TestSanitizeHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // 期望输出中包含的片段（剥离后应为空/无这些内容）
	}{
		{name: "script tag removed", in: `<p>hello<script>alert(1)</script></p>`, want: "script"},
		{name: "applet removed", in: `<applet code="x"></applet>`, want: "applet"},
		{name: "frameset removed", in: `<frameset><frame src="x"></frameset>`, want: "frameset"},
		{name: "onerror removed", in: `<img src="x.png" onerror="alert(1)">`, want: "onerror"},
		{name: "onload removed", in: `<svg onload="alert(1)"></svg>`, want: "onload"},
		{name: "svg removed", in: `<svg><script>alert(1)</script></svg>`, want: "svg"},
		{name: "javascript href removed", in: `<a href="javascript:alert(1)">x</a>`, want: "javascript"},
		{name: "obfuscated javascript href removed", in: `<a href="java&#10;script:alert(1)">x</a>`, want: "javascript"},
		{name: "mixed case javascript removed", in: `<a href="JaVaScRiPt:alert(1)">x</a>`, want: "javascript"},
		{name: "form removed", in: `<form action="/x"><input name="a"></form>`, want: "form"},
		{name: "style removed", in: `<style>body{display:none}</style>`, want: "style"},
		{name: "iframe javascript src removed", in: `<iframe src="javascript:alert(1)"></iframe>`, want: "javascript"},
		{name: "iframe data src removed", in: `<iframe src="data:text/html,<script>1</script>"></iframe>`, want: "src="},
		{name: "iframe srcdoc removed", in: `<iframe srcdoc="&lt;script&gt;parent.x()&lt;/script&gt;"></iframe>`, want: "srcdoc"},
		{name: "iframe onload removed", in: `<iframe src="https://e.com" onload="alert(1)"></iframe>`, want: "onload"},
		{name: "object javascript data removed", in: `<object data="javascript:alert(1)"></object>`, want: "javascript"},
		{name: "embed javascript src removed", in: `<embed src="javascript:alert(1)">`, want: "javascript"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out := SanitizeHTML(tt.in)
			if strings.Contains(out, tt.want) {
				t.Errorf("SanitizeHTML(%q) = %q, still contains %q", tt.in, out, tt.want)
			}
		})
	}
}

func TestSanitizeHTMLKeepsSafeContent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "paragraph kept", in: `<p>hello <strong>world</strong></p>`, want: "<p>hello <strong>world</strong></p>"},
		{name: "image kept", in: `<img src="https://example.com/a.png" alt="a">`, want: `src="https://example.com/a.png"`},
		{name: "data image kept", in: `<img src="data:image/png;base64,iVBORw0KGgo=">`, want: "data:image/png"},
		{name: "link kept", in: `<a href="/post/foo" rel="nofollow">link</a>`, want: `href="/post/foo"`},
		{name: "code block kept", in: `<pre><code class="language-go">fmt.Println("hi")</code></pre>`, want: "language-go"},
		{name: "data attribute kept", in: `<div data-foo="bar">x</div>`, want: `data-foo="bar"`},
		{name: "iframe kept with https src", in: `<iframe src="https://charts.example.com/embed/1" width="100%" height="400" allowfullscreen title="chart"></iframe>`, want: `src="https://charts.example.com/embed/1"`},
		{name: "embed kept with https src", in: `<embed src="https://cdn.example.com/file.pdf">`, want: `src="https://cdn.example.com/file.pdf"`},
		{name: "object kept with https data", in: `<object data="https://cdn.example.com/file.pdf"></object>`, want: `data="https://cdn.example.com/file.pdf"`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out := SanitizeHTML(tt.in)
			if !strings.Contains(out, tt.want) {
				t.Errorf("SanitizeHTML(%q) = %q, missing %q", tt.in, out, tt.want)
			}
		})
	}
}

func TestMD2HTMLSanitized(t *testing.T) {
	// 含恶意脚本的 markdown 输出不得保留可执行载荷
	evil := "# Title\n\n<script>alert('xss')</script>\n\n<img src=x onerror=alert(1)>\n\n[link](javascript:alert(1))"
	out := string(MD2HTML(evil))
	for _, bad := range []string{"<script", "onerror", "javascript:"} {
		if strings.Contains(out, bad) {
			t.Errorf("MD2HTML output still contains %q: %s", bad, out)
		}
	}

	// 正常内容应保留
	normal := string(MD2HTML("**bold** and [a link](https://example.com)"))
	if !strings.Contains(normal, "<strong>bold</strong>") || !strings.Contains(normal, `href="https://example.com"`) {
		t.Errorf("MD2HTML mangled safe markdown: %s", normal)
	}
}

func TestMD2HTMLKeepsIframe(t *testing.T) {
	// 历史博客中的内嵌图表（iframe）应保留，且 srcdoc/on* 仍被剥除
	md := "## Chart\n\n<iframe src=\"https://charts.example.com/embed/42\" width=\"100%\" height=\"500\" allowfullscreen title=\"chart\" srcdoc=\"<script>alert(1)</script>\" onload=\"alert(1)\"></iframe>"
	out := string(MD2HTML(md))
	for _, bad := range []string{"srcdoc", "onload", "<script"} {
		if strings.Contains(out, bad) {
			t.Errorf("MD2HTML output still contains %q: %s", bad, out)
		}
	}
	if !strings.Contains(out, `src="https://charts.example.com/embed/42"`) {
		t.Errorf("MD2HTML dropped iframe src: %s", out)
	}
	if !strings.Contains(out, "<iframe") {
		t.Errorf("MD2HTML dropped iframe tag: %s", out)
	}
}
