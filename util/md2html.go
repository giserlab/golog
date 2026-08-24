package util

import (
	"bytes"
	"html/template"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/toc"
)

func MD2HTML(v string) template.HTML {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(extension.GFM, NewCustomFootnoteExt(), NewInlineAnnotationExt()),
		goldmark.WithExtensions(
			// NoScript: 不向文章内容注入 <script>（会被 SanitizeHTML 剥除且属于
			// 注入风险面），改由主题模板统一惰性加载 mermaid.js 并 initialize。
			&mermaid.Extender{NoScript: true},
			mathjax.MathJax,
			&toc.Extender{
				Title:   "目录",
				TitleID: "post-toc",
			},
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(v), &buf); err != nil {
		panic(err)
	}
	// 输出前必须经过 SanitizeHTML：md2html 以 WithUnsafe 模式渲染（支持
	// 文章中的原始 HTML/图表等），若不清理，低权限用户写入的 <script>
	// 等内容会在其他访问者/管理员浏览器中执行（存储型 XSS）。
	return template.HTML(SanitizeHTML(buf.String()))
}
