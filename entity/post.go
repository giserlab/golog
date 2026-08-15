package entity

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	stripmd "github.com/writeas/go-strip-markdown"
)

type Visibility string

const (
	VisibilityUnknown  Visibility = ""
	VisibilityPublic   Visibility = "public"
	VisibilityPrivate  Visibility = "private"
	VisibilityPassword Visibility = "password"
	VisibilityDraft    Visibility = "draft"
	VisibilityTrash    Visibility = "trash" // trash doesn't really exists, it's an abstraction

)

var (
	MonthSimp = map[string]string{}
)

func Init() {
	MonthSimp["o1"] = "Jan"

}

type PostCount struct {
	All      int
	NonTrash int
	Public   int
	Private  int
	Password int
	Draft    int
	Trash    int
}

type PostW struct {
	Type        string
	ID          string
	Title       string
	Slug        string
	Excerpt     string
	AuthorID    string
	Password    string
	Visibility  Visibility
	Content     string
	PinnedAt    int64
	PublishedAt int64
	CreatedAt   int64
	UpdatedAt   int64
	TrashedAt   int64

	TagIDs []string
}

type PostR struct {
	Type            string
	ID              string
	Title           string
	Slug            string
	OriginalExcerpt string
	AuthorID        string
	Password        string
	Visibility      Visibility
	Content         string
	PinnedAt        int64
	PublishedAt     int64
	CreatedAt       int64
	UpdatedAt       int64
	TrashedAt       int64

	Author UserR
	Tags   []*TagR
}

func (p *PostR) TagsStr() string {
	return strings.Join(p.TagNames(), ",")
}

func (p *PostR) TagNames() []string {
	var tags []string
	for _, tag := range p.Tags {
		tags = append(tags, tag.Name)
	}
	if len(tags) == 0 {
		return make([]string, 0)
	}
	return tags
}

func (p *PostR) PublishedDate() string {
	return p.publishedTime().Format("2006-01-02")
}

func (p *PostR) PublishedAtDatetime() string {
	return p.publishedTime().Format("2006-01-02 03:04 PM")
}

func (p *PostR) PublishedAtISO() string {
	return p.publishedTime().Format("2006-01-02T15:04")
}

// publishedTime 返回按站点配置时区偏移换算后的发布时间，
// 与归档分组（published_at + Timezone）口径一致。
func (p *PostR) publishedTime() time.Time {
	return time.Unix(p.PublishedAt+TimezoneOffset, 0).UTC()
}

func (p *PostR) PublishedYear() string {
	return p.publishedTime().Format("2006")
}

func (p *PostR) PublishedMonth() string {
	return p.publishedTime().Format("01")
}
func (p *PostR) PublishedMonthSimple() string {
	return p.publishedTime().Format("Jan")
}

func (p *PostR) PublishedDay() string {
	return p.publishedTime().Format("02")
}

func (p *PostR) IsPublished() bool {
	return time.Now().Unix() >= p.PublishedAt
}

func (p *PostR) Cover() string {
	ext := strings.Split("jpg,jpeg,png,JPG,JPEG,PNG", ",")
	for _, e := range ext {
		if _, err := os.Stat(fmt.Sprintf("data/uploads/covers/%s.%s", p.ID, e)); os.IsNotExist(err) {
			continue
		}
		return fmt.Sprintf("/uploads/covers/%s.%s", p.ID, e)
	}
	return ""
}

func (p *PostR) Excerpt() string {
	if p.OriginalExcerpt != "" {
		return p.OriginalExcerpt
	}
	content := stripmd.Strip(p.Content)
	if utf8.RuneCountInString(content) > 200 {
		return string([]rune(content)[:200]) + "..."
	}
	return content
}
