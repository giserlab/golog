package handler

import (
	"fmt"
	"net/http"
	"time"

	"golog/entity"
	"golog/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
	"golang.org/x/crypto/bcrypt"
)

type TokenCreateRequest struct {
	Name string `form:"name" binding:"required,max=64" conform:"trim"`
}

func TokensView(c *gin.Context) {
	u, err := self(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var tokens []*entity.TokenR
	userMap := make(map[string]string)

	if u.IsAdmin() {
		tokens, err = store.ListTokens()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		users, err := store.ListUsers()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		for _, user := range users {
			userMap[user.ID] = user.Nickname
		}
	} else {
		tokens, err = store.ListTokensByUser(u.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	c.HTML(http.StatusOK, "admin_tokens", data(c, gin.H{
		"Tokens":       tokens,
		"CreatedToken": getCreatedToken(c),
		"UserMap":      userMap,
	}))
}

func TokenCreate(c *gin.Context, req *TokenCreateRequest) {
	u, err := self(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	plainToken := randstr.String(32, randstr.Base62Chars)
	hash, err := bcrypt.GenerateFromPassword([]byte(plainToken), bcrypt.DefaultCost)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	t := &entity.TokenW{
		ID:        uuid.New().String(),
		Name:      req.Name,
		TokenHash: string(hash),
		UserID:    u.ID,
		CreatedAt: time.Now().Unix(),
	}
	if err := store.CreateToken(t); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	setCreatedToken(c, plainToken)
	setMessage(c, "notice_token_created")
	c.Redirect(http.StatusFound, "tokens")
}

func TokenDelete(c *gin.Context) {
	u, err := self(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	id := c.Param("id")

	token, err := store.GetToken(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !u.IsAdmin() && token.UserID != u.ID {
		c.AbortWithError(http.StatusForbidden, fmt.Errorf("token does not belong to current user"))
		return
	}

	if err := store.DeleteToken(id); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	setMessage(c, "notice_token_deleted")
	c.Redirect(http.StatusFound, "../tokens")
}
