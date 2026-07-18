package handler

import (
	"net/http"

	"golog/store"

	"github.com/gin-gonic/gin"
)

// ===============================
// AdminCommentsView
// ===============================

func AdminCommentsView(c *gin.Context) {
	var (
		page         = queryPage(c)
		countPerPage = 50
		status       = c.DefaultQuery("status", "pending")
	)
	if status == "all" {
		status = ""
	}

	comments, total, err := store.ListCommentsByStatus(status, (page-1)*countPerPage, countPerPage)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.HTML(http.StatusOK, "admin_comments", data(c, gin.H{
		"Comments":   comments,
		"Status":     c.DefaultQuery("status", "pending"),
		"Pagination": pagination(c, page, total, countPerPage),
	}))
}

// ===============================
// AdminCommentApprove
// ===============================

type CommentActionRequest struct {
	ID string `form:"id" binding:"required"`
}

func AdminCommentApprove(c *gin.Context, req *CommentActionRequest) {
	if err := store.UpdateCommentStatus(req.ID, "approved"); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_comment_approved")
	c.Redirect(http.StatusFound, "/admin/comments")
}

// ===============================
// AdminCommentReject
// ===============================

func AdminCommentReject(c *gin.Context, req *CommentActionRequest) {
	if err := store.UpdateCommentStatus(req.ID, "rejected"); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_comment_rejected")
	c.Redirect(http.StatusFound, "/admin/comments")
}

// ===============================
// AdminCommentDelete
// ===============================

func AdminCommentDelete(c *gin.Context, req *CommentActionRequest) {
	if err := store.DeleteComment(req.ID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_comment_deleted")
	c.Redirect(http.StatusFound, "/admin/comments")
}
