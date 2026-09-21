# AGENT.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

```bash
# Build for current platform (macOS arm64 by default in makefile)
make

# Build all platforms
make build

# Cross-compilation targets: darwin_amd64, darwin_arm64, linux_amd64, windows_amd64
# Output goes to bin/<os>_<arch>/golog (or .exe for Windows)

# Run directly
go run main.go

# Run tests
go test ./...

# Run a single test
go test ./handler/ -run TestAssetViewSharedFallback -v

# Run database migrations to latest
go run main.go db:migrate

# Reset user password (generates random password)
go run main.go reset-password <email>

# Create an API token
go run main.go token:create <user_id> <name>

# Delete an API token
go run main.go token:delete <token_id>
```

## Architecture

**Framework**: Gin web framework + SQLite (via modernc.org/sqlite, CGo-free)

### Packages

- **`main.go`** — Entry point. CLI app using `urfave/cli/v2`. Defines CLI commands: `reset-password`, `db:migrate`, `token:create`, `token:delete`. Starts the HTTP server via `handler.Start()`.

- **`handler/`** — HTTP handlers and route registration. All routes defined in `handler.go` `init()`. Uses typed handler pattern via `handleForm[T]` generic wrapper (bind, conform, validate, then call typed handler). Key files:
  - `handler.go` — Route registration, template rendering setup, middleware (session, CSRF, auth), generic form handler `handleForm[T]`
  - `handler_util.go` — Shared utilities: session helpers, auth middleware (`checkConfig`, `checkPublic`, `checkLoggedIn`), pagination, image upload/resize, tag creation, rate limiting
  - `setup.go` — Server startup (`Start()` function), runs auto-migration
  - `altcha.go` — Proof-of-Work anti-spam: HMAC-signed cookie challenges, SHA256-based hashcash
  - `admin_*.go` — Admin panel handlers (posts, users, tags, navigation, appearances, settings, photos, tokens, passkeys)
  - `index_*.go` — Public page handlers (index, article, about, RSS, sitemap, wizard, login, noroute, asset serving)
  - `api_post.go` — API endpoint for creating posts with token auth
  - `passkey.go` / `passkey_test.go` — WebAuthn passkey authentication

- **`store/`** — SQLite data access layer. Direct SQL queries (no ORM). Key files:
  - `store.go` — DB connection init, background cleanup goroutines (trash expiry, WebAuthn session cleanup)
  - `migrate.go` — Migration framework: versioned up/down migrations, auto-migrate on startup
  - `post.go` — Post CRUD, listing with `ListPostsQuery` builder pattern (dynamic WHERE clauses), previous/next post navigation, date/tag grouping
  - `user.go`, `tag.go`, `navigation.go`, `token.go`, `webauthn.go` — Corresponding CRUD

- **`entity/`** — Data types (write models `*W`, read models `*R`). No methods on write models, helper methods on read models.
  - `entity.go` — Pagination, timezone map, locale map, page type/relative root path mapping
  - `config.go` — Blog configuration (appearance, PoW settings, theme, locale, WebAuthn)
  - `post.go` — Post types: `blog` (随笔), `moment` (时刻), `whisper` (日志). Visibility: public, private, password, draft, trash
  - `user.go` — User types, WebAuthn user wrapper
  - `tag.go` — Tag types with PostCount
  - `token.go` — API token model
  - `injection.go` — Build metadata injected at compile time

- **`system/`** — Global config, locale/i18n, theme template loading. Config loaded from `config.json` on disk. Themes embedded via `embed.FS`. Templates parsed at startup and reloaded on config save. Markdown render cache via `sync.Map`. Also owns the shared template `FuncMap` (including `readingTime`, `firstImage`, `plainTitle`, `dict`) used by every theme.

- **`view/`** — Embedded admin templates and assets via `embed.FS`

- **`util/`** — Constants (post type keys/names), Markdown-to-HTML conversion, footnote/inline-highlight extensions, sanitization, browser opener, content metrics (`content.go`: reading-time estimate, first-image extraction, plain-title/plain-text helpers)

