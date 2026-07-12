package handler

import (
	"fmt"
	"net/http"
	"time"

	"golog/entity"
	"golog/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===============================
// PostRevisionsView
// ===============================

func PostRevisionsView(c *gin.Context) {
	id := c.Param("id")
	uid := userID(c)

	post, err := store.GetPost(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !isCurrentUserAdmin(c) && post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	revs, err := store.ListPostRevisions(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.HTML(http.StatusOK, "admin_post_revisions", data(c, gin.H{
		"Post":     post,
		"Revisions": revs,
	}))
}

// ===============================
// PostRevisionView
// ===============================

func PostRevisionView(c *gin.Context) {
	id := c.Param("id")
	revid := c.Param("revid")
	uid := userID(c)

	post, err := store.GetPost(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !isCurrentUserAdmin(c) && post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	rev, err := store.GetPostRevision(revid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if rev.PostID != id {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.HTML(http.StatusOK, "admin_post_revision_view", data(c, gin.H{
		"Post":     post,
		"Revision": rev,
	}))
}

// ===============================
// PostRevisionRestore
// ===============================

func PostRevisionRestore(c *gin.Context) {
	id := c.Param("id")
	revid := c.Param("revid")
	uid := userID(c)

	post, err := store.GetPost(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !isCurrentUserAdmin(c) && post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	rev, err := store.GetPostRevision(revid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if rev.PostID != id {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Save current state as a revision before restoring
	currentRev := &entity.PostRevision{
		ID:        uuid.New().String(),
		PostID:    id,
		Type:      post.Type,
		Title:     post.Title,
		Slug:      post.Slug,
		Excerpt:   post.OriginalExcerpt,
		Password:  post.Password,
		Visibility: post.Visibility,
		Content:   post.Content,
		PublishedAt: post.PublishedAt,
		PinnedAt:  post.PinnedAt,
		Tags:      post.TagsStr(),
		CreatedAt: time.Now().Unix(),
		CreatedBy: uid,
	}
	if err := store.CreatePostRevision(currentRev); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Parse tags from revision
	tagIDs, err := createTags(rev.Tags)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	p := &entity.PostW{
		ID:          id,
		Type:        rev.Type,
		Title:       rev.Title,
		Slug:        rev.Slug,
		Excerpt:     rev.Excerpt,
		AuthorID:    post.AuthorID,
		Password:    rev.Password,
		Visibility:  rev.Visibility,
		Content:     rev.Content,
		PinnedAt:    rev.PinnedAt,
		PublishedAt: rev.PublishedAt,
		TagIDs:      tagIDs,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   time.Now().Unix(),
	}

	if err := store.UpdatePost(p); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Trim revisions
	if err := store.TrimPostRevisions(id, 30); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	setMessage(c, "notice_post_revision_restored")
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("../../../post/%s", id))
}

// ===============================
// PostRevisionDelete
// ===============================

func PostRevisionDelete(c *gin.Context) {
	id := c.Param("id")
	revid := c.Param("revid")
	uid := userID(c)

	post, err := store.GetPost(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !isCurrentUserAdmin(c) && post.AuthorID != uid {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	rev, err := store.GetPostRevision(revid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if rev.PostID != id {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if err := store.DeletePostRevision(revid); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	setMessage(c, "notice_post_revision_deleted")
	c.Redirect(http.StatusFound, "../../revisions")
}
