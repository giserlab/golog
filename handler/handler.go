package handler

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"golog/system"
	"golog/util"
	"golog/view"

	"github.com/YamiOdymel/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/leebenson/conform"
	"github.com/thanhpk/randstr"
	csrf "github.com/utrack/gin-csrf"
)

var (
	Router *gin.Engine
	funcs  = template.FuncMap{
		"add": func(x, y int) int {
			return x + y
		},
		"sub": func(x, y int) int {
			return x - y
		},
		"seq": func(start, end int) []int {
			if start > end {
				return []int{}
			}
			seq := []int{}
			for i := start; i <= end; i++ {
				seq = append(seq, i)
			}
			return seq
		},
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"max": func(a, b int) int {
			if a > b {
				return a
			}
			return b
		},
		"html": func(v string) template.HTML {
			return template.HTML(v)
		},
		"unix2date": func(v int64) string {
			return time.Unix(v, 0).Format(system.Config.DateFormat)
		},
		"timezone": func(v int) string {
			return time.Unix(time.Now().Unix()+int64(v), 0).UTC().Format("2006-01-02 03:04 PM")
		},
		"markdown": func(v string) template.HTML {
			p := parser.NewWithExtensions(parser.CommonExtensions | parser.MathJax | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.Footnotes | parser.SuperSubscript | parser.HardLineBreak | parser.Autolink | parser.Strikethrough)
			doc := p.Parse([]byte(v))

			renderer := html.NewRenderer(html.RendererOptions{
				Flags: html.HrefTargetBlank,
			})

			return template.HTML(util.SanitizeHTML(string(markdown.Render(doc, renderer))))
		},
		"md2html": util.MD2HTML,
		"__": func(v string) template.HTML {
			return template.HTML(system.Locale.String(v))
		},
		"_f": func(v string, data ...any) string {
			return fmt.Sprintf(system.Locale.String(v), data...)
		},
		"ptn": func(v string) string {
			switch v {
			case util.BlogType:
				return util.BlogKey
			case util.MomentType:
				return util.MomentKey
			case util.WhisperType:
				return util.WhisperKey
			default:
				return v
			}
		},
	}
)

// securityHeaders adds security-related HTTP response headers.
func securityHeaders(c *gin.Context) {
	c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	c.Header("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' https://cdn.jsdelivr.net https://unpkg.com 'unsafe-inline' 'unsafe-eval'; "+
			"style-src 'self' https://cdn.jsdelivr.net https://fonts.googleapis.com 'unsafe-inline'; "+
			"img-src 'self' https: http://www.gravatar.com data: blob:; "+
			"font-src 'self' https://fonts.gstatic.com data:; "+
			"frame-ancestors 'none'; "+
			"form-action 'self'; "+
			"base-uri 'self'; "+
			"object-src 'none'")
	c.Next()
}

