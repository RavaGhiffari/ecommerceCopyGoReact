# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# E-Commerce Platform — Go, React & PostgreSQL

## 1. Project goal

Build a full-stack **multi-vendor e-commerce platform** with a Tokopedia-inspired shopping experience (UX and information density only — never copy Tokopedia logos, branding, or assets; use an original identity). Buyers browse/search products, manage a cart and check out; many independent sellers each manage their own store, products, inventory and orders; admins moderate the platform.

This is a **serious engineering learning project, not a UI clone.** The developer is learning Go, backend engineering and API architecture. Priorities, in order: clean architecture, data integrity, security, maintainability, testability, scalability without premature complexity.

Prefer explicit, readable code and clear SQL over generic frameworks. For every important implementation, explain: what problem it solves, why this approach, what alternatives exist, the trade-offs, how correctness is verified — and for backend flows, how a request travels handler → service → repository → SQL, why a given constraint/transaction exists, and what failure cases exist. The goal is understanding, not code volume.

## 2. Current state of the repo

**Phase 0 done. Catalog UI prototype (0.5) done on mock data. Phase 1 (identity & security) implemented and awaiting developer review** — do not start Phase 2 until approved. **Inspect before overwriting; do not assume the repo is empty.**

- Backend in `backend/` (module `github.com/RavaGhiffari/ecommerceCopyGoReact/backend`, go 1.27.3, Gin, pgx v5, google/uuid, x/crypto, testify). Request path: `internal/router` (middleware chain + routes) → `internal/modules/<module>/handler.go` → `service.go` → `repository.go` → sqlc code in `internal/database/queries` (generated from `db/queries/*.sql`; never edit by hand) → pgxpool from `internal/database`. `cmd/api/main.go` wires everything and does graceful shutdown. Middleware order: RequestID → Logging → Recovery → CORS → Timeout → SameOrigin (CSRF) → Authenticate; per-route `RequireAuth` / `RequireRole`. `internal/platform/logger` adds `request_id` to every `slog.*Context` log line.
- Modules: `health` (`/health`, `/ready`; no service, no rules), `auth` (register/login/logout, sessions, rate limits), `users` (profile, `GET`/`PATCH /api/v1/me`). Shared helpers in `internal/platform/`: `password` (Argon2id PHC strings), `token` (random session token + SHA-256), `ratelimit` (in-memory fixed window — single instance only), `validation` (`Errors` → 422 with field details), `authctx` (Principal in context), `request` (strict JSON: application/json only, unknown fields rejected, 1 MiB cap), `response`. Transactions: `database.WithTx`; services decide the boundary, repositories take `queries.DBTX` so they work with a pool or a tx. Services return plain errors / `validation.Errors`; handlers map them to HTTP.
- Auth design (Phase 1 decisions): opaque 32-byte token in cookie `sid` (HttpOnly, SameSite=Lax, Secure in production), only SHA-256 stored in `user_sessions`, TTL `SESSION_TTL` (24h); register logs in; login rotates the session and resets the per IP+email failure counter (5 per 15 min); register limited to 10/hour per IP; suspended users are filtered in the session lookup; unknown email and wrong password give the same 401 (dummy hash equalizes timing); CSRF = SameSite + Origin-vs-Host/allow-list check + JSON-only bodies (no double-submit token). Admins only via `make seed-admin`. There is no `/auth/me`; use `GET /api/v1/me`.
- Migrations: `backend/db/migrations`, golang-migrate, via compose services `migrate` (dev DB) and `migrate-test` (`marketplace_test`), profile `tools`. `000001` `set_updated_at()`, `000002` users (lowercase-email CHECK + unique `LOWER(email)` index), `000003` roles (seeded customer/seller/admin) + user_roles, `000004` user_sessions. **The developer's dev database is named `main`** (from `.env`), not `marketplace_dev`.
- Frontend: Vite + React 19 + TS + Tailwind 4 + oxlint, plus react-router-dom, lucide-react, react-hook-form, zod 4, @hookform/resolvers. `src/lib/api.ts` (`apiRequest` for all methods, envelope-aware, `ApiError.fieldErrors()`), `src/types/api.ts` and `src/types/auth.ts` (mirrors of Go DTOs — keep in sync). **Real API:** auth via `src/services/auth.ts` → `features/auth/AuthProvider` (`useAuth()`), pages `/login`, `/register`, `/account` (guarded by `RequireAuth`, UX only), `/status`. **Mock data (§17):** catalog via `src/services/catalog.ts` (the only importer of `src/mocks/`). Vite proxies `/api` (changeOrigin **false**, required by the CSRF Origin/Host check) and exactly `/health`, `/ready` to `localhost:8081`.
- Docker: `backend/Dockerfile` (builds every `cmd/*`: `/app/api`, `/app/seed-admin`), `docker-compose.yml` (`postgres:18` with volume `pgdata` at `/var/lib/postgresql` as PG18 requires, host port **5433**; `api`; `migrate`; `migrate-test`). The developer's `.env` holds real credentials — never print it, never commit it.
- Old httprouter spike deleted (recoverable from commit `e402892`).
- **Host machine quirks:** Go is **not installed** on the host — run Go/sqlc via Docker (the Makefile does this). GNU make 4.4.1 is installed via scoop; the Makefile sets `MSYS_NO_PATHCONV=1` because Git Bash otherwise rewrites `-w /src` into a Windows path. Docker CLI: `C:\Users\Rava\AppData\Local\Programs\DockerDesktop\resources\bin\docker.exe`. Local PostgreSQL 18 uses host port 5432. **Another app of the developer's (a Laravel project) also uses port 5173 on IPv4** — `localhost:5173` may hit it instead of this Vite server (which binds `[::1]`). Never kill a process by port without confirming via its command line that it belongs to this repo.

