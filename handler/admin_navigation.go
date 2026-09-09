package handler

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"golog/entity"
	"golog/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===============================
// NavigationsView
// ===============================

// normalizeNavigationURL 规范化并校验导航链接地址。
//
// 支持的取值（trim 后）：
//  1. 完整的绝对 URL：带 scheme（http/https/mailto 等），但排除
//     javascript:/data: 等危险 scheme（模板渲染时 html/template 会再过滤，
//     这里提前拦截做纵深防御）；
//  2. 以 / 开头的站内相对路径：如 /feed.xml（含协议相对 //host/path），
//     添加 RSS、站点内页等链接时无需填写完整域名；
//  3. 其余不带 scheme 的字符串按站内相对路径处理，自动补 / 前缀
//     （如 feed.xml → /feed.xml）。导航栏在每个页面都会出现，普通相对
//     地址会随页面路径深度解析错位，因此统一转为根相对地址。
func normalizeNavigationURL(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}
	if strings.HasPrefix(s, "/") {
		// ParseRequestURI 只接受绝对 URI 或绝对路径，
		// 可顺带过滤含非法字符的输入。
		if _, err := url.ParseRequestURI(s); err != nil {
			return "", false
		}
		return s, true
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", false
	}
	if u.Scheme != "" {
		switch strings.ToLower(u.Scheme) {
		case "javascript", "vbscript", "data", "file":
			return "", false
		}
		// 拒绝无实质内容的 scheme 输入，如 "https://"
		if u.Host == "" && u.Opaque == "" && u.Fragment == "" {
			return "", false
		}
		return s, true
	}
	// 无 scheme：视为站内相对路径，统一补 / 前缀。
	rooted := "/" + s
	if _, err := url.ParseRequestURI(rooted); err != nil {
		return "", false
	}
	return rooted, true
}

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
	URL  string `form:"url" binding:"required,max=2048" conform:"trim"`
}

func NavigationCreate(c *gin.Context, req *NavigationCreateRequest) {
	navURL, ok := normalizeNavigationURL(req.URL)
	if !ok {
		formError(c, errors.New("invalid navigation url"))
		return
	}
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
		URL:      navURL,
		Sequence: len(navs) + 1,
	}); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_nagivation_created")
	c.Redirect(http.StatusFound, "/admin/navigations")
}

// ===============================
// NavigationEdit
// ===============================

type NavigationEditRequest struct {
	Names     []string `form:"name[]" binding:"dive,max=64"`
	URLs      []string `form:"url[]" binding:"dive,max=2048"`
	// 注意：不能用 []int。conform.Strings 会递归处理结构体所有字段，
	// 而 Go 中 int 可转换为 string（如 string(65) == "A"），导致 conform
	// 把 []int 当作字符串切片处理，transformValue 里 string→int 转换直接 panic
	// （reflect.Value.Convert: value of type string cannot be converted to type int）。
	// 因此这里绑定为 []string，在 handler 内手动解析为 int。
	Sequences []string `form:"sequence[]" binding:"dive,numeric"`
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
		navURL, ok := normalizeNavigationURL(req.URLs[i])
		if !ok {
			formError(c, errors.New("invalid navigation url"))
			return
		}
		seq := i + 1
		if i < len(req.Sequences) {
			// 表单里 sequence[] 是字符串，这里解析为 int；
			// 解析失败（理论上有 dive,numeric 校验兜底，不应发生）则退回行号。
			if n, err := strconv.Atoi(strings.TrimSpace(req.Sequences[i])); err == nil {
				seq = n
			}
		}
		items = append(items, &entity.NavigationW{
			ID:       uuid.New().String(),
			Name:     name,
			URL:      navURL,
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
