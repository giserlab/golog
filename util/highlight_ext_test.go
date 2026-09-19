package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gomarkdown/markdown"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func newHighlightGoldmark() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(extension.GFM, NewHighlightExt()),
	)
}

func TestHighlightExtension(t *testing.T) {
	md := newHighlightGoldmark()

	render := func(t *testing.T, input string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := md.Convert([]byte(input), &buf); err != nil {
			t.Fatalf("convert failed: %v", err)
		}
		return buf.String()
	}

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"chinese", "这是 ==重点== 内容", "<mark>重点</mark>"},
		{"english", "a ==highlight== word", "<mark>highlight</mark>"},
		{"multiple", "==一== 和 ==二==", "<mark>一</mark>"},
		{"nested emphasis", "==**重点**==", "<mark><strong>重点</strong></mark>"},
		{"adjacent text", "前缀==高亮==后缀", "<mark>高亮</mark>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := render(t, tc.input)
			if !strings.Contains(out, tc.want) {
				t.Errorf("expected %q in output, got: %s", tc.want, out)
			}
			if strings.Contains(out, "==") {
				t.Errorf("highlight markers should be consumed, got: %s", out)
			}
		})
	}
}

func TestHighlightExtensionNegative(t *testing.T) {
	md := newHighlightGoldmark()

	cases := []struct {
		name  string
		input string
	}{
		{"single equals", "a = b"},
		{"double equals as operator", "1 == 1"},
		{"unclosed", "==未闭合"},
		{"empty", "===="},
		{"triple equals", "===x==="},
		{"quad equals", "====x===="},
		{"space after opener", "== x =="},
		{"space before closer", "==x =="},
		{"inline code", "`==x==`"},
		{"fenced code", "```\n==x==\n```"},
		{"escaped", `\==x\==`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := md.Convert([]byte(tc.input), &buf); err != nil {
				t.Fatalf("convert failed: %v", err)
			}
			if out := buf.String(); strings.Contains(out, "<mark>") {
				t.Errorf("input %q must not be highlighted, got: %s", tc.input, out)
			}
		})
	}
}

// TestMD2HTMLHighlight 覆盖站点正文的真实渲染链路（goldmark + SanitizeHTML）。
func TestMD2HTMLHighlight(t *testing.T) {
	out := string(MD2HTML("正文 ==高亮内容== 结束"))
	if !strings.Contains(out, "<mark>高亮内容</mark>") {
		t.Errorf("highlight not rendered: %s", out)
	}

	// 高亮语法不能成为绕过 SanitizeHTML 的通道。
	evil := string(MD2HTML("==<script>alert(1)</script>=="))
	if strings.Contains(evil, "<script") || strings.Contains(evil, "alert(1)") {
		t.Errorf("script inside highlight survived sanitize: %s", evil)
	}
	if !strings.Contains(evil, "<mark>") {
		t.Errorf("mark element should survive sanitize: %s", evil)
	}
}

// TestGomarkdownHighlight 覆盖后台预览使用的 gomarkdown 渲染链路。
func TestGomarkdownHighlight(t *testing.T) {
	render := func(input string) string {
		p := parser.NewWithExtensions(parser.CommonExtensions)
		RegisterGomarkdownHighlight(p)
		doc := p.Parse([]byte(input))
		renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{
			RenderNodeHook: GomarkdownHighlightRenderHook,
		})
		return SanitizeHTML(string(markdown.Render(doc, renderer)))
	}

	if out := render("这是 ==重点== 内容"); !strings.Contains(out, "<mark>重点</mark>") {
		t.Errorf("gomarkdown highlight not rendered: %s", out)
	}
	if out := render("==**重点**=="); !strings.Contains(out, "<mark><strong>重点</strong></mark>") {
		t.Errorf("gomarkdown nested highlight not rendered: %s", out)
	}
	for _, input := range []string{"1 == 1", "===x===", "==未闭合", "`==x==`"} {
		if out := render(input); strings.Contains(out, "<mark>") {
			t.Errorf("input %q must not be highlighted, got: %s", input, out)
		}
	}
}
