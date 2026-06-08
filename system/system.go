package system

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	"golog/entity"
	"golog/util"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/teacat/i18n"
)

const (
	dirPerm           = 0755         // 目录权限
	configFilePerm    = 0644         // 配置文件权限
	defaultDateFormat = "2006-01-02" // 默认日期格式
)

var (
	Config *entity.Config

	localeBase *i18n.I18n
	Locale     *i18n.Locale

	themeLocaleBase *i18n.I18n
	themeLocale     *i18n.Locale

	IndexTmpl    *template.Template
	SingularTmpl *template.Template
	MomentTmpl   *template.Template
	WhisperTmpl  *template.Template
	AboutTmpl    *template.Template
	NotFoundTmpl *template.Template
	PowTmpl      *template.Template

	//go:embed locales
	LocalesFS embed.FS
	//go:embed themes
	ThemesFS embed.FS

	markdownCache sync.Map // Markdown渲染缓存

	funcs = template.FuncMap{
		"add": func(x, y int) int {
			return x + y
		},
		"sub": func(x, y int) int {
			return x - y
		},
		"html": func(v string) template.HTML {
			return template.HTML(v)
		},
		"unix2date": func(v int64) string {
			if Config == nil {
				return time.Unix(v, 0).Format(defaultDateFormat)
			}
			return time.Unix(v, 0).Format(Config.DateFormat)
		},
		"markdown": func(v string) template.HTML {
			// 检查缓存
			if cached, ok := markdownCache.Load(v); ok {
				return cached.(template.HTML)
			}

			// 渲染Markdown
			p := parser.NewWithExtensions(parser.CommonExtensions | parser.MathJax | parser.LaxHTMLBlocks | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.Footnotes | parser.SuperSubscript | parser.LaxHTMLBlocks | parser.MathJax | parser.HardLineBreak | parser.Autolink | parser.Strikethrough)
			doc := p.Parse([]byte(v))
			renderer := html.NewRenderer(html.RendererOptions{
				Flags: html.HrefTargetBlank,
			})
			result := template.HTML(markdown.Render(doc, renderer))

			// 存入缓存
			markdownCache.Store(v, result)
			return result
		},
		"__": func(v string) template.HTML {
			return template.HTML(themeLocale.String(v))
		},
		"_f": func(v string, data ...any) string {
			return fmt.Sprintf(themeLocale.String(v), data...)
		},
		"md2html": util.MD2HTML,
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

func init() {
	if err := os.MkdirAll("data/uploads/images", dirPerm); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll("data/uploads/covers", dirPerm); err != nil {
		log.Fatalln(err)
	}
	// init locale
	localeBase = i18n.New("en_US")
	localeBase.LoadFS(LocalesFS, "locales/*.json")

	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		return // non-exist triggers wizard
	}
	b, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalln(err)
	}
	if err := json.Unmarshal(b, &Config); err != nil {
		log.Fatalln(err)
	}

	if Config != nil {
		if err := SaveConfig(); err != nil {
			log.Fatalln(err)
		}
	}
}

func SaveConfig() error {
	b, err := json.MarshalIndent(Config, "", "    ")
	if err != nil {
		return err
	}
	if err := os.WriteFile("config.json", b, configFilePerm); err != nil {
		return err
	}

	tmpl, err := template.New("template.html").Funcs(funcs).ParseFS(ThemesFS, fmt.Sprintf("themes/%s/template.html", Config.Theme))
	if err != nil {
		return err
	}

	// 加载所有模板
	if err := loadAllTemplates(tmpl); err != nil {
		return err
	}

	// load theme locales, or skip if not exists
	themeLocaleBase = i18n.New("default")
	if _, err := fs.Stat(ThemesFS, fmt.Sprintf("themes/%s/locales", Config.Theme)); err == nil {
		themeLocaleBase.LoadFS(ThemesFS, fmt.Sprintf("themes/%s/locales/*.json", Config.Theme))
		themeLocale = themeLocaleBase.NewLocale(Config.Locale)
	}

	ReloadLocale(Config.Locale)

	markdownCache = sync.Map{}

	return nil
}

// loadTemplateFS 从 embed.FS 加载单个模板文件
func loadTemplateFS(tmpl *template.Template, path string) (*template.Template, error) {
	parent, err := tmpl.Clone()
	if err != nil {
		return nil, err
	}
	return parent.ParseFS(ThemesFS, path)
}

// loadAllTemplates 加载所有模板
func loadAllTemplates(tmpl *template.Template) error {
	var err error
	themePath := fmt.Sprintf("themes/%s", Config.Theme)

	IndexTmpl, err = loadTemplateFS(tmpl, fmt.Sprintf("%s/index.html", themePath))
	if err != nil {
		return err
	}

	SingularTmpl, err = loadTemplateFS(tmpl, fmt.Sprintf("%s/singular.html", themePath))
	if err != nil {
		return err
	}

	MomentTmpl, err = loadTemplateFS(tmpl, fmt.Sprintf("%s/moment.html", themePath))
	if err != nil {
		return err
	}

	WhisperTmpl, err = loadTemplateFS(tmpl, fmt.Sprintf("%s/whisper.html", themePath))
	if err != nil {
		return err
	}

	AboutTmpl, err = loadTemplateFS(tmpl, fmt.Sprintf("%s/about.html", themePath))
	if err != nil {
		return err
	}

	NotFoundTmpl, err = loadTemplateFS(tmpl, fmt.Sprintf("%s/404.html", themePath))
	if err != nil {
		return err
	}

	PowTmpl, err = loadTemplateFS(tmpl, fmt.Sprintf("%s/pow.html", themePath))
	if err != nil {
		return err
	}

	return nil
}

// ===============================
// Locale
// ===============================

func ReloadLocale(v ...string) {
	Locale = localeBase.NewLocale(v...)
}

// ===============================
// Themes
// ===============================

func Themes() (themes []string) {
	entries, err := fs.ReadDir(ThemesFS, "themes")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			themes = append(themes, entry.Name())
		}
	}
	slices.Sort(themes)
	return
}

func ThemeExists(v string) bool {
	return slices.Index(Themes(), v) != -1
}
