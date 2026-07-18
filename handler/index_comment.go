package handler

import (
	"net/http"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===============================
// CommentCreate
// ===============================

type CommentCreateRequest struct {
	PostID      string `form:"post_id" binding:"required" conform:"trim"`
	AuthorName  string `form:"author_name" binding:"required,max=64" conform:"trim"`
	AuthorEmail string `form:"author_email" binding:"required,max=128,email" conform:"trim"`
	AuthorURL   string `form:"author_url" binding:"omitempty,max=256,url" conform:"trim"`
	Content     string `form:"content" binding:"required,max=2000" conform:"trim"`
	Altcha      string `form:"altcha"`
}

func CommentCreate(c *gin.Context, req *CommentCreateRequest) {
	if system.Config == nil || !system.Config.CommentsEnabled {
		noRoute(c)
		return
	}
	if !verifyOneTimeAltcha(req.Altcha) {
		setMessage(c, "notice_form_invalid")
		redirect := c.Request.Referer()
		if redirect == "" {
			redirect = "/"
		}
		c.Redirect(http.StatusFound, redirect)
		return
	}

	post, err := store.GetPost(req.PostID)
	if err != nil {
		if store.IsNotFound(err) {
			noRoute(c)
			return
		}
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if post.Visibility != entity.VisibilityPublic && post.Visibility != entity.VisibilityPassword {
		noRoute(c)
		return
	}

	comment := &entity.CommentW{
		ID:          uuid.New().String(),
		PostID:      req.PostID,
		AuthorName:  req.AuthorName,
		AuthorEmail: req.AuthorEmail,
		AuthorURL:   req.AuthorURL,
		Content:     req.Content,
		Status:      "pending",
		CreatedAt:   time.Now().Unix(),
	}
	if err := store.CreateComment(comment); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	setMessage(c, "notice_comment_submitted")
	c.Redirect(http.StatusFound, "/post/"+post.Slug+"#comments")
}
