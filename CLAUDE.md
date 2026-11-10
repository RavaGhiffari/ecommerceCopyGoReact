# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# E-Commerce Platform — Go, React & PostgreSQL

## 1. Project goal

Build a full-stack **multi-vendor e-commerce platform** with a Tokopedia-inspired shopping experience (UX and information density only — never copy Tokopedia logos, branding, or assets; use an original identity). Buyers browse/search products, manage a cart and check out; many independent sellers each manage their own store, products, inventory and orders; admins moderate the platform.

This is a **serious engineering learning project, not a UI clone.** The developer is learning Go, backend engineering and API architecture. Priorities, in order: clean architecture, data integrity, security, maintainability, testability, scalability without premature complexity.

Prefer explicit, readable code and clear SQL over generic frameworks. For every important implementation, explain: what problem it solves, why this approach, what alternatives exist, the trade-offs, how correctness is verified — and for backend flows, how a request travels handler → service → repository → SQL, why a given constraint/transaction exists, and what failure cases exist. The goal is understanding, not code volume.

## 2. Current state of the repo

Mid-way through Phase 0. **Inspect before overwriting; do not assume the repo is empty.**

- `git` repo initialised. Local PostgreSQL 18 on the dev machine already occupies host port 5432.
- Go spike at the **repo root** (module `goFrameworkFirstTry`, go 1.27.1): `main.go`, `data/`, `repository/`, `handler/`, using `julienschmidt/httprouter` with an in-memory user list. Throwaway — to be replaced by the structure in §4 (moved under `backend/`).
- `frontend/` is the stock Vite + React 19 + TS + Tailwind 4 + oxlint template; `App.tsx` just fetches `/api/users` through the Vite proxy (`/api` → `http://localhost:8081`). Reuse this scaffold, don't regenerate it.
- Docker: root `Dockerfile` (multi-stage Go build, alpine runtime, builds `.` — path must change when code moves to `backend/`), `docker-compose.yml` (services `postgres:18` + `api`; named volume `pgdata` mounted at `/var/lib/postgresql` as PG18 requires; host port `POSTGRES_HOST_PORT` default **5433**), `.env.example`, `.gitignore`. The developer's `.env` holds real credentials — never read it into output, never commit it.
- Not yet existing: config loader, DB connection, migrations, sqlc, Makefile, tests, health endpoints, frontend Dockerfile.
- The shell used by Claude may not have `docker` on PATH; if so, ask the developer to run Docker commands rather than guessing.

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

Currently working:
- `docker compose up -d --build` / `docker compose down` / `docker compose logs -f` (never `down -v` casually — it deletes the database volume)
- Backend spike (repo root): `go run .`, `go build ./...`, `go vet ./...`, `go test ./...`
- Frontend (`cd frontend`): `npm run dev`, `npm run build` (`tsc -b && vite build`), `npm run lint` (**oxlint**, not ESLint)

Planned Makefile targets (Phase 0): `migrate-up`, `migrate-down`, `migrate-version`, `migrate-create name=...`, plus `sqlc` generate, test and run helpers. Single Go test: `go test ./internal/modules/<pkg> -run TestName`.

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

0. **Foundation**: restructure into `backend/`, config, pgxpool, golang-migrate + Makefile, sqlc, `/health` + `/ready`, response helpers, Docker Compose (partly done), frontend↔backend connectivity.
1. **Identity & security**: register/login/logout, Argon2id, sessions, auth middleware, RBAC, CSRF, authorization tests.
2. **Catalog**: categories, stores, products, variants, images, inventory basics, public listing, search/pagination/filter/sort.
3. **Marketplace UI** on real APIs (homepage, nav, cards, detail, store page, search).
4. **Cart & inventory**: cart, stock validation, movements, reservation.
5. **Checkout & orders**: addresses, shipping sim, transactional multi-store checkout, order lifecycle, payment simulation, cancellation + stock release.
6. **Reviews & frontend completion**: reviews, buyer/seller/admin dashboards.
7. **Hardening**: integration + E2E tests, security review, performance/index review, OpenAPI docs, deploy prep.

Scope status: Phases 0–2 have been requested by the developer, but **implementation starts only after the developer approves [PLAN.md](PLAN.md)**. Do not begin Phase 3+ without approval.

## 15. Where the two source briefs disagreed (resolved — newer brief wins)

Router Chi→**Gin**; migrations Goose→**golang-migrate**; auth JWT→**cookie sessions**; hashing bcrypt/Argon2id→**Argon2id**; health `/api/v1/health`→**`/health` + `/ready`**; response envelope gained `success`; role "buyer"→**customer**; stock on `products`→**`inventory` + movements per variant**; single order per checkout→**kept multi-vendor `checkouts` → per-store `orders`** (old brief's rule, not in the new one). If you hit another conflict, flag it rather than silently picking.

## 16. Working agreement

- Before implementing: inspect the existing code, identify affected files, explain the approach and its security/regression implications, and **wait for approval** on substantial changes. Don't overwrite existing config without inspecting it.
- During: small coherent steps, scoped to the requested feature, follow existing conventions, migrations for schema, backend validation, tests for business-critical behaviour, no unnecessary abstractions or dependencies, never hardcode secrets, never bypass security mechanisms.
- After: review the diff, run formatting/lint/static analysis and relevant tests, check migrations up/down, check the frontend compiles, verify real API behaviour, list remaining risks, summarise modified files. **Never claim done because it compiles; never silently skip a failing test — report failures with their output.**
- **Never commit, push, or deploy automatically**; never touch production infrastructure or expose credentials. When a feature is complete, propose a commit message and let the developer decide. Explain any destructive database operation before running it.
