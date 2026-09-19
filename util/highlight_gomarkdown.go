package util

import (
	"io"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// ==================== gomarkdown 的 ==高亮内容== 支持 ====================
//
// 站点正文（主题模板的 md2html）走 goldmark，由 highlight_ext.go 提供
// ==高亮== 语法；后台的 Markdown 预览 / 历史版本仍在用 gomarkdown，
// 这里补上同一套语法，避免同一篇内容在两处渲染结果不一致。

// HighlightNode 是 gomarkdown 中 ==高亮内容== 的 AST 节点，渲染为 <mark>。
type HighlightNode struct {
	ast.Container
}

// RegisterGomarkdownHighlight 为 gomarkdown 解析器注册 ==高亮== 行内语法。
func RegisterGomarkdownHighlight(p *parser.Parser) {
	p.RegisterInline('=', parseGomarkdownHighlight)
}

// parseGomarkdownHighlight 解析 ==高亮内容==，返回消费的字节数与节点。
// 规则与 goldmark 扩展保持一致：恰好两个等号、不跨行、内容首尾不能是空白。
func parseGomarkdownHighlight(p *parser.Parser, data []byte, offset int) (int, ast.Node) {
	// 连续三个及以上等号（===）不视为高亮语法。
	if offset > 0 && data[offset-1] == '=' {
		return 0, nil
	}
	data = data[offset:]
	if len(data) < 5 || data[0] != '=' || data[1] != '=' {
		return 0, nil
	}
	if parser.IsSpace(data[2]) {
		return 0, nil
	}

	end := -1
	for i := 2; i < len(data)-1; i++ {
		if data[i] == '\n' {
			break
		}
		if data[i] != '=' || data[i+1] != '=' || data[i-1] == '=' {
			continue
		}
		if i+2 < len(data) && data[i+2] == '=' {
			continue
		}
		if parser.IsSpace(data[i-1]) {
			continue
		}
		end = i
		break
	}
	if end < 3 {
		return 0, nil
	}

	node := &HighlightNode{}
	p.Inline(node, data[2:end])
	return end + 2, node
}

// GomarkdownHighlightRenderHook 渲染 HighlightNode，其余节点交回 gomarkdown
// 默认渲染器（返回 false）。通过 html.RendererOptions.RenderNodeHook 注册。
func GomarkdownHighlightRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if _, ok := node.(*HighlightNode); !ok {
		return ast.GoToNext, false
	}
	if entering {
		_, _ = io.WriteString(w, "<mark>")
	} else {
		_, _ = io.WriteString(w, "</mark>")
	}
	return ast.GoToNext, true
}