The detailed, phase-by-phase work plan lives in [PLAN.md](PLAN.md).

## 3. Technology stack

| Layer | Choice |
|---|---|
| Backend | Go, **Gin**, pgx (`pgxpool`), **sqlc**, PostgreSQL 18 |
| Migrations | **golang-migrate** (`NNNNNN_name.up.sql` / `.down.sql`), driven by a Makefile |
| Auth | Server-side **cookie sessions**, **Argon2id** password hashing, CSRF protection |
| API | REST JSON under `/api/v1`, OpenAPI/Swagger docs (tooling chosen in hardening phase) |
| Frontend | React + TypeScript, Vite, Tailwind, shadcn/ui, Lucide, React Router, TanStack Query, React Hook Form + Zod; Zustand only if shared client state is truly needed; fetch or Axios |
| Tests | Go `testing` + Testify; Vitest + React Testing Library; Playwright E2E |
| Infra | Docker + Docker Compose (Postgres, API, React dev server); modular monolith |

No microservices, message brokers, Kubernetes, Elasticsearch, distributed transactions, or real payment/shipping gateways without a concrete requirement. Add no dependency without justification.

## 4. Architecture

**Modular monolith**, organised by business domain. Dependency direction: `HTTP Handler → Service → Repository → PostgreSQL`.

- Handlers do HTTP only and never execute SQL.
- Services hold business logic and must **not** depend on Gin types (`*gin.Context`).
- Repositories (sqlc-generated + thin wrappers) hold no business rules.
- Per module: `handler.go`, `service.go`, `repository.go`, `dto.go` (request/response contracts), `model.go`. Add interfaces only where they give real testability or inversion.

Target layout:

```
backend/
├── cmd/api/main.go
├── internal/
│   ├── config/  database/  middleware/  router/ (router.go, routes.go)
│   ├── platform/ (logger/, response/, validator/)
│   └── modules/ auth users stores categories products inventory
│                cart checkout orders payments shipping reviews admin
├── db/migrations/   db/queries/
├── sqlc.yaml  go.mod  Dockerfile  Makefile
frontend/
└── src/ app/ components/ features/(auth products cart checkout orders seller …)
         hooks/ lib/ services/ types/ routes/     + Dockerfile
docker-compose.yml  .env.example  .gitignore  CLAUDE.md  PLAN.md
```

Frontend is organised by feature, not one big `components/`. Never pile logic into a single page or component.

## 5. Commands

