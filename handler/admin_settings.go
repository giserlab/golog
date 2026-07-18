package handler

import (
	"net/http"
	"runtime"
	"strings"
	"time"

	"golog/entity"
	"golog/system"

	"github.com/gin-gonic/gin"
)

// ============================
//  SettingsView
// ============================

func SettingsView(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_settings", data(c, gin.H{
		"Version":            injection.Version,
		"RuntimeVersion":     runtime.Version(),
		"BuildTime":          injection.BuildTime,
		"Commit":             injection.Commit,
		"GitHub":             `<a href="https://github.com/giserlab/golog" target="_blank">https://github.com/giserlab/golog</a>`,
		"Timezones":          entity.Timezones,
		"Locales":            entity.Locales,
		"IsCustomTimeFormat": system.Config.IsCustomTimeFormat(),
		"IsCustomDateFormat": system.Config.IsCustomDateFormat(),
		"Year":               time.Now().Format("2006"),
		"Month":              time.Now().Format("01"),
		"Day":                time.Now().Format("02"),
		"Hour":               time.Now().Format("03"),
		"Hour24":             time.Now().Format("15"),
		"Minute":             time.Now().Format("04"),
		"Clock":              time.Now().Format("PM"),
		"PoWEnabled":         system.Config.PoWEnabled,
		"PoWMaxNumber":       system.Config.PoWMaxNumber,
		"PoWTTL":             system.Config.PoWTTL,
		"PoWBotBypass":       system.Config.PoWBotBypass,
		"PoWBotUserAgents":   strings.Join(system.Config.PoWBotUserAgents, "\n"),
		"CommentsEnabled":    system.Config.CommentsEnabled,
	}))
}

// ============================
//  SettingsEdit
// ============================

type SettingsEditRequest struct {
	Name              string `form:"name" binding:"required,max=64" conform:"trim"`
	Description       string `form:"description" binding:"required,max=128" conform:"trim"`
	About             string `form:"about" binding:"required" conform:"trim"`
	IsPublic          bool   `form:"is_public"`
	Timezone          int    `form:"timezone" binding:"min=-43200,max=50400"`
	DateFormat        string `form:"date_format" binding:"required"`
	DateFormatCustom  string `form:"date_format_custom" conform:"trim"`
	TimeFormat        string `form:"time_format" binding:"required"`
	TimeFormatCustom  string `form:"time_format_custom" conform:"trim"`
	Locale            string `form:"locale" binding:"required"`
	PoWEnabled        bool   `form:"pow_enabled"`
	PoWMaxNumber      int64  `form:"pow_max_number" binding:"min=1000,max=10000000"`
	PoWTTL            int    `form:"pow_ttl" binding:"min=1,max=168"`
	PoWBotBypass      bool   `form:"pow_bot_bypass"`
	PoWBotUserAgents  string `form:"pow_bot_user_agents" conform:"trim"`
	CommentsEnabled   bool   `form:"comments_enabled"`
}

func SettingsEdit(c *gin.Context, req *SettingsEditRequest) {
	system.Config.Name = req.Name
	system.Config.About = req.About
	system.Config.Description = req.Description
	system.Config.IsPublic = req.IsPublic
	system.Config.Timezone = req.Timezone
	system.Config.Locale = req.Locale
	system.Config.PoWEnabled = req.PoWEnabled
	if req.PoWMaxNumber >= 1000 && req.PoWMaxNumber <= 10000000 {
		system.Config.PoWMaxNumber = req.PoWMaxNumber
	} else if system.Config.PoWMaxNumber == 0 {
		system.Config.PoWMaxNumber = 200000 // default
	}
	if req.PoWTTL >= 1 && req.PoWTTL <= 168 {
		system.Config.PoWTTL = req.PoWTTL
	} else if system.Config.PoWTTL == 0 {
		system.Config.PoWTTL = 24 // default
	}
	system.Config.PoWBotBypass = req.PoWBotBypass
	system.Config.PoWBotUserAgents = parsePowBotUserAgents(req.PoWBotUserAgents)
	system.Config.CommentsEnabled = req.CommentsEnabled

	if req.DateFormat == "custom" {
		system.Config.DateFormat = req.DateFormatCustom
	} else {
		system.Config.DateFormat = req.DateFormat
	}
	if req.TimeFormat == "custom" {
		system.Config.TimeFormat = req.TimeFormatCustom
	} else {
		system.Config.TimeFormat = req.TimeFormat
	}
	if err := system.SaveConfig(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_settings_updated")
	c.Redirect(http.StatusFound, "settings")
}

// parsePowBotUserAgents splits a newline-separated list of user-agent strings,
// trimming whitespace and filtering out empty lines.
func parsePowBotUserAgents(raw string) []string {
	if raw == "" {
		return nil
	}
	var agents []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			agents = append(agents, line)
		}
	}
	return agents
}
