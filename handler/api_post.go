package handler

import (
	"net/http"
	"strings"
	"time"

	"golog/entity"
	"golog/store"
	"golog/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type APIPostCreateRequest struct {
	Type        string            `json:"type" conform:"trim"`
	Title       string            `json:"title" binding:"required,max=128" conform:"trim"`
	Slug        string            `json:"slug" binding:"required" conform:"trim"`
	Excerpt     string            `json:"excerpt" conform:"trim"`
	Password    string            `json:"password" binding:"max=128" conform:"trim"`
	Visibility  entity.Visibility `json:"visibility" binding:"required,oneof=public private password draft"`
	Content     string            `json:"content" conform:"trim"`
	PublishedAt int64             `json:"published_at"`
	IsPinned    bool              `json:"is_pinned"`
	Tags        string            `json:"tags" conform:"trim"`
}

type APIPostCreateResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	ID   string `json:"id,omitempty"`
}

func APIPostCreate(c *gin.Context, req *APIPostCreateRequest) {
	// 1. Authenticate via Bearer token
	authHeader := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		c.JSON(http.StatusUnauthorized, APIPostCreateResponse{Code: 401, Msg: "missing or invalid authorization header"})
		return
	}
	providedToken := strings.TrimPrefix(authHeader, prefix)

	token, err := store.GetTokenByHash(providedToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, APIPostCreateResponse{Code: 401, Msg: "invalid token"})
		return
	}

	// 2. Verify the token's user exists
	user, err := store.GetUser(token.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIPostCreateResponse{Code: 500, Msg: "token owner not found"})
		return
	}

	// 3. Create the post
	pid := uuid.New().String()
	ids, err := createTags(req.Tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIPostCreateResponse{Code: 500, Msg: err.Error()})
		return
	}

	p := &entity.PostW{
		ID:          pid,
		Type:        defaultPostType(req.Type),
		Title:       req.Title,
		Slug:        toSlug(req.Slug),
		Excerpt:     req.Excerpt,
		AuthorID:    user.ID,
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
		c.JSON(http.StatusInternalServerError, APIPostCreateResponse{Code: 500, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, APIPostCreateResponse{Code: 200, Msg: "success", ID: pid})
}

// defaultPostType returns the given post type if valid, otherwise defaults to blog.
func defaultPostType(t string) string {
	switch t {
	case util.BlogType, util.MomentType, util.WhisperType:
		return t
	default:
		return util.BlogType
	}
}