From the repo root (Makefile wraps Docker):
- Stack: `make up` (`docker compose up -d --build`), `make down` (keeps the DB volume — never `down -v` casually), `make logs`, `make ps`
- Migrations: `make migrate-up`, `make migrate-down` (one step), `make migrate-version`, `make migrate-create name=create_users_table`, `make migrate-force version=N` (dirty DB only). Raw form: `docker compose run --rm migrate up`.
- Code: `make sqlc` (regenerate after editing `db/queries` or migrations), `make fmt`, `make vet`, `make test` (unit, no DB), `make test-db` (create + migrate `marketplace_test`), `make test-integration` (runs `test-db`, then build tag `integration` against `marketplace_test`; the tests TRUNCATE users and refuse any URL without `_test`)
- Accounts: `make seed-admin email=... name="..."` (password prompted, or `ADMIN_PASSWORD` env; needs the api image from `make up`)
- Single Go test: `docker run --rm -v <repo>/backend:/src -v gomodcache:/go/pkg/mod -w /src golang:1.27-alpine go test ./internal/middleware -run TestCORS -v`
- Smoke check: `curl http://localhost:8081/health` and `/ready`
- Frontend (`cd frontend`): `npm run dev`, `npm run build` (`tsc -b && vite build`), `npm run lint` (**oxlint**, not ESLint)

## 6. Roles and authorization

Roles (RBAC via `roles` + `user_roles`): **customer**, **seller**, **admin**.

- Authorization is **always enforced on the backend**; frontend route guards are UX only. Never trust a client-submitted role, price, or total.
- Authentication = who are you; authorization = what may you access. Check **resource ownership** on every protected operation: a seller touches only their own store, products, inventory and seller orders; a customer sees only their own cart and orders. Changing an ID in a request must never cross those boundaries.
- A seller owns at most one store (revisit explicitly if that changes).
- Customer: register/login, browse/search, cart, checkout, order history/tracking, reviews after eligible purchase, profile and addresses.
- Seller: create/update store, product CRUD and archive, inventory, incoming orders and fulfilment status, basic store stats.
- Admin: users/sellers, categories, product moderation, stores, platform orders, suspend accounts/stores, basic stats, audit logs.

## 7. Domain rules that must hold

- **Passwords**: Argon2id, never plaintext, never in any response or log.
- **Sessions**: opaque random token in an `HttpOnly`, `Secure` (production), `SameSite=Lax`-or-stricter cookie; only the **hash** is stored in `user_sessions`; rotate on login; invalidate server-side on logout; CSRF protection on state-changing requests; rate-limit login/registration.
- **Cart is not an inventory reservation.** Validate stock when adding, re-validate at checkout. Cart shows current prices.
- **Checkout is one PostgreSQL transaction**: validate user → read cart → lock/condition-update inventory → validate availability → compute prices **from server data** → create records → reserve inventory + write `inventory_movements` → clear purchased cart items → commit; any failure rolls everything back. Duplicate submissions must not create duplicate orders (idempotency key).
- **Multi-store**: one cart may span stores. A **checkout** is the parent; it contains one **order per store** (seller order). The customer sees the whole checkout; each seller can only manage their own order.
- **Inventory** lives in `inventory` (per variant: `available_quantity`, `reserved_quantity`); it can never go negative; every change writes an `inventory_movements` row (auditable). Prevent overselling with conditional atomic updates (`... WHERE available_quantity >= $n`) and/or row locks; verify with concurrency tests. Cancellation/expiry releases reserved stock.
- **History is immutable**: order items snapshot product name, SKU and unit price; shipping info is snapshotted on the order; never recompute historical totals from current prices. Deleting/archiving a product must not destroy order history.
- **Order statuses**: `pending_payment, paid, processing, shipped, delivered, cancelled, refunded`, with `order_status_histories` and validated transitions; unique human-readable `order_number`.
- **Payments are simulated** (payment reference, simulate success/failure, expiry), structured so Midtrans/Xendit can slot in. Payment events are processed **idempotently**; duplicate payment processing must be safe; webhook signatures validated when real gateways arrive.
- **Shipping is simulated** (Regular / Express / Same Day via configurable mock rules); no external shipping APIs.
- **Reviews**: rating 1–5, optional comment, only for products the customer actually purchased, one review per eligible purchase, average rating exposed.
- **Search**: PostgreSQL full-text or trigram; listings paginate and support category filter, price sort, keyword search.

## 8. Database conventions

