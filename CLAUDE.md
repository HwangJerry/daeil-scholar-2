# CLAUDE.md — dflh-saf-v2

Alumni community web application: legacy PHP → modern Go + React rewrite.

## Rule
1. Always comply with the rule of @.claude/hooks/validate-code-file.sh for generating codes.
2. Evaluate whether the code file being written has a single, clearly defined responsibility. A file with one responsibility means: it handles one domain concept, one feature concern, or one layer of abstraction. Files that mix unrelated concerns (e.g., API calls + UI rendering + data transformation) violate single responsibility. If the file is a type definition file, a barrel export, or a configuration file, it is exempt. Return ok=true if it has a single responsibility. If ok=false, include the reason AND this directive: '[FIX REQUIRED] Extract the mixed concern into a separate file. Do not ask for confirmation — fix inline and continue with the current task.
3. UI 디자인 시스템은 `UI_DESIGN_DOC.md`를 참고하여 진행한다. 새로운 컴포넌트나 스타일을 추가할 때 반드시 해당 문서의 디자인 토큰과 컴포넌트 가이드라인을 따른다.

## Quick Reference

| Stack | Version |
|-------|---------|
| Go | 1.23 |
| React | 19 |
| Vite | 7 |
| Tailwind CSS | v4 (CSS-first `@theme`) |
| MariaDB | **10.1.38** (see constraints below) |
| Router (Go) | chi/v5 |
| Router (React) | react-router-dom v7 |
| State | Zustand + React Query v5 |

## Build & Dev Commands

### Backend (Go)

```bash
cd backend && go run ./cmd/server        # Run dev server on :8080
cd backend && go run ./cmd/backfill       # Run backfill migration tool
cd backend && go build ./cmd/server       # Build server binary
cd backend && go test ./...               # Run all tests
cd backend && go vet ./...                # Lint
```

### Frontend (User SPA)

```bash
cd frontend && npm run dev      # Vite dev server on :5173
cd frontend && npm run build    # tsc + vite build → dist/
cd frontend && npm run lint     # ESLint
```

### Admin SPA

```bash
cd admin && npm run dev         # Vite dev server on :3001, base path /admin/
cd admin && npm run build       # tsc + vite build → dist/
cd admin && npm run lint        # ESLint
```

### Nginx Dev Proxy

Unifies all three services behind `localhost:8000`:

```bash
nginx -c "$(pwd)/nginx/dev.conf" -p "$(pwd)/nginx/"
```

| Path | Target |
|------|--------|
| `/` | Frontend Vite `:5173` |
| `/admin/` | Admin Vite `:3001` |
| `/api/`, `/files/`, `/uploads/` | Go backend `:8080` |

### Deployment

```bash
./deploy.sh [user@host]   # Cross-compile Go (linux/amd64), build SPAs, SCP to Gabia, restart
```

Steps: `GOOS=linux GOARCH=amd64 go build` → `npm run build` (both SPAs) → `scp` binaries + dist → `systemctl restart alumni-backend` → `systemctl reload nginx`.

## Architecture

### Monorepo Layout

```
frontend/           # User SPA (React + Vite + Tailwind v4)
admin/              # Admin SPA (React + Vite + Tailwind v4 + MD editor)
backend/            # Go API server
  cmd/server/       # main.go entry point
  cmd/backfill/     # One-off data migration tool
  internal/
    config/         # Env-based config (Config struct)
    handler/        # HTTP handlers (chi routes)
    service/        # Business logic
    repository/     # Database access (sqlx)
    model/          # Domain structs + DB models
    presenter/      # Response formatting (e.g., decode content for API)
    middleware/     # Auth, CORS, CSRF, AdminAuth, Logger, BodyLimit
    job/            # Background jobs (donation snapshot, session cleanup)
  migrations/       # SQL migration files (numbered, manual apply)
nginx/              # Dev + prod Nginx configs
deploy/             # systemd service + logrotate configs
```

### Backend Layered Pattern

```
Handler → Service → Repository
            ↕
         Presenter (response formatting)
```

