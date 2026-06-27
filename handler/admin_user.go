package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"golog/entity"
	"golog/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ============================
//  UsersView
// ============================

func UsersView(c *gin.Context) {
	users, err := store.ListUsers()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.HTML(http.StatusOK, "admin_users", data(c, gin.H{
		"Users": users,
	}))
}

// ============================
//  UserCreate
// ============================

type UserCreateRequest struct {
	Email    string `form:"email" binding:"required,email" conform:"trim"`
	Password string `form:"password" binding:"required,min=8,max=128" conform:"trim"`
	Role     string `form:"role" binding:"omitempty,oneof=admin user" conform:"trim"`
}

func UserCreate(c *gin.Context, req *UserCreateRequest) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	nickname := strings.Split(req.Email, "@")[0]

	exists, err := store.UserNicknameExists(nickname)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if exists {
		nickname = fmt.Sprintf("%s-%d", nickname, time.Now().Unix())
	}
	role := req.Role
	if role == "" {
		role = "user"
	}
	if err := store.CreateUser(&entity.UserW{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Password:  string(hashedPwd),
		Nickname:  nickname,
		Bio:       "",
		Role:      role,
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_user_created")
	c.Redirect(http.StatusFound, "users")
}

// ============================
//  UserEditView
// ============================

func UserEditView(c *gin.Context) {
	selfUser, err := self(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	u, err := store.GetUser(c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !selfUser.IsAdmin() && selfUser.ID != u.ID {
		setMessage(c, "notice_unauthorized")
		c.Redirect(http.StatusFound, "../posts")
		return
	}
	var users []*entity.UserR
	if selfUser.IsAdmin() {
		users, err = store.ListUsers()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	count, err := store.CountPostsByUser(u.ID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.HTML(http.StatusOK, "admin_user_edit", data(c, gin.H{
		"Users":      users,
		"User":       u,
		"IsSelf":     selfUser.ID == u.ID,
		"IsAdmin":    selfUser.IsAdmin(),
		"IsOnlyUser": len(users) == 1,
		"PostCount":  count,
	}))
}

// ============================
//  UserEdit
// ============================

type UserEditRequest struct {
	Email    string `form:"email" binding:"required,email" conform:"trim"`
	Password string `form:"password" binding:"omitempty,min=8,max=128" conform:"trim"`
	Nickname string `form:"nickname" binding:"required,min=1,max=32" conform:"trim"`
	Bio      string `form:"bio" binding:"max=255" conform:"trim"`
	Role     string `form:"role" binding:"omitempty,oneof=admin user" conform:"trim"`
}

func UserEdit(c *gin.Context, req *UserEditRequest) {
	selfUser, err := self(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	id := c.Param("id")
	target, err := store.GetUser(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !selfUser.IsAdmin() && selfUser.ID != target.ID {
		setMessage(c, "notice_unauthorized")
		c.Redirect(http.StatusFound, "../posts")
		return
	}
	if req.Password != "" {
		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if err := store.UpdateUserPassword(id, string(hashedPwd)); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	role := target.Role
	if req.Role != "" && selfUser.IsAdmin() && selfUser.ID != target.ID {
		role = req.Role
	}

	if err := store.UpdateUser(id, req.Nickname, req.Bio, req.Email, role); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_user_updated")
	if selfUser.IsAdmin() {
		c.Redirect(http.StatusFound, "../users")
	} else {
		c.Redirect(http.StatusFound, "../posts")
	}
}

// ============================
//  UserDelete
// ============================

type UserDeleteRequest struct {
	TransferToID string `form:"transfer_to_id" binding:"omitempty,uuid"`
}

func UserDelete(c *gin.Context, req *UserDeleteRequest) {
	id := c.Param("id")

	if req.TransferToID != "" {
		if _, err := store.GetUser(req.TransferToID); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if err := store.TransferPosts(id, req.TransferToID); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		setMessage(c, "notice_user_deletedwithposts")
	} else {
		if err := store.DeletePostsByUser(id); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		setMessage(c, "notice_user_deleted")
	}
	if err := store.DeleteUser(id); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Redirect(http.StatusFound, "../../users")
}
