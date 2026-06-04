package util

import (
	"fmt"
	"strconv"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// customFootnoteExt 保留原始脚注标签，并添加 data-* 属性供前端交互使用。
type customFootnoteExt struct{}

func NewCustomFootnoteExt() goldmark.Extender {
	return &customFootnoteExt{}
}

func (e *customFootnoteExt) Extend(m goldmark.Markdown) {
	// 复用 goldmark 默认的 footnote 解析器与 AST 转换器
	extension.Footnote.Extend(m)

	// 在默认 transformer（优先级 999）之后收集标签映射
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&footnoteLabelTransformer{}, 1000),
		),
	)

	// 覆盖默认 footnote HTML 渲染器（默认 500）。
	// goldmark renderer 按升序排序后从尾部遍历注册，后注册者覆盖前者。
	// 因此数值需小于 500 才能确保后注册。
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&customFootnoteRenderer{}, 0),
		),
	)
}

// footnoteLabelTransformer 遍历 AST，收集 Index → Ref 的映射并存入 Document 属性。
type footnoteLabelTransformer struct{}

func (t *footnoteLabelTransformer) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	indexToRef := make(map[int]string)
	gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		if fn, ok := n.(*extast.Footnote); ok {
			indexToRef[fn.Index] = string(fn.Ref)
		}
		return gast.WalkContinue, nil
	})
	doc.SetAttributeString("footnote-labels", indexToRef)
}

// customFootnoteRenderer 覆盖默认脚注渲染，保留原始标签文本。
type customFootnoteRenderer struct{}

func (r *customFootnoteRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extast.KindFootnoteLink, r.renderFootnoteLink)
	reg.Register(extast.KindFootnoteBacklink, r.renderFootnoteBacklink)
	reg.Register(extast.KindFootnote, r.renderFootnote)
	reg.Register(extast.KindFootnoteList, r.renderFootnoteList)
}

func (r *customFootnoteRenderer) renderFootnoteLink(
	w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*extast.FootnoteLink)
	is := strconv.Itoa(n.Index)
	ref := r.getRef(node, n.Index)
	if ref == "" {
		ref = is
	}

	_, _ = w.WriteString(`<sup id="fnref`)
	if n.RefIndex > 0 {
		_, _ = fmt.Fprintf(w, "%d", n.RefIndex)
	}
	_, _ = w.WriteString(`:`)
	_, _ = w.WriteString(is)
	_, _ = w.WriteString(`"><a href="#fn:`)
	_, _ = w.WriteString(is)
	_, _ = w.WriteString(`" class="footnote-ref" role="doc-noteref" data-index="`)
	_, _ = w.WriteString(is)
	_, _ = w.WriteString(`" data-ref="`)
	_, _ = w.WriteString(string(util.EscapeHTML([]byte(ref))))
	_, _ = w.WriteString(`">`)
	_, _ = w.WriteString(string(util.EscapeHTML([]byte(ref))))
	_, _ = w.WriteString(`</a></sup>`)
	return gast.WalkContinue, nil
}

func (r *customFootnoteRenderer) renderFootnoteBacklink(
	w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*extast.FootnoteBacklink)
	is := strconv.Itoa(n.Index)
	_, _ = w.WriteString(`&#160;<a href="#fnref`)
	if n.RefIndex > 0 {
		_, _ = fmt.Fprintf(w, "%d", n.RefIndex)
	}
	_, _ = w.WriteString(`:`)
	_, _ = w.WriteString(is)
	_, _ = w.WriteString(`" class="footnote-backref" role="doc-backlink">`)
	_, _ = w.WriteString(`↩`)
	_, _ = w.WriteString(`</a>`)
	return gast.WalkContinue, nil
}

func (r *customFootnoteRenderer) renderFootnote(
	w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*extast.Footnote)
	is := strconv.Itoa(n.Index)
	ref := r.getRef(node, n.Index)
	if entering {
		_, _ = w.WriteString(`<li id="fn:`)
		_, _ = w.WriteString(is)
		_, _ = w.WriteString(`" data-ref="`)
		_, _ = w.WriteString(string(util.EscapeHTML([]byte(ref))))
		_, _ = w.WriteString(`"`)
		if node.Attributes() != nil {
			html.RenderAttributes(w, node, html.ListItemAttributeFilter)
		}
		_, _ = w.WriteString(">\n")
	} else {
		_, _ = w.WriteString("</li>\n")
	}
	return gast.WalkContinue, nil
}

func (r *customFootnoteRenderer) renderFootnoteList(
	w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="footnotes" role="doc-endnotes"`)
		if node.Attributes() != nil {
			html.RenderAttributes(w, node, html.GlobalAttributeFilter)
		}
		_, _ = w.WriteString(">\n")
		_, _ = w.WriteString("<hr>\n")
		_, _ = w.WriteString("<ol>\n")
	} else {
		_, _ = w.WriteString("</ol>\n")
		_, _ = w.WriteString("</div>\n")
	}
	return gast.WalkContinue, nil
}

func (r *customFootnoteRenderer) getRef(n gast.Node, index int) string {
	doc := n.OwnerDocument()
	if doc == nil {
		return ""
	}
	if v, ok := doc.AttributeString("footnote-labels"); ok {
		if refs, ok := v.(map[int]string); ok {
			return refs[index]
		}
	}
	return ""
}