- **Handler**: Parse HTTP request, call service, write response. No business logic.
- **Service**: Business rules, caching, orchestration. No SQL.
- **Repository**: Raw SQL queries via `sqlx`. No HTTP concerns.
- **Presenter**: Transforms domain models for API responses (e.g., decoding content).
- **Model**: Shared structs for DB rows, API requests/responses.

### Infrastructure

Single Gabia cloud server (1-core, 1GB RAM, 50GB HDD). Nginx reverse proxy → Go `:8080`. MariaDB `:3306` on the same box. Very resource-constrained: `GOMEMLIMIT=80MiB`, `GOMAXPROCS=1`.

## MariaDB 10.1.38 Constraints

**This is critical.** The production database is MariaDB 10.1.38 (EOL 2020). Many modern SQL features are unavailable.

| Feature | Supported? | Alternative |
|---------|-----------|-------------|
| CTE (`WITH ... AS`) | **NO** (10.2+) | Use subqueries |
| Window Functions (`ROW_NUMBER`, `RANK`) | **NO** (10.2+) | `LIMIT`/`OFFSET` or `@variable` |
| JSON column type | **NO** (10.2+) | `TEXT` + app-level JSON parsing |
| `DEFAULT` expressions on datetime | **NO** (10.2+) | Set `NOW()` in application code |
| `INSERT ... ON DUPLICATE KEY UPDATE` | YES | Use for upserts |
| `LIMIT` in subqueries | YES | — |

```go
// BAD — will fail on MariaDB 10.1
WITH recent AS (SELECT * FROM WEO_BOARDBBS ORDER BY SEQ DESC LIMIT 10) SELECT * FROM recent;

// GOOD — use subquery instead
SELECT * FROM (SELECT * FROM WEO_BOARDBBS ORDER BY SEQ DESC LIMIT 10) AS recent;
```

## Authentication

### Dual Auth System (PHP coexistence period)

1. **Primary**: JWT token in `HttpOnly` cookie (`alumni_token`, `SameSite=Lax`, 24h TTL)
2. **Fallback**: Legacy PHP `DDusrSession_id` cookie → lookup in `WEO_MEMBER_LOG` table

On Go login (Kakao OAuth), the backend issues both JWT + legacy `DDusr*` cookies (5 types) so PHP pages still work. The auth middleware checks JWT first, falls back to legacy session lookup.

### Kakao OAuth Flow

`/api/auth/kakao` → Kakao OAuth → `/api/auth/kakao/callback` → JWT issued. If the Kakao account isn't linked to an alumni member, a linking flow (`/api/auth/kakao/link`) is triggered.

### Admin Auth

Admin endpoints (`/api/admin/*`) require both the standard auth middleware and an additional `AdminAuthMiddleware` that checks the user's operator status.

## Content Pipeline (Markdown → HTML → Base64)

Admin writes content in Markdown (via `@uiw/react-md-editor` in the admin SPA).

### Save flow

```
Markdown → goldmark (GFM + syntax highlighting) → HTML
         → bluemonday sanitize → lazy-load images
         → base64 encode → store in DB (CONTENTS column)
         → original Markdown stored in CONTENTS_MD column
         → auto-extract summary (first 200 chars, stripped) + thumbnail URL
```

### Read flow

```
DB CONTENTS (base64) → base64 decode → HTML string → frontend renders via DOMPurify
```

The `content.go` service handles conversion. The `FeedPresenter` calls `DecodeContent()` before returning API responses.

## Code Conventions

### Go Backend

- **Layered architecture**: handler → service → repository. Don't skip layers.
- **Config**: All configuration via environment variables (`config.Load()`). No config files.
- **DB driver**: `sqlx` with `go-sql-driver/mysql`. DSN uses `charset=utf8mb4&parseTime=true&loc=Asia%2FSeoul`.
- **Logging**: `zerolog` (structured JSON logs).
- **Caching**: `go-cache` (in-memory) for donation summaries, alumni filters, feed data.
- **Error responses**: Use the `respond.go` helpers (`respondJSON`, `respondError`).
- **Background jobs**: In `internal/job/`. Started in `main.go`, stopped on graceful shutdown.
- **File uploads**: Go through the `UploadOrchestrator` (validate → resize → store → record in DB).