`util/highlight_ext.go` adds the inline `==高亮内容==` syntax to goldmark (`<mark>`), so it renders in every theme through `md2html`; `util/highlight_gomarkdown.go` mirrors the same syntax for the gomarkdown previews (admin revision view) via `RegisterGomarkdownHighlight` + `GomarkdownHighlightRenderHook`, and the EasyMDE editor templates repeat it client-side in `previewRender` (marked cannot take plugins, so the HTML is post-processed outside tags). Only exactly two `=` on each side match: `===x===`, `1 == 1`, unclosed/empty pairs and code spans stay literal, while nesting like `==**重点**==` works. Highlights must not leak into plain text: `entity.PostR.Excerpt()` strips the markers and `util.PlainTitle` drops `=`.

`util/sanitize.go` runs on every Markdown render (`MD2HTML` and the `markdown` template func) and follows a **trusted-author** model: 正文由作者本人撰写，嵌入标签（`iframe`/`embed`/`object`）的 `src`/`data` **不做 scheme 白名单**——`//host/path`（B 站等分享代码的写法）、`/path`、`file:` 等一律原样保留，只做“不剥除即可渲染”的直通处理；剥除 `src` 会导致文章里只剩空 `iframe`（页面显示空白）。安全边界只有一条通用规则加三项剥除：危险元素（`script`/`style`/`form`/`svg`/`base`/`meta` 等）整体删除、`srcdoc` 与 `on*` 一律剥除、`src`/`href`/`action`/`data` 中的 `javascript:` 与非图片 `data:` 一律剥除（前缀匹配前先 TrimSpace，防 `"  javascript:"` 绕过）。改这里必须同步 `util/sanitize_test.go` 的正反用例。

### Themes

Three built-in themes under `system/themes/`:

