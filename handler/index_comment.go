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
	ParentID    string `form:"parent_id" binding:"omitempty,max=64" conform:"trim"`
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

	// 回复校验：父留言必须属于同一篇文章且已通过审核。
	// 未通过审核的留言在公开页面不可见，因此不允许对它回复（防止构造请求绕过）。
	var parent *entity.CommentR
	if req.ParentID != "" {
		parent, err = store.GetComment(req.ParentID)
		if err != nil || parent.PostID != req.PostID || parent.Status != "approved" {
			setMessage(c, "notice_comment_parent_invalid")
			c.Redirect(http.StatusFound, "/post/"+post.Slug+"#comments")
			return
		}
	}

	comment := &entity.CommentW{
		ID:          uuid.New().String(),
		PostID:      req.PostID,
		ParentID:    req.ParentID,
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

	// 通知文章作者（及可选的管理员邮箱）。parent 为 nil 时按顶层留言处理。
	notifyNewComment(c, post, commentReadFromWrite(comment), parent)

	setMessage(c, "notice_comment_submitted")
	c.Redirect(http.StatusFound, "/post/"+post.Slug+"#comments")
}

// commentReadFromWrite 把刚写入的留言模型转换为读取模型，供邮件通知使用。
func commentReadFromWrite(w *entity.CommentW) *entity.CommentR {
	return &entity.CommentR{
		ID:          w.ID,
		PostID:      w.PostID,
		ParentID:    w.ParentID,
		AuthorName:  w.AuthorName,
		AuthorEmail: w.AuthorEmail,
		AuthorURL:   w.AuthorURL,
		Content:     w.Content,
		Status:      w.Status,
		CreatedAt:   w.CreatedAt,
	}
}
