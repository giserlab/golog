package entity

import "testing"

// TestBuildCommentTree 覆盖留言树的关键行为：
// 顶层留言保持原有顺序、回复挂到顶层留言下、回复的回复被压平到同一条线程，
// 以及父留言缺失时回复被提升为顶层（避免内容因审核状态而消失）。
func TestBuildCommentTree(t *testing.T) {
	comments := []*CommentR{
		{ID: "a", CreatedAt: 1},
		{ID: "b", ParentID: "a", CreatedAt: 2},
		{ID: "c", ParentID: "b", CreatedAt: 3}, // 回复的回复 → 压平到 a 下
		{ID: "d", CreatedAt: 4},
		{ID: "e", ParentID: "missing", CreatedAt: 5}, // 父留言缺失 → 提升为顶层
		{ID: "f", ParentID: "e", CreatedAt: 6},       // 挂在缺失父链的已有节点下
	}

	roots := BuildCommentTree(comments)
	gotRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		gotRoots = append(gotRoots, root.ID)
	}
	wantRoots := []string{"a", "d", "e"}
	if !equalStrings(gotRoots, wantRoots) {
		t.Fatalf("roots = %v, want %v", gotRoots, wantRoots)
	}

	repliesOf := func(id string) []string {
		for _, root := range roots {
			if root.ID != id {
				continue
			}
			ids := make([]string, 0, len(root.Replies))
			for _, reply := range root.Replies {
				ids = append(ids, reply.ID)
			}
			return ids
		}
		return nil
	}

	if got := repliesOf("a"); !equalStrings(got, []string{"b", "c"}) {
		t.Fatalf("replies of a = %v, want [b c]", got)
	}
	if got := repliesOf("d"); len(got) != 0 {
		t.Fatalf("replies of d = %v, want none", got)
	}
	if got := repliesOf("e"); !equalStrings(got, []string{"f"}) {
		t.Fatalf("replies of e = %v, want [f]", got)
	}
}

func TestBuildCommentTreeEmpty(t *testing.T) {
	if got := BuildCommentTree(nil); len(got) != 0 {
		t.Fatalf("BuildCommentTree(nil) = %v, want empty", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
