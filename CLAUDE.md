# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository overview

This repo is a 3-part system:

- `frontend/`: Vue 3 + Vite admin UI (Pinia, vue-router, Ant Design Vue).
- `backend/`: Gin + GORM API server exposing `/api/*` endpoints for user/admin workflows and dashboards.
- `sync_data/`: Go worker that syncs external Jumia/consignment data into MySQL, refreshes tokens, and updates service-status records.

`backend` and `sync_data` are separate Go modules but operate on the same MySQL schema and very similar model/config structures.

## Common commands

### Frontend (`frontend/`)

```bash
cd frontend
npm install
npm run dev
npm run build
npm run preview
```

### Backend API (`backend/`)

```bash
cd backend
go mod download
go run .
go run . -db        # run migrations only, then exit
go build -o main main.go
go test ./...
go test ./... -run TestName -count=1   # single test by name
```

### Sync worker (`sync_data/`)

```bash
cd sync_data
go mod download
go run .
go run . -db        # run migrations only, then exit
go build -o main main.go
go test ./...
go test ./... -run TestName -count=1   # single test by name
```

## Configuration and runtime coupling

- Shared runtime config is in `sync_data/settings.yaml`.
- `sync_data` reads `settings.yaml` from its own directory (`sync_data/core/conf.go`).
- `backend` reads `../sync_data/settings.yaml` (`backend/core/conf.go`), so run backend commands from the `backend/` directory to keep relative config resolution correct.
- `sync_data` writes refreshed Jumia tokens back into YAML via `core.SetYaml()`.

## High-level architecture

### End-to-end data flow

1. `sync_data/service/cron_ser/*` pulls external platform data (orders, order items, transactions, inventory, logistics, token refresh) and writes to MySQL.
2. Sync jobs also update `service_status` rows (`models.ServiceStatus`) to report health/state.
3. `backend/api/service_api/*` reads those status rows and business tables to serve dashboards and operational endpoints.
4. `frontend/src/view/*` calls backend `/api` endpoints and renders admin/user dashboards, tables, and CSV upload workflows.

### Backend layering

- Router composition: `backend/routers/enter.go` mounts `/api`, then `UserRouter` and `ServiceRouter`.
- API entry aggregation: `backend/api/enter.go` with `ApiGroupApp`.
- Auth guards: `backend/middleware/jwt_auth.go` (`JwtAuth`, `JwtAdmin`) parse JWT from header `token` (not `Authorization`).
- Response envelope: `backend/models/res/response.go` standardizes `{code,data,msg}`.
- Query utility: `backend/service/common/service_list.go` provides shared paginated filtering (`ComList`) used by many user APIs.

### Frontend auth/routing pattern

- Login page posts to `/user/login` and stores JWT in Pinia (`frontend/src/view/login.vue`, `frontend/src/stores/userStore.js`).
- Store decodes JWT and maps role claims to `admin`/`user`.
- Router guard in `frontend/src/router/index.js` enforces `meta.requiresAuth` and redirects unauthenticated users to `/login`.
- Axios interceptor in `frontend/src/utils/request.js` sends JWT in header `token`.

### Dashboard and status integration

- Admin/user dashboard endpoints are in `backend/api/service_api/dashboard.go` and `user_dashboard.go`.
- Frontend home page (`frontend/src/view/home.vue`) switches between admin pie charts and user KPI cards based on role.
- Service health endpoint (`GET /api/service`) reads `service_status`; worker updates these IDs during sync jobs.

## Platform-Specific Notes

- Kilimall seller-order crawling baseline uses browser-captured request to `GET https://seller-api.kilimall.ke/order-list` with query keys: `limit`, `pagination`, `orderStatus`, `returnSkus`, `regionId`, `regionCode`, `timeType`, `startTime`, `endTime`.
- In code, this is implemented in `sync_data/service/cron_ser/sync_kilimall_logistics.go` using a lightweight `net/http` client (no browser automation), with delay+jitter between requests and retry/backoff on transient failures.
- Kilimall auth is header+cookie based (not OAuth bearer): set `accesstoken` header plus `Cookie` (especially `seller-sid=...`) from `sync_data/settings.yaml` under `kilimall.auth_token` and `kilimall.cookie`.
- Required request headers include at least `accept`, `accept-language`, `origin`, `referer`, `kili-language`, `request-nonce`, `user-agent`, and auth headers above; content type is not needed for this GET endpoint.
- When HTTP 401/403 or auth-related API errors appear, sync marks `service_status(id=13)` as `错误` so dashboard can prompt manual YAML credential refresh.
- Kilimall parser now uses strong typed mapping for `data.orders[]` + `orders[].skus[]`, populating frontend-relevant fields into `models.OrderItem` (`order_number`, `tracking_number`, `status`, `product_name`, `seller_sku`, `image_url`, pricing fields).
- Kilimall sync also upserts parent rows into `models.Order` (`orders` table), so existing backend `/api/user/order` aggregation logic can attach nested `orderItems` without missing-parent issues.

## Important project-specific gotchas

- `frontend/src/utils/request.js` currently points `baseURL` to `http://localhost:8080/api` (local dev).
- CORS in `backend/routers/enter.go` allows `http://localhost:5173` and any `chrome-extension://` origin.
- `sync_data/main.go` currently executes `SyncUserProductInventoryForone()` and blocks; scheduled cron startup (`CronInit`) is present in code but commented out.
- There are currently no committed `_test.go` files in this repository; `go test` commands are still the canonical way to run tests when tests are added.
- Kilimall logistics crawler scaffold exists at `sync_data/service/cron_ser/sync_kilimall_logistics.go` and reads auth from `sync_data/settings.yaml` under `kilimall` (`cookie`, `auth_token`, `base_url`, `logistics_api`, `page_size`, `max_retries`, `delay_ms`).
- Kilimall 401/403 is treated as auth expiry and updates `service_status` (id=13) to `错误`, so dashboard can prompt manual YAML credential refresh.
- Current Kilimall sync writes normalized logistics fields into `models.OrderItem` (upsert by `id`); no `models.LogisticsRecord` type is currently present in repository.
- Kilimall cost-sheet sync is in `sync_data/service/cron_ser/sync_kilimall_cost_sheets.go`; run with `go run . -kilimall-cost`. Pagination uses `pagination=N&skip=(N-1)*limit&limit=N` (not `page=`).
- `models.Order` has `TotalShippingCost` and `NetProfit` fields (added 2026-04-24); `NetProfit = TotalAmountLocalValue - TotalShippingCost`, recalculated after each cost-sheet sync.
- `POST /api/system/kilimall-cookie` updates `kilimall.cookie` and `kilimall.auth_token` in `settings.yaml` via YAML AST and resets `service_status(id=13)` to "正常". Protected by `LocalOnly()` middleware (127.0.0.1 only, no JWT required).
- Chrome extension at `kilimall-token-grabber/` (Manifest V3) silently intercepts `accesstoken` header and `seller-sid` cookie from `*.kilimall.ke` requests and auto-POSTs to the above endpoint. Uses `chrome.storage.session` for dedup across Service Worker restarts.
- `backend/middleware/local_only.go` provides `LocalOnly()` middleware restricting access to 127.0.0.1/::1 only.
- `frontend/src/App.vue` polls `GET /api/service` every 30 seconds when Kilimall auth alert is visible; stops polling once alert clears.
