package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func TestFootnoteRendering(t *testing.T) {
	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(extension.GFM, NewCustomFootnoteExt(), NewInlineAnnotationExt()),
	)

	t.Run("inline annotation {label: content}", func(t *testing.T) {
		input := "这是一段文本{注释一: 这是行内注释的内容}，继续写。"
		var buf bytes.Buffer
		if err := md.Convert([]byte(input), &buf); err != nil {
			t.Fatalf("convert failed: %v", err)
		}
		html := buf.String()

		// 应该渲染为 <sup class="annotation-ref" data-content="...">注释一</sup>
		if !strings.Contains(html, `<sup class="annotation-ref" data-content="这是行内注释的内容">注释一</sup>`) {
			t.Errorf("inline annotation not rendered correctly, got: %s", html)
		}
		// 底部不应该出现 footnotes 列表
		if strings.Contains(html, `class="footnotes"`) {
			t.Errorf("inline annotation should not produce footnotes list, got: %s", html)
		}
		// 不应该出现数字索引
		if strings.Contains(html, `>1</a>`) {
			t.Errorf("inline annotation should not use numeric index, got: %s", html)
		}
	})

	t.Run("standard footnote [^label]", func(t *testing.T) {
		input := "这是标准脚注[^标准标签]。\n\n[^标准标签]: 这是标准脚注的内容。"
		var buf bytes.Buffer
		if err := md.Convert([]byte(input), &buf); err != nil {
			t.Fatalf("convert failed: %v", err)
		}
		html := buf.String()

		// 应该出现 footnote-ref 链接（数字索引）
		if !strings.Contains(html, `class="footnote-ref"`) {
			t.Errorf("standard footnote ref not rendered, got: %s", html)
		}
		// 底部应该出现 footnotes 列表
		if !strings.Contains(html, `class="footnotes"`) {
			t.Errorf("standard footnote list not rendered, got: %s", html)
		}
		// 应该保留原始标签文本作为 data-ref
		if !strings.Contains(html, `data-ref="标准标签"`) {
			t.Errorf("standard footnote should preserve label as data-ref, got: %s", html)
		}
	})

	t.Run("mixed usage", func(t *testing.T) {
		input := "行内{注释A: 内容A}和标准[^标签B]共存。\n\n[^标签B]: 内容B。"
		var buf bytes.Buffer
		if err := md.Convert([]byte(input), &buf); err != nil {
			t.Fatalf("convert failed: %v", err)
		}
		html := buf.String()

		if !strings.Contains(html, `class="annotation-ref"`) {
			t.Errorf("inline annotation not found, got: %s", html)
		}
		if !strings.Contains(html, `class="footnote-ref"`) {
			t.Errorf("standard footnote ref not found, got: %s", html)
		}
		if !strings.Contains(html, `class="footnotes"`) {
			t.Errorf("standard footnote list not found, got: %s", html)
		}
	})

	t.Run("code block should not parse annotation", func(t *testing.T) {
		input := "```\n{注释: 内容}\n```"
		var buf bytes.Buffer
		if err := md.Convert([]byte(input), &buf); err != nil {
			t.Fatalf("convert failed: %v", err)
		}
		html := buf.String()

		// 代码块中的 {注释: 内容} 应该保持原样（被转义）
		if strings.Contains(html, `class="annotation-ref"`) {
			t.Errorf("code block content should not be parsed as annotation, got: %s", html)
		}
	})
}