- **UUID primary keys** throughout; foreign keys; unique/check constraints; `NOT NULL` where appropriate; indexes driven by real query patterns; avoid N+1.
- **`NUMERIC` for money — never floats.** `TIMESTAMPTZ`, stored in UTC.
- Schema changes **only through migrations**, each with a working down migration; never edit an applied migration in a shared environment; avoid destructive migrations without a data strategy; test up/down against a clean Postgres.
- Entities, introduced progressively per phase (don't build all at once): `users, roles, user_roles, user_sessions, user_addresses, stores, categories (parent_id), products, product_images, product_variants, inventory, inventory_movements, carts, cart_items, checkouts, orders, order_items, order_status_histories, payments, payment_events, reviews`; later `shipping_methods, shipments, coupons, audit_logs`.
- sqlc generates type-safe code from `db/queries/*.sql`; no generic ORM abstraction. Use `pgxpool` (configurable max/min conns, lifetime, timeout, health checks) and pass `context.Context` everywhere. Parameterised SQL only.

## 9. API conventions

Base path `/api/v1`. Health endpoints `GET /health` and `GET /ready` (outside the prefix; readiness checks the DB; don't leak infra details).

Route groups: public (`auth/register|login|logout`, `products`, `products/:id/related`, `products/:id/reviews`, `categories`, `stores/:slug`), customer (`me`, `cart`, `cart/items`, `checkout`, `checkout/:id`, `orders`, `orders/:id`, `orders/:id/cancel`, `payments/:id/simulate`, `reviews`), seller (`seller/stores`, `seller/store`, `seller/products`, `seller/orders`, `seller/orders/:id/status`, `seller/inventory`), admin (`admin/users`, `admin/users/:id/status`, `admin/stores`, `admin/orders`, `admin/audit-logs`). Endpoint details may be adjusted, but keep REST conventions and ownership rules.

Responses:

```json
{ "success": true,  "data": {}, "message": "Product retrieved successfully" }
{ "success": false, "error": { "code": "PRODUCT_NOT_FOUND", "message": "…", "details": {} } }
```

`details` is optional (validation field errors). List responses add pagination metadata. Status codes: 200, 201, 204, 400, 401, 403, 404, 409, 422, 429, 500. Never return raw SQL errors, stack traces or internals. Keep frontend TypeScript types aligned with these contracts (the hand-mirrored `User` interface in the spike `App.tsx` is the drift to avoid).

## 10. Middleware, security, observability

Central middleware: request ID, structured JSON request logging, panic recovery, CORS (restrictive origins), authentication, authorization, rate limiting, request size limits, security headers, timeouts. Also: validate everything server-side (lengths, UUIDs, ranges, emails, enums), prevent SQLi/XSS, never expose secrets to React or logs, never commit `.env`, keep security-relevant audit records. Graceful shutdown; basic metrics; correlation IDs.

## 11. Frontend expectations

Reads as a real marketplace, not an admin dashboard. Nav with original logo, search, category navigation, cart, login/register, account dropdown, seller entry point. Homepage: banner/carousel, category shortcuts, popular + recommended, product grid, footer. Product cards: image, name, price, store name, location, rating, sold count. Product detail (gallery, price, reviews, store, stock, quantity, add to cart, buy now, related), cart grouped by store, checkout (address, per-seller groups, shipping, summary, simulated payment), customer account, seller dashboard, admin dashboard. Routes: `/`, `/products`, `/products/:slug`, `/categories/:slug`, `/stores/:slug`, `/cart`, `/checkout`, `/orders`, `/orders/:id`, `/login`, `/register`, `/seller/*`, `/admin/*`. Desktop-first but responsive; always implement loading, empty and error states; never hardcode product data.

## 12. Docker environment

Compose runs PostgreSQL (healthcheck, named volume, configurable host port), the Go API (waits for DB health), and the React dev server. No hardcoded secrets; separate dev/prod configuration; `backend/Dockerfile` and `frontend/Dockerfile`. Normal shutdown must not delete the DB volume.

## 13. Testing

First-class requirement. Backend: service unit tests, repository integration tests against real PostgreSQL (not SQLite), handler, middleware, authentication, authorization, transaction-rollback, checkout, and inventory-concurrency tests. Frontend: form validation, product rendering, cart interactions, auth states, API errors, loading/empty states. E2E (Playwright): register, login, discovery, add to cart, checkout, payment simulation, order history, seller product + order management, unauthorized access. Every bug fix gets a regression test.

## 14. Roadmap — one phase at a time

0. **Foundation**: restructure into `backend/`, config, pgxpool, golang-migrate + Makefile, sqlc, `/health` + `/ready`, response helpers, Docker Compose, frontend↔backend connectivity. *(Done and reviewed.)*
0.5. **UI-first catalog prototype** (current, §17): homepage, product list/search, product detail, store page on typed mock data. No backend changes.
1. **Identity & security**: register/login/logout, Argon2id, sessions, auth middleware, RBAC, CSRF, authorization tests.
2. **Catalog**: categories, stores, products, variants, images, inventory basics, public listing, search/pagination/filter/sort.
3. **Marketplace UI** on real APIs (homepage, nav, cards, detail, store page, search).
4. **Cart & inventory**: cart, stock validation, movements, reservation.
5. **Checkout & orders**: addresses, shipping sim, transactional multi-store checkout, order lifecycle, payment simulation, cancellation + stock release.
6. **Reviews & frontend completion**: reviews, buyer/seller/admin dashboards.
7. **Hardening**: integration + E2E tests, security review, performance/index review, OpenAPI docs, deploy prep.

Scope status: Phases 0, 0.5 (catalog prototype) and 1 are implemented; Phase 1 awaits developer review. Each later phase starts only after explicit developer approval; stop at the end of each phase for review.

## 15. Where the two source briefs disagreed (resolved — newer brief wins)

Router Chi→**Gin**; migrations Goose→**golang-migrate**; auth JWT→**cookie sessions**; hashing bcrypt/Argon2id→**Argon2id**; health `/api/v1/health`→**`/health` + `/ready`**; response envelope gained `success`; role "buyer"→**customer**; stock on `products`→**`inventory` + movements per variant**; single order per checkout→**kept multi-vendor `checkouts` → per-store `orders`** (old brief's rule, not in the new one). If you hit another conflict, flag it rather than silently picking.

## 16. Working agreement

- Before implementing: inspect the existing code, identify affected files, explain the approach and its security/regression implications, and **wait for approval** on substantial changes. Don't overwrite existing config without inspecting it.
- During: small coherent steps, scoped to the requested feature, follow existing conventions, migrations for schema, backend validation, tests for business-critical behaviour, no unnecessary abstractions or dependencies, never hardcode secrets, never bypass security mechanisms.
- After: review the diff, run formatting/lint/static analysis and relevant tests, check migrations up/down, check the frontend compiles, verify real API behaviour, list remaining risks, summarise modified files. **Never claim done because it compiles; never silently skip a failing test — report failures with their output.**
- **Never commit, push, or deploy automatically**; never touch production infrastructure or expose credentials. When a feature is complete, propose a commit message and let the developer decide. Explain any destructive database operation before running it.

## 17. UI-first catalog prototype (current, phase 0.5)

The developer chose to see the marketplace interface before backend business logic. This is an intentional prototyping strategy, **not a relaxation of production standards**. It ends when the developer approves the prototype.

### Scope

Read-only catalog pages only: homepage, product list with category filter / price sort / keyword search / pagination, product detail, store page, not-found. Cart, checkout, orders, seller and admin screens are **out of scope** until their backend exists. (Login, register and account were in this list; they now use the real Phase 1 API, not mocks.)

### Rules

- Stack and structure stay as in §3/§4: React, TypeScript, Tailwind, `features/`, `components/`, `lib/`, `types/`. New dependencies need justification (react-router-dom for routing, lucide-react for icons are the approved additions).
- **Mock data** is typed fixtures in `frontend/src/mocks/`. **Only `frontend/src/services/` may import them.** Components get typed props or call feature hooks; they never import mock datasets and contain no mock business logic. Replacing mocks with the real API must only touch `services/`.
- Types in `frontend/src/types/` describe the data the real API is expected to return; keep them aligned with §8/§9 when the backend lands.
- Every data-driven view implements **loading, empty, error and success** states. Services accept `?simulate=slow|empty|error` in the URL during the prototype so those states can be checked by hand.
- **Nothing pretends to be a real operation:** no fake auth (no stored tokens, no client-side role gates), no add-to-cart that "works", no checkout, no simulated payment, no stock reservation. Controls for unbuilt features are visibly disabled and say so.
- Product URLs are store-scoped (`/stores/:storeSlug/products/:productSlug`) per the approved decision `UNIQUE (store_id, slug)`.
- No backend changes, no migrations, and the existing `/status` page stays.

### Completion criteria

1. All scoped pages exist with working navigation and routing.
2. Search, filter, sort and pagination work against mock data.
3. Loading, empty and error states exist on every data view.
4. Components are reusable and feature-organised; no component imports from `mocks/`.
5. `npm run build` and `npm run lint` pass.
6. The developer reviews and approves.

Prototype completion does **not** mean production readiness. After approval, resume the backend roadmap at Phase 1 and replace mock services with real API calls one feature at a time.
