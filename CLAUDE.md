# CLAUDE.md

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

# Import legacy config.json into database (needed after upgrading from a version
# that used config.json; the server will not start populated until this is done)
go run main.go config:import [path/to/config.json]
```

## Architecture

**Framework**: Gin web framework + SQLite (via modernc.org/sqlite, CGo-free)

### Packages

- **`main.go`** — Entry point. CLI app using `urfave/cli/v2`. Defines CLI commands: `reset-password`, `db:migrate`, `token:create`, `token:delete`, `config:import`. Starts the HTTP server via `handler.Start()`.

- **`handler/`** — HTTP handlers and route registration. All routes defined in `handler.go` `init()`. Uses typed handler pattern via `handleForm[T]` generic wrapper (bind, conform, validate, then call typed handler). Key files:
  - `handler.go` — Route registration, template rendering setup, middleware (session, CSRF, auth), generic form handler `handleForm[T]`
  - `handler_util.go` — Shared utilities: session helpers, auth middleware (`checkConfig`, `checkPublic`, `checkLoggedIn`), pagination, image upload/resize, tag creation, rate limiting
  - `setup.go` — Server startup (`Start()` function), runs auto-migration, then loads config from database
  - `altcha.go` — Proof-of-Work anti-spam via ALTCHA (`github.com/altcha-org/altcha-lib-go`): `/altcha/challenge` endpoint, ALTCHA widget verification, HMAC-signed long-lived verification cookie with configurable TTL. Excluded for admin/login/wizard/assets/uploads/feeds/sitemap/altcha.
  - `admin_*.go` — Admin panel handlers (posts, users, tags, navigation, appearances, settings, photos, tokens, passkeys)
  - `index_*.go` — Public page handlers (index, article, about, RSS, sitemap, wizard, login, noroute, asset serving)
  - `api_post.go` — API endpoint for creating posts with token auth
  - `passkey.go` / `passkey_test.go` — WebAuthn passkey authentication

- **`store/`** — SQLite data access layer. Direct SQL queries (no ORM). Key files:
  - `store.go` — DB connection init, background cleanup goroutines (trash expiry, WebAuthn session cleanup)
  - `migrate.go` — Migration framework: versioned up/down migrations, auto-migrate on startup
  - `config.go` — Load/save site configuration from the `config` table (single-row, JSON column)
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

- **`system/`** — Global config pointer (`system.Config`), locale/i18n, theme template loading. Config is persisted in the SQLite `config` table (loaded at startup via `handler.Start` → `store.LoadConfig`). `system.Config` is set by `handler.Start()` after auto-migration. `system.SaveConfig()` fills PoW defaults and calls the injected `configWriter` (which writes to the database via `store.SaveConfig`), then reloads templates and locale. Markdown render cache via `sync.Map`.

- **`view/`** — Embedded admin templates and assets via `embed.FS`

- **`util/`** — Constants (post type keys/names), Markdown-to-HTML conversion, footnote extension, sanitization, browser opener

### Upgrade Notes (config.json → database)

Starting from the version that includes migration v4, site configuration is stored in the SQLite `config` table instead of `config.json`. After upgrading:

1. Run database migrations: `golog db:migrate`
2. Import the legacy config: `golog config:import config.json`
3. Start the server normally

The old `config.json` can be removed or kept as a backup after import.

### Themes

Two built-in themes under `system/themes/`:

- `default/` — Full-featured
- `note/` — Minimal
- `shared/` — Shared assets (highlight.js, lightbox, footnote, lazy-img)

Theme templates: `template.html` (base), `index.html`, `singular.html`, `moment.html`, `whisper.html`, `about.html`, `404.html`, `altcha.html`. Each theme has locale files under `locales/`.

### Testing Patterns

- Tests use `gin.TestMode` and `httptest.NewRecorder()` with `gin.CreateTestContext()`
- System config may need to be set up in tests (backup/restore pattern in asset test)
- Tests exist in `handler/` and `util/`

### Key Dependencies

- **github.com/gin-gonic/gin** — HTTP framework
- **modernc.org/sqlite** — CGo-free SQLite driver
- **github.com/gomarkdown/markdown** — Markdown rendering (admin previews)
- **github.com/yuin/goldmark** — Markdown rendering (public themes), with mermaid/mathjax/TOC extensions
- **github.com/go-webauthn/webauthn** — Passkey authentication
- **github.com/gin-contrib/sessions** — Cookie-based sessions
- **github.com/utrack/gin-csrf** — CSRF protection
- **github.com/sunshineplan/imgconv** — Image upload resizing
- **github.com/teacat/i18n** — Internationalization (zh-cn, zh-tw, en-us)
- **github.com/altcha-org/altcha-lib-go** — Self-hosted proof-of-work CAPTCHA alternative

### Notable Features

- **Config storage**: Site configuration is persisted in the SQLite `config` table (single-row, JSON column) instead of a file. The global `system.Config` pointer is loaded after database migration during server startup. All admin settings/appearances changes call `system.SaveConfig()`, which triggers template reloading via the injected `configWriter`.
- **PoW anti-spam**: ALTCHA-based proof-of-work for public routes and 404s. The browser widget fetches `/altcha/challenge`, solves the challenge, and submits the payload to `/altcha/solve`. On success an HMAC-signed verification cookie is issued with configurable TTL (`PoWTTL`). Configurable `PoWMaxNumber` controls challenge difficulty. Excluded for admin/login/wizard/assets/uploads/feeds/sitemap/altcha. Search-engine crawlers can be allowed to bypass verification via `PoWBotBypass` and a configurable `PoWBotUserAgents` list.
- **Admin post isolation**: Logged-in users can only view, edit, trash, and delete their own posts in the admin panel. New posts are always created with the current user as author. Users with the `admin` role can manage all posts and access site-wide settings.
- **User roles**: Users have a `role` column (`admin` or `user`). Existing users from pre-role databases default to `admin` to preserve historical behavior; new users default to `user`. The first user created via the wizard is `admin`.
- **API tokens**: bcrypt-hashed tokens for programmatic post creation via `/api/posts`
- **Automatic cover compression**: Uploaded cover images resized to max 1024px width
- **Trash system**: Posts soft-deleted for 30 days, then auto-purged by background goroutine

## Self-Maintenance Rule

- After every major change (new modules, new pages, architectural adjustments, introduction of new libraries/technologies, directory structure changes, important convention modifications, etc.), **must** set the "final step" as:
  "Check whether CLAUDE.md needs to be updated synchronously, and provide update suggestions or modify it directly."
- The definition of major changes includes, but is not limited to:
- Adding/refactoring major directories
- Introducing new frameworks/state management/build tools
- Major routing/component tree changes
- Changes to specifications/styles/linter rules
- Keep this file as the project's "living document" and "single source of truth."
