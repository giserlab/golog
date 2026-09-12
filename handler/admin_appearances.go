package handler

import (
	"net/http"

	"golog/entity"
	"golog/system"

	"github.com/gin-gonic/gin"
)

// ===============================
// AppearancesView
// ===============================

func AppearancesView(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_appearances", data(c, gin.H{
		"Themes": system.Themes(),
	}))
}

// ===============================
// AppearancesEdit
// ===============================

type AppearancesEditRequest struct {
	FooterText     string             `form:"footer_text" conform:"trim"`
	ColorScheme    entity.ColorScheme `form:"color_scheme" binding:"omitempty,oneof=light dark"`
	ContainerWidth string             `form:"container_width" binding:"oneof=small medium large"`
	FontFamily     entity.FontFamily  `form:"font_family" binding:"omitempty,oneof=sans serif"`
	FontSize       string             `form:"font_size" binding:"omitempty,oneof=small medium large"`
	HighlightJS    bool               `form:"highlight_js"`
	AuthorBlock    entity.AuthorBlock `form:"author_block" binding:"oneof=none start end"`
	PostsPerPage   int                `form:"posts_per_page" binding:"min=1,max=999"`
	Theme          string             `form:"theme" binding:"required"`
	Favicon        string             `form:"favicon" conform:"trim"`
	CustomCSS      string             `form:"custom_css" conform:"trim"`
	// 自定义代码：这些字段与自定义 CSS 同属“自定义代码”区块，必须和上面的
	// 外观字段放在同一个表单里提交，否则用户点击本表单的“保存更改”时，
	// 注入代码的修改会被静默丢弃（历史上它们是独立表单，已因此产生缺陷）。
	InjectedHead      string `form:"injected_head" conform:"trim"`
	InjectedFoot      string `form:"injected_foot" conform:"trim"`
	InjectedPostStart string `form:"injected_post_start" conform:"trim"`
	InjectedPostEnd   string `form:"injected_post_end" conform:"trim"`
}

func AppearancesEdit(c *gin.Context, req *AppearancesEditRequest) {
	if !system.ThemeExists(req.Theme) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	system.Config.FooterText = req.FooterText
	system.Config.ColorScheme = req.ColorScheme
	system.Config.ContainerWidth = req.ContainerWidth
	system.Config.FontFamily = req.FontFamily
	system.Config.FontSize = req.FontSize
	system.Config.HighlightJS = req.HighlightJS
	system.Config.AuthorBlock = req.AuthorBlock
	system.Config.PostsPerPage = req.PostsPerPage
	system.Config.Theme = req.Theme
	system.Config.Favicon = req.Favicon
	system.Config.CustomCSS = req.CustomCSS
	system.Config.InjectedHead = req.InjectedHead
	system.Config.InjectedFoot = req.InjectedFoot
	system.Config.InjectedPostStart = req.InjectedPostStart
	system.Config.InjectedPostEnd = req.InjectedPostEnd

	if err := system.SaveConfig(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_appearances_updated")
	c.Redirect(http.StatusFound, "/admin/appearances")
}

// ===============================
// AppearancesEditInjected
// ===============================

// AppearancesEditInjectedRequest 与外观主表单共用同一批注入代码字段。
// 保留该独立端点是为了兼容直接调用的旧客户端；后台页面已改为随外观主表单
// 一起提交，避免出现“保存了但没写入”的误导。
type AppearancesEditInjectedRequest struct {
	InjectedHead      string `form:"injected_head" conform:"trim"`
	InjectedFoot      string `form:"injected_foot" conform:"trim"`
	InjectedPostStart string `form:"injected_post_start" conform:"trim"`
	InjectedPostEnd   string `form:"injected_post_end" conform:"trim"`
}

func AppearancesEditInjected(c *gin.Context, req *AppearancesEditInjectedRequest) {
	system.Config.InjectedHead = req.InjectedHead
	system.Config.InjectedFoot = req.InjectedFoot
	system.Config.InjectedPostStart = req.InjectedPostStart
	system.Config.InjectedPostEnd = req.InjectedPostEnd

	if err := system.SaveConfig(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_injected_updated")
	c.Redirect(http.StatusFound, "../appearances")
}