### Frontend (React/TypeScript)

- **Tailwind v4** with CSS-first `@theme` configuration, Royal Indigo (`#4F46E5`) primary color.
- **Component library**: Radix UI primitives (individual packages) + Lucide icons.
- **Styling utilities**: `clsx` + `tailwind-merge` + `class-variance-authority` (cva).
- **Data fetching**: TanStack React Query v5. API clients in `src/api/`.
- **State management**: Zustand for client state, React Query for server state.
- **HTML rendering**: Sanitize with DOMPurify before `dangerouslySetInnerHTML`.
- **Mobile-first**: Bottom nav (<768px), top header (>=768px).

### Both SPAs

- Vite 7 with `@tailwindcss/vite` plugin.
- TypeScript strict mode (`tsc -b` before build).
- ESLint with react-hooks + react-refresh plugins.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Go server port |
| `ALLOWED_ORIGIN` | `http://localhost:3000` | CORS origin |
| `DB_HOST` | `127.0.0.1` | MariaDB host |
| `DB_PORT` | `3306` | MariaDB port |
| `DB_USER` | `daeilUSER` | DB user |
| `DB_PASSWORD` | `daeilPWD` | DB password |
| `DB_NAME` | `daeilDB` | DB name |
| `KAKAO_CLIENT_ID` | (empty) | Kakao OAuth app key |
| `KAKAO_CLIENT_SECRET` | (empty) | Kakao OAuth secret |
| `KAKAO_REDIRECT_URI` | `http://localhost:8080/api/auth/kakao/callback` | OAuth callback |
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `JWT_MAX_AGE` | `24h` | Token TTL |
| `UPLOAD_BASE_PATH` | `/var/www/uploads` | New file uploads directory |
| `UPLOAD_LEGACY_PATH` | `/var/www/legacy/files` | Legacy PHP uploads directory |
| `UPLOAD_MAX_FILE_SIZE_MB` | `10` | Max upload size |
| `EASYPAY_IMMEDIATELY_MALL_ID` | `05542574` | EasyPay one-time payment merchant ID |
| `EASYPAY_PROFILE_MALL_ID` | `05543499` | EasyPay recurring payment merchant ID |
| `EASYPAY_GW_URL` | `testgw.easypay.co.kr` | EasyPay gateway host (prod: `gw.easypay.co.kr`) |
| `EASYPAY_GW_PORT` | `80` | EasyPay gateway port |
| `EASYPAY_BIN_BASE` | `/var/www/html/_sys/payment` | Base path for `ep_cli` binary and certs |
| `EASYPAY_RETURN_BASE_URL` | `http://localhost:8080` | Base URL for PG return callbacks |
| `PG_AUDIT_LOG_PATH` | `/var/logs/pg/pg-audit.log` | PG payment audit log file path |

## Key Dependencies

### Go

- `go-chi/chi/v5` — HTTP router
- `jmoiron/sqlx` + `go-sql-driver/mysql` — DB access
- `golang-jwt/jwt/v5` — JWT auth
- `yuin/goldmark` + `goldmark-highlighting` — Markdown→HTML
- `microcosm-cc/bluemonday` — HTML sanitization
- `disintegration/imaging` — Image resizing
- `patrickmn/go-cache` — In-memory cache
- `rs/zerolog` — Structured logging

### Frontend (shared between user + admin)

- `react` 19, `react-dom`, `react-router-dom` v7
- `@tanstack/react-query` v5
- `tailwindcss` v4, `@tailwindcss/typography`
- `zustand` — Client state
- `lucide-react` — Icons
- `class-variance-authority`, `clsx`, `tailwind-merge` — Style utilities
- `dompurify` — Sanitize HTML before rendering

### Admin-only

- `@uiw/react-md-editor` — Markdown editor for content authoring
