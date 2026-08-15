package handler

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"golog/entity"
	"golog/store"
	"golog/util"

	"github.com/gin-gonic/gin"
)

// ===============================
// SiteMapView
// ===============================

func SiteMapView(c *gin.Context) {
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
	type URL struct {
		Loc     string `xml:"loc"`
		Lastmod string `xml:"lastmod"`
	}
	type URLSET struct {
		XMLName xml.Name `xml:"urlset"`
		Xmlns   string   `xml:"xmlns,attr"`
		URLs    []URL    `xml:"url"`
	}
	var urls []URL
	for _, post := range posts {
		urls = append(urls, URL{
			Loc:     sitemapURL(c, post),
			Lastmod: time.Unix(post.UpdatedAt, 0).Format("2006-01-02"),
		})
	}
	c.XML(http.StatusOK, URLSET{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	})
}

func sitemapURL(c *gin.Context, post *entity.PostR) string {
	root := requestScheme(c) + "://" + c.Request.Host
	return fmt.Sprintf("%s/post/%s", root, post.Slug)
}
func sitemapGUID(c *gin.Context, post *entity.PostR) string {
	root := requestScheme(c) + "://" + c.Request.Host
	return fmt.Sprintf("%s/blog/%s", root, post.ID)
}
