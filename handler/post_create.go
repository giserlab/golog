package handler

import (
	"net/http"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"
	"golog/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type PostCreateObject struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Content  string `json:"content"`
}

type PostCreateResponseObject struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func PostCreateViewAuto(c *gin.Context) {
	// Require API key; endpoint is disabled if no key is configured
	if system.Config.APIKey == "" {
		c.JSON(http.StatusForbidden, PostCreateResponseObject{Code: 403, Msg: "API endpoint is disabled"})
		return
	}
	if c.Query("api_key") != system.Config.APIKey {
		c.JSON(http.StatusForbidden, PostCreateResponseObject{Code: 403, Msg: "invalid api key"})
		return
	}

	username := c.Query("user")
	password := c.Query("password")
	content := c.Query("content")

	user, err := store.GetUserByEmail(username)
	if err != nil {
		c.JSON(200, PostCreateResponseObject{Code: 201, Msg: "user not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.JSON(200, PostCreateResponseObject{Code: 201, Msg: "user not found"})
		return
	}
	now := time.Now().Unix()
	err = store.CreatePost(&entity.PostW{
		ID:          uuid.New().String(),
		Type:        util.WhisperType,
		AuthorID:    user.ID,
		Content:     content,
		Visibility:  entity.VisibilityPublic,
		Password:    "",
		Title:       content,
		Excerpt:     content,
		Slug:        content,
		PinnedAt:    0,
		PublishedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		c.JSON(200, PostCreateResponseObject{Code: 201, Msg: err.Error()})
		return
	}

	c.JSON(200, PostCreateResponseObject{Code: 200, Msg: "success"})
}