func init() {
	gin.SetMode(gin.ReleaseMode)

	Router = gin.Default()
	Router.Use(securityHeaders)

	Router.Use(
		sessions.Sessions("golog", cookie.NewStore([]byte(randstr.String(64, randstr.Base62Chars)))),
		csrf.Middleware(csrf.Options{
			Secret: randstr.String(64, randstr.Base62Chars),
			ErrorFunc: func(c *gin.Context) {
				c.AbortWithError(http.StatusBadRequest, errors.New("CSRF token mismatch"))
			},
		}))

	render := multitemplate.NewRenderer()
	render.AddFromFSFuncs("wizard", funcs, view.Templates, "templates/wizard.html")
	render.AddFromFSFuncs("login", funcs, view.Templates, "templates/login.html")
	render.AddFromFSFuncs("admin_users", funcs, view.Templates, "templates/admin_base.html", "templates/admin_users.html")
	render.AddFromFSFuncs("admin_user_edit", funcs, view.Templates, "templates/admin_base.html", "templates/admin_user_edit.html")
	render.AddFromFSFuncs("admin_navigations", funcs, view.Templates, "templates/admin_base.html", "templates/admin_navigations.html")
	render.AddFromFSFuncs("admin_tags", funcs, view.Templates, "templates/admin_base.html", "templates/admin_pagination.html", "templates/admin_tags.html")
	render.AddFromFSFuncs("admin_tag_edit", funcs, view.Templates, "templates/admin_base.html", "templates/admin_tag_edit.html")
	render.AddFromFSFuncs("admin_settings", funcs, view.Templates, "templates/admin_base.html", "templates/admin_settings.html")
	render.AddFromFSFuncs("admin_appearances", funcs, view.Templates, "templates/admin_base.html", "templates/admin_appearances.html")
	render.AddFromFSFuncs("admin_post_create", funcs, view.Templates, "templates/admin_base.html", "templates/admin_post_create.html")
	render.AddFromFSFuncs("admin_posts", funcs, view.Templates, "templates/admin_base.html", "templates/admin_pagination.html", "templates/admin_posts.html")
	render.AddFromFSFuncs("admin_post_edit", funcs, view.Templates, "templates/admin_base.html", "templates/admin_post_edit.html")
	render.AddFromFSFuncs("admin_photos", funcs, view.Templates, "templates/admin_base.html", "templates/admin_pagination.html", "templates/admin_photos.html")
	Router.HTMLRender = render

	fs, err := fs.Sub(view.Assets, "assets")
	if err != nil {
		log.Fatalln(err)
	}
	Router.NoRoute(checkConfig, NoRouteView)
	Router.Static("/post/uploads", "data/uploads")
	Router.GET("/wizard", WizardView)
	Router.POST("/wizard", handleForm(Wizard))
	Router.GET("/login", checkConfig, LoginView)
	Router.POST("/login", checkConfig, throttle, handleForm(Login))

	// admin assets (publicly accessible so login/wizard pages can load them)
	Router.StaticFS("/admin/assets", http.FS(fs))

	// admin
	adminRoute := Router.Group("/admin", checkConfig, checkLoggedIn)
	{
		adminRoute.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "posts") })

		adminRoute.Static("/uploads", "data/uploads")
		adminRoute.Static("/post/uploads", "data/uploads")

		adminRoute.GET("/users", UsersView)
		adminRoute.POST("/users", handleForm(UserCreate))

		adminRoute.GET("/user/:id", UserEditView)
		adminRoute.POST("/user/:id", handleForm(UserEdit))
		adminRoute.POST("/user/:id/delete", handleForm(UserDelete))

		adminRoute.GET("/navigations", NavigationsView)
		adminRoute.POST("/navigations", handleForm(NavigationCreate))
		adminRoute.POST("/navigations/edit", handleForm(NavigationEdit))

		adminRoute.GET("/tags", TagsView)
		adminRoute.POST("/tags", handleForm(TagCreate))

		adminRoute.GET("/tag/:id", TagEditView)
		adminRoute.POST("/tag/:id", handleForm(TagEdit))
		adminRoute.POST("/tag/:id/delete", TagDelete)

		adminRoute.GET("/settings", SettingsView)
		adminRoute.POST("/settings", handleForm(SettingsEdit))

		adminRoute.GET("/appearances", AppearancesView)
		adminRoute.POST("/appearances", handleForm(AppearancesEdit))
		adminRoute.POST("/appearances/injected", handleForm(AppearancesEditInjected))

		adminRoute.GET("/post/create", PostCreateView)
		adminRoute.POST("/post/create", handleForm(PostCreate))

		adminRoute.GET("/posts", PostsView)
		adminRoute.POST("/trashes/clear", TrashClear)
		adminRoute.GET("/post/:id", PostEditView)
		adminRoute.POST("/post/:id", handleForm(PostEdit))
		adminRoute.POST("/post/:id/delete", PostDelete)
		adminRoute.POST("/post/:id/trash", PostTrash)
		adminRoute.POST("/post/:id/untrash", PostUntrash)

		adminRoute.POST("/photos/api", PhotoCreate)
		adminRoute.GET("/photos", PhotosView)
		adminRoute.POST("/photos", handleForm(PhotoUpload))
		adminRoute.POST("/photo/delete", handleForm(PhotoDelete))

		adminRoute.POST("/logout", Logout)
	}

	publicRoute := Router.Group("/", checkConfig, checkPublic)
	{
		publicRoute.Static("/uploads", "data/uploads")
		publicRoute.GET("/", IndexView)
		publicRoute.GET("/about", AboutView)
		publicRoute.GET("/sitemap.xml", SiteMapView)
		publicRoute.GET("/rss.xml", RSSView)
		publicRoute.GET("/feed.xml", RSSView)
		publicRoute.GET("/assets/:asset", AssetView)

		publicRoute.GET("/tag/:tag", IndexView)
		publicRoute.GET("/author/:author", IndexView)
		publicRoute.GET("/archive/:year", IndexView)
		publicRoute.GET("/archive/:year/:month", IndexView)
		publicRoute.GET("/archive/:year/:month/:day", IndexView)

		publicRoute.GET("/post/:slug", SingularView)
		publicRoute.GET("/post/auto/create", PostCreateViewAuto)
		publicRoute.GET("/blog/:id", SingularViewByID)
		publicRoute.GET("/moment", MomentView)
		publicRoute.GET("/moment/:year", MomentView)
		publicRoute.GET("/whisper", WhisperView)
		publicRoute.POST("/post/:slug", throttle, SingularView)
	}
}

func handleForm[T any](fn func(*gin.Context, T)) gin.HandlerFunc {
	valid := validator.New(validator.WithRequiredStructEnabled())

	return func(c *gin.Context) {
		var req T

		if err := c.ShouldBind(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := conform.Strings(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := valid.Struct(req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fn(c, req)
	}
}
