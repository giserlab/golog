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
	Tag        string
	Author     string
	AuthorUser *entity.UserR
	Date       string
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
		Type:        "",
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
// ArchiveView
// ===============================

func ArchiveView(c *gin.Context) {
	self, err := self(c)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var routes = []entity.Route{
		{
			Name: "首页",
			Path: "/",
		},
		{
			Name: "随笔",
			Path: "",
		},
	}
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
	if err := system.PostTmpl.Execute(&tpl, data(c, gin.H{
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
// AuthorView
// ===============================

func AuthorView(c *gin.Context) {
	self, err := self(c)
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var routes = []entity.Route{
		{Name: "首页", Path: "/"},
	}
	var (
		page         = queryPage(c)
		countPerPage = system.Config.PostsPerPage
	)
	user, err := store.GetUser(c.Param("author"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	routes = append(routes, entity.Route{
		Name: user.Nickname,
		Path: "",
	})
	q := &store.ListPostsQuery{
		AuthorID:    user.ID,
		Type:        util.BlogType,
		Offset:      (page - 1) * countPerPage,
		Limit:       countPerPage,
		IsPublished: store.PtrBool(true),
		IsTrashed:   store.PtrBool(false),
	}
	if self == nil {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword}
	} else {
		q.Visibilities = []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword, entity.VisibilityPrivate}
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
	if err := system.AuthorTmpl.Execute(&tpl, data(c, gin.H{
		"AuthorUser":  user,
		"Posts":       posts,
		"Routes":      routes,
		"Pagination":  pagination(c, page, count, countPerPage),
		"Navigations": navs,
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
	// 草稿（draft）在任何情况下都不能通过公开路由查看（登录与否都不行），
	// 只能通过作者预览路由 /preview/:slug 访问。
	if p.Visibility == entity.VisibilityDraft {
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
	renderSingular(c, p, isUnlocked)
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
	// 草稿（draft）在任何情况下都不能通过公开路由查看（登录与否都不行），
	// 只能通过作者预览路由 /preview/:slug 访问。
	if p.Visibility == entity.VisibilityDraft {
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
	renderSingular(c, p, isUnlocked)
}

// ===============================
// SingularPreviewView
// ===============================

// SingularPreviewView 是后台"查看文章"使用的作者预览路由（/preview/:slug）。
// 与公开路由 /post/:slug 分离：路由本身已由 checkLoggedIn 强制登录鉴权，
// 这里再按后台文章隔离规则限制只有作者本人或管理员可预览，
// 因此草稿、私密、定时等未公开文章也能以解锁状态查看。
func SingularPreviewView(c *gin.Context) {
	uid := userID(c)
	p, err := store.GetPostBySlug(c.Param("slug"))
	if err != nil && !store.IsNotFound(err) {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if p == nil {
		noRoute(c)
		return
	}
	if !isCurrentUserAdmin(c) && p.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	renderSingular(c, p, true)
}

// renderSingular 渲染公开文章页模板（system.SingularTmpl）。
// 调用方负责可见性与鉴权检查（公开路由的可见性/定时判断，或预览路由的登录/作者校验）。
func renderSingular(c *gin.Context, p *entity.PostR, isUnlocked bool) {
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
		query.AuthorUser = user
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
