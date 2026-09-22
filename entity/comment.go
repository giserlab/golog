package entity

import "time"

type CommentW struct {
	ID          string
	PostID      string
	ParentID    string // 被回复的留言 ID；空字符串表示顶层留言
	AuthorName  string
	AuthorEmail string
	AuthorURL   string
	Content     string
	Status      string // pending | approved | rejected
	CreatedAt   int64
}

type CommentR struct {
	ID          string
	PostID      string
	ParentID    string
	AuthorName  string
	AuthorEmail string
	AuthorURL   string
	Content     string
	Status      string
	CreatedAt   int64
	// ParentAuthor 是父留言的作者名，由查询时 LEFT JOIN 填充，便于后台直接展示
	// “回复 @某某”而不必再查一次。顶层留言或父留言已删除时为空字符串。
	ParentAuthor string
}

func (c *CommentR) CreatedDate() string {
	return time.Unix(c.CreatedAt+TimezoneOffset, 0).UTC().Format("2006-01-02 15:04")
}

// CommentNode 是留言树的一个节点：顶层留言携带其全部回复。
// 模板通过嵌入字段直接访问 CommentR 的字段（{{ .AuthorName }} 等），
// 再用 {{ range .Replies }} 渲染同一层级下的回复。
type CommentNode struct {
	*CommentR
	Replies []*CommentNode
}

// BuildCommentTree 把扁平留言列表整理为“顶层留言 + 一级回复”的树。
//
// 设计取舍：回复只展示一层。回复的回复会挂到同一条顶层留言下，避免无限嵌套
// 在小屏幕上难以阅读；界面上通过“回复 @某某”保留对话上下文。
// 父留言不在列表里（待审核、已驳回或已删除）的回复会提升为顶层留言，
// 保证内容不会因为审核状态而凭空消失。
func BuildCommentTree(comments []*CommentR) []*CommentNode {
	nodes := make(map[string]*CommentNode, len(comments))
	order := make([]*CommentNode, 0, len(comments))
	for _, c := range comments {
		node := &CommentNode{CommentR: c}
		nodes[c.ID] = node
		order = append(order, node)
	}

	roots := make([]*CommentNode, 0, len(order))
	for _, node := range order {
		root := rootNode(node, nodes)
		if root == nil || root == node {
			roots = append(roots, node)
			continue
		}
		root.Replies = append(root.Replies, node)
	}
	return roots
}

// rootNode 沿 parent_id 向上找到可达的最高层节点。
// 父留言缺失（待审核、已驳回或已删除）时停在当前节点，这样它的下级回复
// 仍然会挂在这条链上，而不会各自变成互不相关的顶层留言。
// 出现环（数据异常）时返回 nil，由调用方按顶层处理。
func rootNode(node *CommentNode, nodes map[string]*CommentNode) *CommentNode {
	current := node
	for i := 0; i <= len(nodes); i++ {
		if current.ParentID == "" {
			return current
		}
		parent, ok := nodes[current.ParentID]
		if !ok {
			return current
		}
		current = parent
	}
	return nil
}
