package handler

import (
	"bytes"
	"net/http"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"
	"golog/util"

	"github.com/gin-gonic/gin"
)

// ===============================
// IndexView
// ===============================

type IndexQuery struct {
	Tag    string
	Author string
	Date   string
}

func (q *IndexQuery) IsEmpty() bool {
	return q.Tag == "" && q.Author == "" && q.Date == ""
}

func IndexView(c *gin.Context) {
	self, err := self(c)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var routes = []entity.Route{}
	routes = append(routes, entity.Route{
		Name: "首页",
		Path: "/",
	})
	var (
		page         = queryPage(c)
		countPerPage = system.Config.PostsPerPage
		query        = &IndexQuery{}
	)
	q := &store.ListPostsQuery{
		Type:        util.BlogType,
		Offset:      (page - 1) * countPerPage,
		Limit:       countPerPage,
		Title:       c.Query("title"),
		IsPublished: store.PtrBool(true),
		IsTrashed:   store.PtrBool(false),
	}
	if self == nil {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword}
	} else {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword, entity.VisibilityPrivate}
	}
	// tag
	if v := c.Param("tag"); v != "" {
		tag, err := store.GetTagBySlug(v)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		q.TagID = tag.ID
		query.Tag = tag.Name
		routes = append(routes, entity.Route{
			Name: "标签",
			Path: "/tag/" + query.Tag,
		})
		routes = append(routes, entity.Route{
			Name: tag.Name,
			Path: "",
		})
		routes[0].Path = "/"
	}
	// author
	if v := c.Param("author"); v != "" {
		user, err := store.GetUser(v)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		q.AuthorID = user.ID
		query.Author = user.Nickname
	}
	// dates
	if y := c.Param("year"); y != "" {
		q.PublishedYear = y
		query.Date = y
		if m := c.Param("month"); m != "" {
			q.PublishedMonth = m
			query.Date += "/" + m

			if d := c.Param("day"); d != "" {
				q.PublishedDay = d
				query.Date += "/" + d
			}
		}
		routes = append(routes, entity.Route{
			Name: "归档",
			Path: "/archive/" + query.Date,
		})
		routes = append(routes, entity.Route{
			Name: c.Param("year") + "-" + c.Param("month"),
			Path: "",
		})
		routes[0].Path = "/"
	}
	posts, err := store.ListPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	count, err := store.CountPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var tpl bytes.Buffer
	if err := system.IndexTmpl.Execute(&tpl, data(c, gin.H{
		"Posts":       posts,
		"Routes":      routes,
		"Search":      q.Title,
		"Pagination":  pagination(c, page, count, countPerPage),
		"Navigations": navs,
		"Filter":      query,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
}

// ===============================
// SingularView
// ===============================

func SingularView(c *gin.Context) {
	self, err := self(c)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	p, err := store.GetPostBySlug(c.Param("slug"))
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if p == nil {
		noRoute(c)
		return
	}
	if self == nil && p.Visibility != entity.VisibilityPublic && p.Visibility != entity.VisibilityPassword {
		noRoute(c)
		return
	}
	if self == nil && p.PublishedAt > time.Now().Unix() {
		noRoute(c)
		return
	}
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	comments, err := store.ListCommentsByPost(p.ID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	prevPost, err := store.GetPreviousPost(p.ID)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	nextPost, err := store.GetNextPost(p.ID)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var isUnlocked bool
	if self != nil || p.Visibility == entity.VisibilityPublic {
		isUnlocked = true
	} else {
		if c.PostForm("password") == p.Password {
			isUnlocked = true
		} else if c.Request.Method == http.MethodPost {
			setMessage(c, "notice_post_incorrect")
		}
	}
	var routes = []entity.Route{}
	routes = append(routes, entity.Route{
		Name: "首页",
		Path: "/",
	})
	routes = append(routes, entity.Route{
		Name: "随笔",
		Path: "/",
	})
	routes = append(routes, entity.Route{
		Name: p.Slug,
		Path: "",
	})
	var tpl bytes.Buffer
	if err := system.SingularTmpl.Execute(&tpl, data(c, gin.H{
		"Post":         p,
		"Navigations":  navs,
		"Routes":       routes,
		"PreviousPost": prevPost,
		"NextPost":     nextPost,
		"IsUnlocked":   isUnlocked,
		"Comments":     comments,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
}

func SingularViewByID(c *gin.Context) {
	self, err := self(c)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	p, err := store.GetPostByID(c.Param("id"))
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if p == nil {
		noRoute(c)
		return
	}
	if self == nil && p.Visibility != entity.VisibilityPublic && p.Visibility != entity.VisibilityPassword {
		noRoute(c)
		return
	}
	if self == nil && p.PublishedAt > time.Now().Unix() {
		noRoute(c)
		return
	}
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	comments, err := store.ListCommentsByPost(p.ID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	prevPost, err := store.GetPreviousPost(p.ID)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	nextPost, err := store.GetNextPost(p.ID)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var isUnlocked bool
	if self != nil || p.Visibility == entity.VisibilityPublic {
		isUnlocked = true
	} else {
		if c.PostForm("password") == p.Password {
			isUnlocked = true
		} else if c.Request.Method == http.MethodPost {
			setMessage(c, "notice_post_incorrect")
		}
	}
	var routes = []entity.Route{}
	routes = append(routes, entity.Route{
		Name: "首页",
		Path: "/",
	})
	routes = append(routes, entity.Route{
		Name: "随笔",
		Path: "/",
	})
	routes = append(routes, entity.Route{
		Name: p.Slug,
		Path: "",
	})
	var tpl bytes.Buffer
	if err := system.SingularTmpl.Execute(&tpl, data(c, gin.H{
		"Post":         p,
		"Navigations":  navs,
		"Routes":       routes,
		"PreviousPost": prevPost,
		"NextPost":     nextPost,
		"IsUnlocked":   isUnlocked,
		"Comments":     comments,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
}

func MomentView(c *gin.Context) {
	self, err := self(c)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var routes = []entity.Route{}
	routes = append(routes, entity.Route{
		Name: "首页",
		Path: "/",
	})
	var (
		page         = queryPage(c)
		countPerPage = system.Config.PostsPerPage
		query        = &IndexQuery{}
	)
	q := &store.ListPostsQuery{
		Type:        util.MomentType,
		Offset:      (page - 1) * countPerPage,
		Limit:       countPerPage,
		Title:       c.Query("title"),
		IsPublished: store.PtrBool(true),
		IsTrashed:   store.PtrBool(false),
	}
	if self == nil {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword}
	} else {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword, entity.VisibilityPrivate}
	}
	// dates
	if y := c.Param("year"); y != "" {
		q.PublishedYear = y
		query.Date = y
		if m := c.Param("month"); m != "" {
			q.PublishedMonth = m
			query.Date += "/" + m

			if d := c.Param("day"); d != "" {
				q.PublishedDay = d
				query.Date += "/" + d
			}
		}
		routes = append(routes, entity.Route{
			Name: "时刻",
			Path: "/moment",
		})
		routes = append(routes, entity.Route{
			Name: c.Param("year"),
			Path: "",
		})
		routes[0].Path = "/"
	}
	routes[0].Path = "/"
	routes = append(routes, entity.Route{
		Name: "时刻",
		Path: "",
	})
	posts, err := store.ListPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	count, err := store.CountPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var tpl bytes.Buffer
	if err := system.MomentTmpl.Execute(&tpl, data(c, gin.H{
		"Posts":       posts,
		"Routes":      routes,
		"Pagination":  pagination(c, page, count, countPerPage),
		"Navigations": navs,
		"Filter":      query,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
}

func WhisperView(c *gin.Context) {
	self, err := self(c)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var routes = []entity.Route{}
	routes = append(routes, entity.Route{
		Name: "首页",
		Path: "/",
	})
	var (
		page         = queryPage(c)
		countPerPage = system.Config.PostsPerPage
		query        = &IndexQuery{}
	)
	q := &store.ListPostsQuery{
		Type:        util.WhisperType,
		Offset:      (page - 1) * countPerPage,
		Limit:       countPerPage,
		Title:       c.Query("title"),
		IsPublished: store.PtrBool(true),
		IsTrashed:   store.PtrBool(false),
	}
	if self == nil {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword}
	} else {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword, entity.VisibilityPrivate}
	}
	// tag
	if v := c.Param("tag"); v != "" {
		tag, err := store.GetTagBySlug(v)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		q.TagID = tag.ID
		query.Tag = tag.Name
		routes = append(routes, entity.Route{
			Name: "标签",
			Path: "/tag/" + query.Tag,
		})
		routes = append(routes, entity.Route{
			Name: tag.Name,
			Path: "",
		})
		routes[0].Path = "/"
	}
	// author
	if v := c.Param("author"); v != "" {
		user, err := store.GetUser(v)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		q.AuthorID = user.ID
		query.Author = user.Nickname
	}
	// dates
	if y := c.Param("year"); y != "" {
		q.PublishedYear = y
		query.Date = y
		if m := c.Param("month"); m != "" {
			q.PublishedMonth = m
			query.Date += "/" + m

			if d := c.Param("day"); d != "" {
				q.PublishedDay = d
				query.Date += "/" + d
			}
		}
		routes = append(routes, entity.Route{
			Name: "归档",
			Path: "/archive/" + query.Date,
		})
		routes = append(routes, entity.Route{
			Name: c.Param("year") + "-" + c.Param("month"),
			Path: "",
		})
		routes[0].Path = "/"
	}
	routes[0].Path = "/"
	routes = append(routes, entity.Route{
		Name: "日志",
		Path: "",
	})
	posts, err := store.ListPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	count, err := store.CountPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var tpl bytes.Buffer
	if err := system.WhisperTmpl.Execute(&tpl, data(c, gin.H{
		"Posts":       posts,
		"Routes":      routes,
		"Pagination":  pagination(c, page, count, countPerPage),
		"Navigations": navs,
		"Filter":      query,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
}
