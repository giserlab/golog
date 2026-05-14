package util

import (
	"bytes"
	"html/template"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/toc"
)

func MD2HTML(v string) template.HTML {
	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithExtensions(
			&mermaid.Extender{},
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
	return template.HTML(buf.Bytes())
}
