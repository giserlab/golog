package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golog/entity"
	"golog/system"

	"github.com/gin-gonic/gin"
)

func TestAssetViewSharedFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 备份当前主题
	originalTheme := ""
	if system.Config != nil {
		originalTheme = system.Config.Theme
	}

	// 确保有 Config 可用
	if system.Config == nil {
		system.Config = &entity.Config{Theme: "default"}
	}

	restore := func() {
		if originalTheme == "" {
			system.Config = nil
		} else {
			system.Config.Theme = originalTheme
		}
	}
	defer restore()

	cases := []struct {
		name       string
		theme      string
		asset      string
		wantStatus int
	}{
		{
			name:       "shared js fallback for note theme",
			theme:      "note",
			asset:      "/highlight.js",
			wantStatus: http.StatusOK,
		},
		{
			name:       "shared js fallback for default theme",
			theme:      "default",
			asset:      "/lazy-img.js",
			wantStatus: http.StatusOK,
		},
		{
			name:       "theme-specific js takes precedence",
			theme:      "default",
			asset:      "/lightbox.js",
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing asset returns 404",
			theme:      "note",
			asset:      "/lightbox.js",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			system.Config.Theme = tt.theme

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/assets"+tt.asset, nil)
			c.Params = gin.Params{{Key: "asset", Value: tt.asset}}

			AssetView(c)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
