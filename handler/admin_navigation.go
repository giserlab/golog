package handler

import (
	"net/http"
	"sort"
	"strings"

	"golog/entity"
	"golog/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===============================
// NavigationsView
// ===============================

func NavigationsView(c *gin.Context) {
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.HTML(http.StatusOK, "admin_navigations", data(c, gin.H{
		"Navigations": navs,
	}))
}

// ===============================
// NavigationCreate
// ===============================

type NavigationCreateRequest struct {
	Name string `form:"name" binding:"required,max=64" conform:"trim"`
	URL  string `form:"url" binding:"required,url" conform:"trim"`
}

func NavigationCreate(c *gin.Context, req *NavigationCreateRequest) {
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := store.ClearNavigations(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, n := range navs {
		n.Sequence = i + 1
		if err := store.CreateNavigation(&entity.NavigationW{
			ID:       n.ID,
			Name:     n.Name,
			URL:      n.URL,
			Sequence: n.Sequence,
		}); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	if err := store.CreateNavigation(&entity.NavigationW{
		ID:       uuid.New().String(),
		Name:     req.Name,
		URL:      req.URL,
		Sequence: len(navs) + 1,
	}); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_nagivation_created")
	c.Redirect(http.StatusFound, "navigations")
}

// ===============================
// NavigationEdit
// ===============================

type NavigationEditRequest struct {
	Names     []string `form:"name[]" binding:"dive,max=64"`
	URLs      []string `form:"url[]" binding:"dive,url"`
	Sequences []int    `form:"sequence[]" binding:"dive,numeric"`
	IsDeleted []bool   `form:"is_deleted[]"`
}

func NavigationEdit(c *gin.Context, req *NavigationEditRequest) {
	var items []*entity.NavigationW
	// 表单数组（name[]/url[]/sequence[]/is_deleted[]）长度可能不一致，
	// 按下标访问前必须做边界检查，避免 index out of range panic。
	rows := len(req.Names)
	if len(req.URLs) < rows {
		rows = len(req.URLs)
	}
	for i := 0; i < rows; i++ {
		if i < len(req.IsDeleted) && req.IsDeleted[i] {
			continue
		}
		name := strings.TrimSpace(req.Names[i])
		if name == "" {
			continue
		}
		seq := i + 1
		if i < len(req.Sequences) {
			seq = req.Sequences[i]
		}
		items = append(items, &entity.NavigationW{
			ID:       uuid.New().String(),
			Name:     name,
			URL:      strings.TrimSpace(req.URLs[i]),
			Sequence: seq,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Sequence < items[j].Sequence
	})
	if err := store.ClearNavigations(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, n := range items {
		n.Sequence = i + 1
		if err := store.CreateNavigation(n); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	setMessage(c, "notice_nagivation_updated")
	c.Redirect(http.StatusFound, "../navigations")
}
