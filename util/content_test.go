package util

import "testing"

func TestFirstImage(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "markdown image",
			content: "正文开始\n\n![封面](/uploads/images/a.jpg)\n\n后续内容",
			want:    "/uploads/images/a.jpg",
		},
		{
			name:    "markdown image with title",
			content: `![截图](/uploads/images/b.png "说明文字")`,
			want:    "/uploads/images/b.png",
		},
		{
			name:    "markdown image with angle brackets",
			content: "![带空格](</uploads/images/my file.png>)",
			want:    "/uploads/images/my file.png",
		},
		{
			name:    "html img tag",
			content: `<p>前言</p><img src="/uploads/images/c.webp" alt="c">`,
			want:    "/uploads/images/c.webp",
		},
		{
			name:    "first of many",
			content: "![](/first.jpg) 和 ![](/second.jpg)",
			want:    "/first.jpg",
		},
		{
			name:    "no image",
			content: "只有文字，没有任何图片。",
			want:    "",
		},
		{
			name:    "link is not an image",
			content: "[文档](/docs/index.html)",
			want:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FirstImage(tc.content); got != tc.want {
				t.Errorf("FirstImage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPlainTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"## 便利蜂是怎样选址的？", "便利蜂是怎样选址的？"},
		{"**加粗**标题", "加粗标题"},
		{"[链接标题](/post/x)", "链接标题"},
		{"  前后空格  ", "前后空格"},
	}
	for _, tc := range cases {
		if got := PlainTitle(tc.in); got != tc.want {
			t.Errorf("PlainTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestReadingMinutes(t *testing.T) {
	cjk := make([]rune, 0, 800)
	for i := 0; i < 800; i++ {
		cjk = append(cjk, '测')
	}

	cases := []struct {
		name    string
		content string
		want    int
	}{
		{name: "empty content still takes a minute", content: "", want: 1},
		{name: "short note", content: "今天发布了新版本。", want: 1},
		{name: "800 chinese chars", content: string(cjk), want: 2},
		{name: "english words", content: repeatWord("word", 400), want: 2},
		{name: "code blocks are skipped", content: "```go\n" + repeatWord("fmt.Println(1)", 900) + "\n```", want: 1},
		{name: "images add reading time", content: "![](/a.jpg)![](/b.jpg)![](/c.jpg)![](/d.jpg)![](/e.jpg)![](/f.jpg)", want: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ReadingMinutes(tc.content); got != tc.want {
				t.Errorf("ReadingMinutes() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCountImages(t *testing.T) {
	content := "![](/a.jpg)\n\n<img src=\"/b.png\">\n\n![图片](/c.webp \"标题\")"
	if got := CountImages(content); got != 3 {
		t.Errorf("CountImages() = %d, want 3", got)
	}
}

func repeatWord(word string, times int) string {
	out := ""
	for i := 0; i < times; i++ {
		out += word + " "
	}
	return out
}
