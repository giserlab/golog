package handler

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"golog/system"

	"github.com/gin-gonic/gin"
)

// ===============================
// AssetView
// ===============================

func AssetView(c *gin.Context) {
	theme := "default"
	if system.Config != nil {
		theme = system.Config.Theme
	}
	asset := c.Param("asset")

	cleanAsset := path.Clean("/" + asset)
	cleanAsset = strings.TrimPrefix(cleanAsset, "/")
	if cleanAsset == "" || cleanAsset == "." {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 优先使用主题自己的资源
	themePath := fmt.Sprintf("themes/%s/assets", theme)
	if _, err := fs.Stat(system.ThemesFS, path.Join(themePath, cleanAsset)); err == nil {
		fsys, err := fs.Sub(system.ThemesFS, themePath)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.FileFromFS(cleanAsset, http.FS(fsys))
		return
	}

	// 回退到共享资源
	sharedPath := "themes/shared"
	fsys, err := fs.Sub(system.ThemesFS, sharedPath)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.FileFromFS(cleanAsset, http.FS(fsys))
}
