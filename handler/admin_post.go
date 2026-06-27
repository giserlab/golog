package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"golog/entity"
	"golog/store"
	"golog/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===============================
// PostsView
// ===============================

func PostsView(c *gin.Context) {
	uid := userID(c)
	var (
		page         = queryPage(c)
		countPerPage = 30
		visibility   = entity.Visibility(c.Query("visibility"))
	)
	postType := c.Query("type")
	if postType == "" {
		postType = util.BlogType
	}
	q := &store.ListPostsQuery{
		Type:          postType,
		Offset:        (page - 1) * countPerPage,
		Limit:         countPerPage,
		Title:         c.Query("title"),
		AuthorID:      uid,
		Visibilities:  []entity.Visibility{entity.VisibilityPublic, entity.VisibilityPassword, entity.VisibilityPrivate, entity.VisibilityDraft},
		IsTrashed:     store.PtrBool(false),
		PublishedDate: c.Query("published_date"),
	}
	if visibility != "" && visibility != "trash" {
		q.Visibilities = []entity.Visibility{entity.Visibility(c.Query("visibility"))}
		q.IsTrashed = store.PtrBool(false)
	}
	if visibility == entity.VisibilityTrash {
		q.IsTrashed = store.PtrBool(true)
	}
	posts, err := store.ListallPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	count, err := store.CountPosts(q)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	counts, err := store.CountPostsByTypeAndUser(q.Type, uid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	dates, err := store.ListPostDatesByUser(uid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.HTML(http.StatusOK, "admin_posts", data(c, gin.H{
		"Query":         q,
		"IsQuerySetted": q.Title != "" || q.PublishedDate != "",
		"Posts":         posts,
		"Dates":         dates,
		"PostCount":     counts,
		"Visibility":    visibility,
		"Pagination":    pagination(c, page, count, countPerPage),
	}))
}

// ===============================
// PostCreate
// ===============================

func PostCreateView(c *gin.Context) {
	tags, err := store.ListTags(0, 999, "")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	mostUsedTags, err := store.ListMostUsedTags()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.HTML(http.StatusOK, "admin_post_create", data(c, gin.H{
		"Tags":         tags,
		"MostUsedTags": mostUsedTags,
		"Post":         &entity.PostR{Type: util.BlogType},
	}))
}

// ===============================
// PostCreate
// ===============================

type PostCreateRequest struct {
	Type        string            `form:"type" binding:"required" conform:"trim"`
	Title       string            `form:"title" binding:"required,max=128" conform:"trim"`
	Slug        string            `form:"slug" binding:"required" conform:"trim"`
	Excerpt     string            `form:"excerpt" conform:"trim"`
	Password    string            `form:"password" binding:"max=128" conform:"trim"`
	Visibility  entity.Visibility `form:"visibility" binding:"required,oneof=public private password draft"`
	Content     string            `form:"content" conform:"trim"`
	PublishedAt int64             `form:"published_at"`
	IsPinned    bool              `form:"is_pinned"`
	Tags        string            `form:"tags"`
}

func PostCreate(c *gin.Context, req *PostCreateRequest) {
	pid := uuid.New().String()
	uid := userID(c)

	if _, err := saveCover(c, pid); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ids, err := createTags(req.Tags)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	p := &entity.PostW{
		ID:          pid,
		Type:        req.Type,
		Title:       req.Title,
		Slug:        toSlug(req.Slug),
		Excerpt:     req.Excerpt,
		AuthorID:    uid,
		Password:    "",
		Visibility:  req.Visibility,
		Content:     req.Content,
		PublishedAt: req.PublishedAt,
		TagIDs:      ids,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}
	if req.PublishedAt == 0 {
		p.PublishedAt = time.Now().Unix()
	}
	if req.IsPinned {
		p.PinnedAt = time.Now().Unix()
	}
	if req.Visibility == entity.VisibilityPassword {
		p.Password = req.Password
	}
	if err := store.CreatePost(p); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_post_created")
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("../post/%s", pid))
}

// ===============================
// PostEditView
// ===============================

type PostEditViewObject struct {
	Type              string            `json:"type"`
	Visibility        entity.Visibility `json:"visibility"`
	CoverImageURL     string            `json:"cover_image_url"`
	Tags              []string          `json:"tags"`
	TagsStr           string            `json:"tags_str"`
	Slug              string            `json:"slug"`
	TagInputValue     string            `json:"tag_input_value"`
	PublishedDateTime string            `json:"published_datetime"`
	PublishedAt       string            `json:"published_at"`
	IsClearCover      bool              `json:"is_clear_cover"`
}

func PostEditView(c *gin.Context) {
	uid := userID(c)
	post, err := store.GetPost(c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	tags, err := store.ListTags(0, 999, "")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	jsonData, err := json.Marshal(&PostEditViewObject{
		Type:              post.Type,
		Visibility:        post.Visibility,
		CoverImageURL:     post.Cover(),
		Tags:              post.TagNames(),
		TagsStr:           post.TagsStr(),
		Slug:              post.Slug,
		TagInputValue:     "",
		PublishedDateTime: post.PublishedAtISO(),
		PublishedAt:       strconv.Itoa(int(post.PublishedAt)),
		IsClearCover:      false,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	mostUsedTags, err := store.ListMostUsedTags()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.HTML(http.StatusOK, "admin_post_edit", data(c, gin.H{
		"Tags":         tags,
		"Post":         post,
		"MostUsedTags": mostUsedTags,
		"JSONData":     string(jsonData),
	}))
}

// ===============================
// PostEdit
// ===============================

type PostEditRequest struct {
	Type         string            `form:"type" binding:"required" conform:"trim"`
	Title        string            `form:"title" binding:"required,max=128" conform:"trim"`
	Slug         string            `form:"slug" binding:"required" conform:"trim"`
	Excerpt      string            `form:"excerpt" conform:"trim"`
	Password     string            `form:"password" binding:"max=128" conform:"trim"`
	Visibility   entity.Visibility `form:"visibility" binding:"required,oneof=public private password draft"`
	Content      string            `form:"content" conform:"trim"`
	PublishedAt  int64             `form:"published_at" binding:"required"`
	IsPinned     bool              `form:"is_pinned"`
	IsClearCover bool              `form:"is_clear_cover"`
	Tags         string            `form:"tags"`
}

func PostEdit(c *gin.Context, req *PostEditRequest) {
	id := c.Param("id")
	uid := userID(c)
	post, err := store.GetPost(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	ids, err := createTags(req.Tags)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	p := &entity.PostW{
		ID:          id,
		Type:        req.Type,
		Title:       req.Title,
		Slug:        toSlug(req.Slug),
		Excerpt:     req.Excerpt,
		AuthorID:    uid,
		Password:    req.Password,
		Visibility:  req.Visibility,
		Content:     req.Content,
		PinnedAt:    0,
		PublishedAt: req.PublishedAt,
		TagIDs:      ids,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   time.Now().Unix(),
	}
	if req.IsClearCover {
		if err := os.Remove(fmt.Sprintf("data/uploads/covers/%s.jpg", id)); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else {
		if _, err := saveCover(c, id); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	if req.IsPinned {
		p.PinnedAt = time.Now().Unix()
	}
	if req.Visibility == entity.VisibilityPassword && p.Password == "" {
		p.Password = post.Password
	}
	if req.Visibility != entity.VisibilityPassword {
		p.Password = ""
	}
	if err := store.UpdatePost(p); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_post_updated")
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("../post/%s", id))
}

// ===============================
// PostTrash
// ===============================

func PostTrash(c *gin.Context) {
	id := c.Param("id")
	uid := userID(c)
	post, err := store.GetPost(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err := store.TrashPost(id); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_post_trashed")
	c.Redirect(http.StatusFound, "../../posts")
}

// ===============================
// PostUntrash
// ===============================

func PostUntrash(c *gin.Context) {
	id := c.Param("id")
	uid := userID(c)
	post, err := store.GetPostByID(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err := store.UntrashPost(id); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_post_untrashed")
	c.Redirect(http.StatusFound, "../../posts")
}

// ===============================
// TrashClear
// ===============================

func TrashClear(c *gin.Context) {
	if err := store.ClearTrashPostsByUser(userID(c)); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_post_clear")
	c.Redirect(http.StatusFound, "../posts")
}

// ===============================
// PostDelete
// ===============================

func PostDelete(c *gin.Context) {
	id := c.Param("id")
	uid := userID(c)
	post, err := store.GetPostByID(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err := store.DeletePost(id); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_post_deleted")
	c.Redirect(http.StatusFound, "../../posts")
}
