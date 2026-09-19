package util

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ==================== 行内高亮 ==高亮内容== ====================

// KindHighlight 是行内高亮 AST 节点的类型标识。
var KindHighlight = gast.NewNodeKind("Highlight")

// Highlight 表示 ==高亮内容== 语法的 AST 节点，最终渲染为 <mark>。
type Highlight struct {
	gast.BaseInline
}

// Kind 实现 gast.Node。
func (n *Highlight) Kind() gast.NodeKind {
	return KindHighlight
}

// Dump 实现 gast.Node。
func (n *Highlight) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// highlightDelimiterProcessor 负责把成对的 "==" 配对成 Highlight 节点。
type highlightDelimiterProcessor struct{}

func (p *highlightDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '='
}

func (p *highlightDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *highlightDelimiterProcessor) OnMatch(consumes int) gast.Node {
	return &Highlight{}
}

var defaultHighlightDelimiterProcessor = &highlightDelimiterProcessor{}

// highlightParser 解析 ==高亮内容==；中间的内容仍按行内语法解析，
// 因此 ==**重点**== 这类嵌套写法可以正常渲染。
type highlightParser struct{}

func (p *highlightParser) Trigger() []byte {
	return []byte{'='}
}

func (p *highlightParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, defaultHighlightDelimiterProcessor)
	// 只接受恰好两个等号，且不能紧跟在另一个等号之后：
	// 这样 "===" / "====" 等连续等号不会误判成高亮语法。
	if node == nil || node.OriginalLength != 2 || before == '=' {
		return nil
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (p *highlightParser) CloseBlock(parent gast.Node, pc parser.Context) {
	// nothing to do
}

// highlightRenderer 把 Highlight 节点渲染为 <mark>。
type highlightRenderer struct{}

func (r *highlightRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindHighlight, r.renderHighlight)
}

func (r *highlightRenderer) renderHighlight(
	w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		if n.Attributes() != nil {
			_, _ = w.WriteString("<mark")
			html.RenderAttributes(w, n, html.GlobalAttributeFilter)
			_ = w.WriteByte('>')
		} else {
			_, _ = w.WriteString("<mark>")
		}
	} else {
		_, _ = w.WriteString("</mark>")
	}
	return gast.WalkContinue, nil
}

// highlightExt 是行内高亮的 goldmark 扩展。
type highlightExt struct{}

// NewHighlightExt 返回把 ==高亮内容== 渲染为 <mark> 的 goldmark 扩展。
func NewHighlightExt() goldmark.Extender {
	return &highlightExt{}
}

func (e *highlightExt) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(&highlightParser{}, 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&highlightRenderer{}, 500),
		),
	)
}
