package handler

import (
	"bytes"
	"net/http"

	"golog/entity"
	"golog/store"
	"golog/system"

	"github.com/gin-gonic/gin"
)

// ===============================
// NoRouteView
// ===============================

func NoRouteView(c *gin.Context) {
	noRoute(c)
}

func noRoute(c *gin.Context) {
	if system.NotFoundTmpl == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	var routes = []entity.Route{
		{
			Name: "首页",
			Path: "/",
		},
		{
			Name: "404",
			Path: "",
		},
	}
	navs, err := store.ListNavigations()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var tpl bytes.Buffer
	if err := system.NotFoundTmpl.Execute(&tpl, data(c, gin.H{
		"Routes":      routes,
		"Navigations": navs,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusNotFound, "text/html; charset=utf-8", tpl.Bytes())
}
