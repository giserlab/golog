package util

import (
	"regexp"
	"strings"
	"unicode"
)

// 阅读时长估算参数：中文按每分钟 400 字，拉丁文按每分钟 200 词。
// 图片会额外占用浏览时间（每张 10 秒），这样图表密集的文章不会被低估。
const (
	readingCharsPerMinute = 400
	readingWordsPerMinute = 200
	readingSecondsPerImg  = 10
)

var (
	// markdownImageRegExp 匹配 Markdown 行内图片语法，兼容可选的 title 部分。
	markdownImageRegExp = regexp.MustCompile(`!\[[^\]]*\]\(\s*(?:<([^>]+)>|([^\s)]+))(?:\s+["'][^"']*["'])?\s*\)`)
	// htmlImageRegExp 匹配正文中直接书写的 <img src="...">。
	htmlImageRegExp = regexp.MustCompile(`(?i)<img[^>]*\ssrc\s*=\s*["']([^"']+)["']`)
	// htmlTagRegExp 匹配任意 HTML 标签，用于统计正文可见文字。
	htmlTagRegExp = regexp.MustCompile(`(?s)<[^>]*>`)
	// markdownLinkRegExp 保留链接文字、丢弃链接地址。
	markdownLinkRegExp = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	// markdownMarkRegExp 去除 Markdown 的强调、代码、引用等标记符号。
	markdownMarkRegExp = regexp.MustCompile("[*_~`>#|]+")
	// fencedCodeRegExp 匹配围栏代码块（```lang ... ```）。
	fencedCodeRegExp = regexp.MustCompile("(?s)```.*?```")
	// inlineCodeRegExp 匹配行内代码。
	inlineCodeRegExp = regexp.MustCompile("`[^`]*`")
	// headingIDRegExp 匹配标题尾部的锚点标记 {#id}。
	headingIDRegExp = regexp.MustCompile(`\s*\{#[^}]*\}\s*$`)
)

// PlainTitle 去除标题中的 Markdown 标记，返回适合放入 alt、title 等属性的纯文本。
func PlainTitle(v string) string {
	s := strings.TrimSpace(v)
	s = strings.TrimLeft(s, "#")
	s = markdownLinkRegExp.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("*", "", "_", "", "`", "", "~", "", "=", "", "<", "", ">", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// FirstImage 返回 Markdown 正文中的第一张图片地址。
// 支持 Markdown 图片语法与直接书写的 <img> 标签；使用 srcset 的图片会被跳过。
// 未找到图片时返回空字符串。
func FirstImage(content string) string {
	if m := markdownImageRegExp.FindStringSubmatch(content); m != nil {
		src := m[1]
		if src == "" {
			src = m[2]
		}
		if src = strings.TrimSpace(src); src != "" {
			return src
		}
	}
	if m := htmlImageRegExp.FindStringSubmatch(content); m != nil {
		if src := strings.TrimSpace(m[1]); src != "" {
			return src
		}
	}
	return ""
}

// ReadingMinutes 估算阅读时长（分钟），最少返回 1 分钟。
// 中文按每分钟 400 字、拉丁文按每分钟 200 词计算，图片每张额外计入 10 秒。
func ReadingMinutes(content string) int {
	var (
		cjk   int
		words int
		inWrd bool
	)
	for _, r := range stripMarkdownForCount(content) {
		switch {
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r):
			cjk++
			inWrd = false
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			if !inWrd {
				words++
				inWrd = true
			}
		default:
			inWrd = false
		}
	}

	minutes := float64(cjk)/readingCharsPerMinute + float64(words)/readingWordsPerMinute
	minutes += float64(CountImages(content)) * readingSecondsPerImg / 60
	if minutes < 1 {
		return 1
	}
	return int(minutes + 0.5)
}

// CountImages 统计 Markdown 正文中的图片数量。
func CountImages(content string) int {
	n := len(markdownImageRegExp.FindAllString(content, -1))
	// 已计入 Markdown 语法的图片不会再被 <img> 规则重复匹配，
	// 但为稳妥起见仍去掉这部分文本再统计 HTML 图片。
	rest := markdownImageRegExp.ReplaceAllString(content, "")
	return n + len(htmlImageRegExp.FindAllString(rest, -1))
}

// stripMarkdownForCount 去除 Markdown 语法与代码块，返回用于字数统计的纯文本。
func stripMarkdownForCount(content string) string {
	s := content
	s = fencedCodeRegExp.ReplaceAllString(s, " ")
	s = inlineCodeRegExp.ReplaceAllString(s, " ")
	// 图片替换为占位空格（图片数量由 CountImages 单独统计）。
	s = markdownImageRegExp.ReplaceAllString(s, " ")
	s = htmlImageRegExp.ReplaceAllString(s, " ")
	s = htmlTagRegExp.ReplaceAllString(s, " ")
	// 链接与加粗等标记只保留文字部分。
	s = markdownLinkRegExp.ReplaceAllString(s, "$1")
	s = markdownMarkRegExp.ReplaceAllString(s, " ")
	s = headingIDRegExp.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
