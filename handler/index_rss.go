package handler

import (
	"encoding/xml"
	"net/http"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"
	"golog/util"

	"github.com/gin-gonic/gin"
)

// ===============================
// RSSView
// ===============================

func RSSView(c *gin.Context) {
	q := &store.ListPostsQuery{
		Type:         util.BlogType,
		IsPublished:  store.PtrBool(true),
		IsTrashed:    store.PtrBool(false),
		Visibilities: []entity.Visibility{entity.VisibilityPublic},
	}
	posts, err := store.ListPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	type Item struct {
		Guid        string `xml:"guid"`
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		Author      string `xml:"author"`
		PubDate     string `xml:"pubDate"`
	}
	type AtomLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
		Type string `xml:"type,attr"`
	}

	type Channel struct {
		XMLName     xml.Name `xml:"channel"`
		AtomLink    AtomLink `xml:"atom:link"`
		Title       string   `xml:"title"`
		Link        string   `xml:"link"`
		Description string   `xml:"description"`
		Language    string   `xml:"language"`
		PubDate     string   `xml:"pubDate"`
		Items       []Item   `xml:"item"`
	}
	var items []Item
	for _, post := range posts {
		item := Item{
			Guid:    sitemapGUID(c, post),
			Title:   post.Title,
			Author:  "WSH",
			Link:    sitemapURL(c, post),
			PubDate: time.Unix(post.UpdatedAt, 0).Format(time.RFC1123Z),
		}

		totalLen := 120
		if len(post.Content) > 0 {
			runes := []rune(post.Content)
			if len(runes) > totalLen {
				item.Description = string(runes[:totalLen]) + "..."
			} else {
				item.Description = post.Content
			}
		}
		items = append(items, item)
	}
	suffix := requestScheme(c) + "://"
	root := suffix + c.Request.Host
	type Rss struct {
		XMLName   xml.Name `xml:"rss"`
		Version   string   `xml:"version,attr"`
		XMLNSAtom string   `xml:"xmlns:atom,attr"`
		Channel   Channel  `xml:"channel"`
	}
	c.XML(http.StatusOK, Rss{
		Version:   "2.0",
		XMLNSAtom: "http://www.w3.org/2005/Atom",
		Channel: Channel{
			AtomLink: AtomLink{
				Href: root + "/rss.xml",
				Rel:  "self",
				Type: "application/rss+xml",
			},
			Title:       system.Config.Name,
			Link:        root,
			Description: system.Config.Description,
			Language:    system.Config.Locale,
			PubDate:     time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	})

}
