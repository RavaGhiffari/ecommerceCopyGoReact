# PLAN.md — Phases 0–2

Status: **awaiting developer approval.** Nothing below is implemented yet except the Docker pieces noted in Phase 0. Rules and conventions live in [CLAUDE.md](CLAUDE.md).

## Open decisions (need a yes/no or an override)

1. **Gin** (newer brief) instead of Chi — OK?
2. **Cookie sessions + Argon2id + CSRF** instead of JWT — OK? (More moving parts than JWT; better security story and a good learning topic.)
3. **Product price vs variant price.** Plan: every product has ≥1 variant; the *variant* price is the authoritative price for cart/checkout, and the product list shows the lowest active variant price. `products` has no price column of its own. OK?
4. **Inventory semantics.** Plan: checkout moves stock `available → reserved`; successful payment consumes the reservation; failure/cancel/expiry releases it.
5. **Repo restructure.** Move the Go spike into `backend/` (`cmd/api`, `internal/...`), delete the demo routes/`data`/`repository`/`handler` spike packages, and move the root `Dockerfile` to `backend/Dockerfile`. OK to delete the spike?
6. Do you want the **React dev server in Compose** now, or keep running the frontend with `npm run dev` on the host during Phases 0–2? (Recommended: host, add the Compose service later.)

---

## Phase 0 — Foundation

Already done: root `Dockerfile`, `docker-compose.yml` (Postgres 18 + API, port 5433), `.env.example`, `.gitignore`.

| # | Work | Files |
|---|---|---|
| 0.1 | Restructure Go code under `backend/`; `go mod init` stays `goFrameworkFirstTry` (or rename — decide); add Gin, pgx, godotenv-free env reading | `backend/cmd/api/main.go`, `backend/go.mod` |
| 0.2 | Config from env vars (port, DB URL, pool sizes, CORS origins, cookie settings, app env) with validation on startup | `internal/config/config.go` |
| 0.3 | `pgxpool` factory with min/max conns, lifetime, health check, ping on start | `internal/database/postgres.go` |
| 0.4 | Response helpers (`success`/`error` envelope, pagination meta) and error-code type | `internal/platform/response/` |
| 0.5 | Logger (JSON `slog`), request ID, recovery, CORS, timeout middleware; graceful shutdown | `internal/platform/logger/`, `internal/middleware/`, `main.go` |
| 0.6 | `GET /health` (liveness) and `GET /ready` (DB ping) | `internal/router/` |
| 0.7 | golang-migrate + Makefile (`migrate-up/down/version/create`), first no-op or `pgcrypto`/`citext` extensions migration | `Makefile`, `db/migrations/000001_*` |
| 0.8 | `sqlc.yaml` + one trivial query to prove generation works | `sqlc.yaml`, `db/queries/` |
| 0.9 | Frontend: `VITE_API_BASE_URL`, status page calling `/health`/`/ready` with loading/error states; remove spike `App.tsx` logic | `frontend/src/...` |
| 0.10 | Update `docker-compose.yml`/Dockerfile for `backend/` paths; add `migrate` workflow | compose, `backend/Dockerfile` |

**Acceptance:** `docker compose up -d --build` starts Postgres + API; `/health` and `/ready` respond; `make migrate-up` then `migrate-down` work on a clean DB; `sqlc generate` succeeds; frontend shows live API status; `go build/vet/test ./...` and `npm run build && npm run lint` pass.

## Phase 1 — Identity & security

**Migrations:** `users` (UUID, unique email, `status` CHECK active/suspended), `roles` (seeded customer/seller/admin), `user_roles`, `user_sessions` (token hash, expiry, `user_id` index).

**Backend (`modules/auth`, `modules/users`, `middleware`):**
- Argon2id hashing (params in config), register (always creates `customer`; role never from client), login (constant-time-ish failure message, session rotation), logout (deletes session row), `GET/PATCH /me`.
- Session token: 32 random bytes → base64url in cookie; SHA-256 hash stored. Cookie `HttpOnly`, `Secure` in prod, `SameSite=Lax`.
- CSRF: double-submit token (or `Origin`/header check) on non-GET requests.
- Middleware: authenticate (loads user + roles into context), `RequireRole(...)`, rate limit on `/auth/*`.
- Admin-seeding path (CLI flag or migration-free seed command) so an admin can exist — **needs your decision**.

**Tests:** hashing, register/login/logout flows, expired/invalid session, suspended user rejected, protected route returns 401, wrong role returns 403, CSRF rejection, rate limit 429. Repository tests against real Postgres.

**Frontend:** login/register pages (RHF + Zod), auth context via TanStack Query on `/me`, header user state, route guards (UX only).

**Acceptance:** per CLAUDE.md §14 Phase 1 — no plaintext/hashes in responses, protected endpoints reject anonymous callers, roles enforced server-side, logout invalidates the session.

## Phase 2 — Catalog (stores, categories, products, inventory basics)

**Migrations:** `stores` (unique slug, unique `owner_id`), `categories` (self-FK `parent_id`, unique slug), `products` (store_id, category_id, slug unique per store or global — decide, `status` active/archived, search `tsvector` + GIN index, `pg_trgm` index on name), `product_images`, `product_variants` (SKU unique, `price NUMERIC(12,2) CHECK >= 0`), `inventory` (`CHECK available_quantity >= 0 AND reserved_quantity >= 0`), `inventory_movements`.

**Backend:**
- `stores`: `POST /seller/stores` (promotes user to `seller` role in the same transaction), `GET/PATCH /seller/store`, public `GET /stores/:slug`.
- `categories`: public list/tree; admin create/update.
- `products`: seller CRUD scoped by `store_id` taken from the session user's store (never from the request body); archive instead of hard-delete; public list with pagination, `category`, `sort=price_asc|price_desc|newest`, `q` keyword; public detail; `related`.
- `inventory`: seller sets stock via adjustment endpoint that writes an `inventory_movements` row in the same transaction.
- Ownership checks in the SQL (`WHERE id = $1 AND store_id = $2`) so a foreign ID returns 404.

**Tests:** seller A cannot read/update/archive seller B's product; slug uniqueness conflict → 409; pagination/filter/sort correctness; negative stock rejected by DB constraint; inventory adjustment writes movement atomically; search returns expected results.

**Frontend:** seller store setup and product form (preparation for Phase 3 UI); public listing page consuming the real API.

**Acceptance:** per CLAUDE.md §14 Phase 2.

## Working method for every step

For each numbered item: state the files to change → implement → run `gofmt`, `go vet`, tests, `npm run build/lint` → report results honestly → propose a commit message. I won't commit; you decide.

## Risks to watch

- Docker CLI not on my shell PATH — I'll ask you to run compose/migrate commands (or give me a working path).
- Local PG18 on 5432 vs Docker on 5433 — tests must target the Compose DB explicitly.
- Argon2id parameters can make tests slow — use lower cost in test config only.
- Cookie + CSRF across `localhost:5173` ↔ `localhost:8081` needs a correct CORS/credentials setup (or keep using the Vite proxy so everything is same-origin in dev).
