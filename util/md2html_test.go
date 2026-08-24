package util

import (
	"strings"
	"testing"
)

func TestMD2HTMLMermaid(t *testing.T) {
	md := "## 流程图\n\n```mermaid\ngraph TD;\n    A-->B;\n```\n"
	out := string(MD2HTML(md))

	// mermaid 容器必须保留
	if !strings.Contains(out, `class="mermaid"`) {
		t.Errorf("mermaid container missing: %s", out)
	}
	// 图内容必须保留（转义后的文本）
	if !strings.Contains(out, "graph TD;") {
		t.Errorf("mermaid code missing: %s", out)
	}
	// NoScript 模式下扩展不再注入 script（由主题模板惰性加载 mermaid.js），
	// 输出中不应出现任何 script 标签。
	if strings.Contains(out, "<script") {
		t.Errorf("mermaid output must not contain <script>: %s", out)
	}
}