- `default/` — Full-featured
- `note/` — Minimal
- `corporate/` — Enterprise blog: sticky header, full-width hero (falls back to the newest post's cover or its first inline image), homepage tiles every post into an even card grid (1 column ≤1080px, 3 columns ≥1081px) with no featured slot and uniform card sizing — the cover (or its placeholder), tag row, 2-line title, 3-line excerpt and single-line meta row all have fixed heights, so cards are exactly equal in size whether or not a post has a cover, tags or a pin, light/dark aware. Content is viewport-fluid (`--page: calc(100% - 2*gutter)`), and on desktop (≥1025px) the content area is exactly 90% of the viewport with the hero kept full-bleed and its text aligned to that 5% inset; the header, hero text, cards and footer therefore share the same left/right edges regardless of window size. On PC the `--gutter` becomes `5%` so the content area is naturally 90% wide (never `max-width: 90%`, which would shrink `.container` itself and break the hero bleed); the hero then uses `width: 100vw` with `margin-left: -5vw`, so width and offset share one reference and the hero is exactly the viewport width — `overflow-x: clip` sits on `.container` only as a sub-pixel guard for the scrollbar gutter that `vw` reserves, leaving `position: sticky` intact (PC width overrides also live at the END of `template.css` — same-specificity rules earlier in the file lose to the base rules); the 全局宽度 setting only changes `--gutter`/base font, and article pages cap their measure with `--article-max`. The palette mirrors the `default` theme (brand `#00b8dd`; light `#fff`/`#363636`/`#8c99a6`/`#cee5ff`/`rgba(177,193,220,.342)`; dark `#070a0f`/`#9babbc`/`#476b91`/`#283039`) and all accents derive from `--brand`, so re-theming means editing `assets/variable.css` only. Brand-filled surfaces with white text use `--brand-dark` to keep that text legible on the bright cyan. Design tokens live in `assets/variable.css`, layout in `assets/template.css`, a light/dark toggle sits at the right of the sticky header (`#theme-toggle` in `template.html`, icon swapped by `html.dark`); the inline head script resolves the colour scheme from `localStorage['golog-color-scheme']` → stored site setting → `prefers-color-scheme` and writes both `data-theme` and the `light`/`dark` class, and clicking the button persists the choice. The article TOC on `post.html` is built by inline JS into `#post-toc-mount`, rendered as a borderless block pinned to the top of the content area on desktop and as an inline collapsible card above the body on ≤1180px (one `--line` border plus radius, and its nav must keep `overflow-x: hidden` + `overflow-wrap: anywhere` so a long heading cannot widen it and eat the right border). The TOC reads `--container-pad-top`, so that variable lives in `:root`, not on `.container`, and its desktop `right` uses `clamp(8px, 5vw, 8%)` (not a percentage of the content area) so it cannot drift off-screen, and interactions (lazy-image fade-in with preload margin) live in `assets/corporate.js`.
- `shared/` — Shared assets (highlight.js, lightbox, footnote, PoW solver, lazy-img)

Theme templates: `template.html` (base), `index.html`, `post.html`, `singular.html`, `moment.html`, `whisper.html`, `about.html`, `404.html`, `altcha.html`. Each theme has locale files under `locales/`. A theme without `altcha.html` falls back to `themes/shared/altcha.html`.

Adding a theme only requires a directory containing `template.html` (plus `locales/` and `assets/`); `system.Themes()` discovers it automatically, and `AssetView` serves `/assets/*` from the theme first, then falls back to `shared/`.

The mobile nav drawer (`.side-bar`) must never be parked just outside the viewport: an off-viewport fixed element (`translateX(102%)` / `right: -100%`) makes some mobile browsers widen the visual viewport, which shows up as the whole page being zoomed out with the right edge clipped. It is hidden with `visibility: hidden` and slides in by animating `right`.

Theme assets are embedded with `go:embed`, so **editing `system/themes/**` has no effect on a running server until the binary is rebuilt** — always `make` (or `go build`) and restart before judging a theme change. To check whether a given binary already contains a fix, grep its strings, e.g. `strings bin/golog_*_darwin_arm64 | grep 'overflow-wrap: anywhere'`.

Lazy images: mark an image with `class="lazy-img"` plus a `data-src` address. A theme must load **exactly one** lazy-load script — either the shared `lazy-img.js` or its own. Loading both makes two IntersectionObservers race for the same `data-src`; the loser reads `null` and overwrites `src` with the string `"null"`, which shows as a flash then a broken image. `corporate` uses its own loader (`assets/corporate.js`, preload margin + fade-in + SVG fallback on error) and therefore must not reference `lazy-img.js`; `TestCorporateSingleLazyLoader` guards this.

### Testing Patterns

- Tests use `gin.TestMode` and `httptest.NewRecorder()` with `gin.CreateTestContext()`
- System config may need to be set up in tests (backup/restore pattern in asset test)
- Theme rendering is covered by `system/corporate_theme_test.go`, which executes the real embedded templates against handler-shaped page data; fake maps must match the pointer shapes `handler.data()` produces (e.g. `*map[[2]string]int` for `TagMap`/`Stats`/`MomentStats`)
- Tests exist in `handler/`, `system/`, and `util/`

### Key Dependencies

- **github.com/gin-gonic/gin** — HTTP framework
- **modernc.org/sqlite** — CGo-free SQLite driver
- **github.com/gomarkdown/markdown** — Markdown rendering (admin previews)
- **github.com/yuin/goldmark** — Markdown rendering (public themes), with mermaid/mathjax/TOC extensions plus the local footnote and `==highlight==` extensions
- **github.com/go-webauthn/webauthn** — Passkey authentication
- **github.com/gin-contrib/sessions** — Cookie-based sessions
- **github.com/utrack/gin-csrf** — CSRF protection
- **github.com/sunshineplan/imgconv** — Image upload resizing
- **github.com/teacat/i18n** — Internationalization (zh-cn, zh-tw, en-us)

### Notable Features

- **PoW anti-spam**: Hashcash-style proof-of-work for public routes. HMAC-signed cookies with configurable difficulty and TTL. Excluded for admin/login/feeds/sitemap.
- **API tokens**: bcrypt-hashed tokens for programmatic post creation via `/api/posts`
- **Post covers**: an uploaded cover file is auto-compressed to max 1024px width, or the create/edit form accepts an image URL (`posts.cover_url`, absolute / site-relative); `PostR.Cover()` prefers the URL and falls back to the upload, and switching to a URL or clearing removes the stale upload
- **Trash system**: Posts soft-deleted for 30 days, then auto-purged by background goroutine

## Self-Maintenance Rule

- After every major change (new modules, new pages, architectural adjustments, introduction of new libraries/technologies, directory structure changes, important convention modifications, etc.), **must** set the "final step" as:  
  "Check whether AGENT.md needs to be updated synchronously, and provide update suggestions or modify it directly."
- The definition of major changes includes, but is not limited to:
- Adding/refactoring major directories
- Introducing new frameworks/state management/build tools
- Major routing/component tree changes
- Changes to specifications/styles/linter rules
- Keep this file as the project's "living document" and "single source of truth."
