# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Starting a new session? Read [SESSION_HANDOFF.md](SESSION_HANDOFF.md) first**, then [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md) (verified state, checkpoints, decision log, known issues). This file holds the rules; those hold the current state.

# E-Commerce Platform — Go, React & PostgreSQL

## 1. Project goal

Build a full-stack **multi-vendor e-commerce platform** with a Tokopedia-inspired shopping experience (UX and information density only — never copy Tokopedia logos, branding, or assets; use an original identity). Buyers browse/search products, manage a cart and check out; many independent sellers each manage their own store, products, inventory and orders; admins moderate the platform.

This is a **serious engineering learning project, not a UI clone.** The developer is learning Go, backend engineering and API architecture. Priorities, in order: clean architecture, data integrity, security, maintainability, testability, scalability without premature complexity.

Prefer explicit, readable code and clear SQL over generic frameworks. For every important implementation, explain: what problem it solves, why this approach, what alternatives exist, the trade-offs, how correctness is verified — and for backend flows, how a request travels handler → service → repository → SQL, why a given constraint/transaction exists, and what failure cases exist. The goal is understanding, not code volume.

## 2. Current state of the repo

**Phases 0, 0.5, 1, 2, 4 and 5 done (committed), and the frontend migration to shadcn/ui is committed (`db15fef`). Phase 6 plan approved (PLAN.md, Q1–Q15 defaults); slices 6a–6d committed; 6e (admin UI) done, awaiting review. Phase 7 (hardening) needs its own plan and approval.** Work slice by slice (6a → 6e) and stop after each for review. **Inspect before overwriting; do not assume the repo is empty.**

- Backend in `backend/` (module `github.com/RavaGhiffari/ecommerceCopyGoReact/backend`, go 1.27.3, Gin, pgx v5, google/uuid, x/crypto, testify). Request path: `internal/router` (middleware chain + routes) → `internal/modules/<module>/handler.go` → `service.go` → `repository.go` → sqlc code in `internal/database/queries` (generated from `db/queries/*.sql`; never edit by hand) → pgxpool from `internal/database`. `cmd/api/main.go` wires everything and does graceful shutdown. Middleware order: RequestID → Logging → Recovery → CORS → Timeout → SameOrigin (CSRF) → Authenticate; per-route `RequireAuth` / `RequireRole`. `internal/platform/logger` adds `request_id` to every `slog.*Context` log line.
- Modules: `health` (`/health`, `/ready`; no service, no rules), `auth` (register/login/logout, sessions, rate limits), `users` (profile, `GET`/`PATCH /api/v1/me`), `categories` (public list; admin `POST /admin/categories`, `PUT /admin/categories/:id`; two levels), `stores` (`POST /seller/store` opens a store AND grants the seller role in one tx; `GET`/`PATCH /seller/store`; public `GET /stores/:slug`), `products` (public `GET /products?q=&category=&store=&sort=popular|newest|price_asc|price_desc&page=&pageSize=`, `GET /stores/:slug/products/:productSlug`, `GET /products/:id/related`; seller CRUD under `/seller/products`, archive via `DELETE`, variants, `PUT .../variants/:variantId/stock`), `addresses` (`/me/addresses` CRUD + `POST /me/addresses/:id/default`; at most 10 per user; one default per user via a partial unique index; every write locks the user row `FOR NO KEY UPDATE`), `cart` (`GET /cart`, `POST /cart/items`, `PATCH`/`DELETE /cart/items/:id`; every mutation locks the cart row in one tx, never touches inventory, returns the whole cart; unbuyable lines are flagged with `issue` and left out of subtotals), `inventory` (functions, not a service: `Create` / `SetAvailable` / `LockMany` / `Reserve` / `Release` / `Consume` / `Restock` must run inside the caller's tx; lock row `FOR UPDATE` → update → ledger row; `LockMany` locks in `variant_id` order), `shipping` (pure functions: flat fees from config, Same Day only when store city = address city), `checkout` (`POST /checkout/quote` read-only; `POST /checkout` needs header `Idempotency-Key` (UUID) and runs one tx: lock cart → idempotency lookup → address by user → `LockMany` → re-read lines → issues 409 / shipping 422 → checkout + per-store orders (snapshots, history) + payment → `Reserve` + ledger → delete purchased cart lines; `GET /checkout/:id`), `orders` (buyer `GET /orders`, `GET /orders/:id`, `POST /orders/:id/cancel` = paid → refunded + restock; `Create` inside the checkout tx; order number `PK-YYYYMMDD-XXXXXX` in WIB, retried via `ON CONFLICT DO NOTHING`; **every status change goes through `Repository.Transition` and the table in `orders/transitions.go`, which also names the allowed actor**). Lifecycle (5c, `checkout/lifecycle.go`): `POST /payments/:id/simulate` `{outcome, eventId}` (event id makes it idempotent), `POST /checkout/:id/cancel`, expiry sweeper `RunSweeper` started in `main.go` (`CHECKOUT_SWEEP_INTERVAL`, `FOR UPDATE SKIP LOCKED`); every change locks the checkout row first, lock order checkout → orders → inventory → products → payment. Seller orders (5d, `orders/service_seller.go`): `GET /seller/orders?status=&page=&pageSize=`, `GET /seller/orders/:id`, `PATCH /seller/orders/:id/status` `{status, note?}`; store from the session, `LockStoreOrder` + `Transition(BySeller)`. Seller stats (6c, `orders/stats.go`): `GET /seller/stats`, computed on read; revenue = `items_subtotal` of `RevenueStatuses` (paid/processing/shipped/delivered), 30-day window on `orders.created_at`; UI `features/seller/SellerStatsPanel` on `/seller`. Admin (6d): module `admin` (`GET /admin/stats`; `/admin/users` and `/admin/stores` with `?q=&status=`; `PATCH …/:id/status` active|suspended; an admin cannot change themselves or another admin; suspending a user deletes their sessions), platform orders in `orders/admin.go` (`/admin/orders`, read-only), moderation in `reviews/admin.go` (`/admin/reviews`, `PATCH …/:id/status` visible|hidden moves the rating counters), `/admin/audit-logs`. **Every admin write calls `audit.Write(ctx, tx, …)` inside the same transaction as the change** (category create/update too); a no-op (same status) writes no row. Admin UI (6e): `/admin/*` children of one route behind `features/admin/RequireAdmin` (UX only) + `AdminLayout`; API calls in `services/admin.ts`, types in `types/admin.ts`. `reviews` (6a): `POST /reviews` (only an item of the caller's own `delivered` order, once: `order_item_id` UNIQUE + `ON CONFLICT DO NOTHING` → 409), `PATCH /reviews/:id` (row locked, counters move by the delta), public `GET /products/:id/reviews` (summary + distribution, masked names), `GET /me/reviews`, `GET /me/reviews/pending`; `products.rating_count/rating_total` change only through `AdjustProductRating` (atomic `x = x + delta`) in the review transaction; averages via `platform/rating.Average`. Seller routes resolve the store from the session and every query filters by that `store_id`, so foreign IDs are 404. Shared helpers in `internal/platform/`: `password` (Argon2id PHC strings), `token` (random session token + SHA-256), `ratelimit` (in-memory fixed window — single instance only), `validation` (`Errors` → 422 with field details; nested keys like `variants[0].price`), `authctx` (Principal in context), `request` (strict JSON: application/json only, unknown fields rejected, 1 MiB cap), `slug` (name → slug; same pattern as the DB CHECK), `response`.
- Catalog design (Phase 2 decisions): money is `NUMERIC(12,2)` ↔ `shopspring/decimal` (codec registered in `database.NewPool` AfterConnect) ↔ JSON **string** with 2 decimals (`"349000.00"`); never float. Prices live on variants; a product is public only if active, its store is active and it has ≥1 variant; listings show min/max variant price. Products must sit on a leaf category. Categories are two levels (service check + DB trigger `categories_max_two_levels`); a parent filter includes its children. Search = PostgreSQL FTS (`search` generated tsvector, `'simple'` config, GIN index) with prefix terms built from sanitised words in `products/search.go`; no `pg_trgm`. Images are http(s) URLs only (placeholder in the UI when none). SKUs are global and upper-cased; product slugs `UNIQUE (store_id, slug)`; store slugs are immutable. `sold_count` is a denormalised counter (checkout will update it); ratings are 0 until reviews exist. Transactions: `database.WithTx`; services decide the boundary, repositories take `queries.DBTX` so they work with a pool or a tx. Services return plain errors / `validation.Errors`; handlers map them to HTTP.
- Auth design (Phase 1 decisions): opaque 32-byte token in cookie `sid` (HttpOnly, SameSite=Lax, Secure in production), only SHA-256 stored in `user_sessions`, TTL `SESSION_TTL` (24h); register logs in; login rotates the session and resets the per IP+email failure counter (5 per 15 min); register limited to 10/hour per IP; suspended users are filtered in the session lookup; unknown email and wrong password give the same 401 (dummy hash equalizes timing); CSRF = SameSite + Origin-vs-Host/allow-list check + JSON-only bodies (no double-submit token). Admins only via `make seed-admin`. There is no `/auth/me`; use `GET /api/v1/me`.
- Migrations: `backend/db/migrations`, golang-migrate, via compose services `migrate` (dev DB) and `migrate-test` (`marketplace_test`), profile `tools`. `000001` `set_updated_at()`, `000002` users (lowercase-email CHECK + unique `LOWER(email)` index), `000003` roles (seeded customer/seller/admin) + user_roles, `000004` user_sessions, `000005` categories (+ two-level trigger), `000006` stores (`owner_id` UNIQUE), `000007` products / product_variants / product_images, `000008` inventory / inventory_movements (CHECK ≥ 0; append-only ledger), `000009` carts / cart_items (one cart per user, quantity 1..99, unique variant per cart), `000010` user_addresses, `000011` checkouts / orders / order_items / order_status_histories / payments / payment_events (+ `restock` ledger type), `000012` reviews + `products.rating_count/rating_total` (invariant: they equal COUNT/SUM of visible reviews), `000013` wishlist_items (practice feature, `/me/wishlist`; PLAN.md §6.4b; the admin `audit_logs` migration is `000014`), `000014` audit_logs (append-only), `000015` reviews `updated_at` trigger only on rating/comment (so moderation is not shown as an edit). Since 5b the invariant also holds for `reserved_quantity` vs `SUM(reserved_delta)`. Invariant: for every variant, `SUM(inventory_movements.available_delta) = inventory.available_quantity`. **The developer's dev database is named `main`** (from `.env`), not `marketplace_dev`. Demo data: `make seed-dev` (sellers `penjual1..4@contoh.id` / `penjual-123`; demo buyer `pembeli.demo@contoh.id` / `pembeli-123` with orders in every status; local only). The catalog step is skipped when categories exist, the orders step when the demo buyer exists.
- Frontend: Vite + React 19 + TS + Tailwind 4 + oxlint, plus react-router-dom, lucide-react, react-hook-form, zod 4, @hookform/resolvers, and **shadcn/ui** (Radix, style `radix-nova`; components in `src/components/ui/`, import via the `@/` alias; brand colour = `--primary` in `src/index.css`; after `npx shadcn add`, fix the generated `from "cn"` import to `@/lib/utils`, see PROJECT_STATE §3). `src/lib/api.ts` (`apiRequest` for all methods, envelope-aware, `ApiError.fieldErrors()`), `src/types/api.ts` and `src/types/auth.ts` (mirrors of Go DTOs — keep in sync). **Real API:** auth via `src/services/auth.ts` → `features/auth/AuthProvider` (`useAuth()`), pages `/login`, `/register`, `/account` (guarded by `RequireAuth`, UX only), `/status`. Catalog via `src/services/catalog.ts` (real API; types in `src/types/marketplace.ts`, prices are `Money` strings). Seller UI via `src/services/seller.ts` + `src/types/seller.ts`: `/seller` (open store or dashboard), `/seller/products/new`, `/seller/products/:id` (guarded by `RequireSeller`, UX only). Cart via `src/services/cart.ts` → `features/cart/CartProvider` (`useCart()`, inside `AuthProvider`) → `/cart` (RequireAuth), header badge, add-to-cart and "Beli sekarang" (add, then `/checkout`) in `ProductDetailView`. Checkout & orders (5e): `src/services/checkout.ts` + `src/services/orders.ts`, types `src/types/checkout.ts` + `src/types/orders.ts`; pages `/checkout` (quote-driven; `Idempotency-Key` once per visit), `/checkout/:id` (payment simulation, countdown), `/orders`, `/orders/:id`, `/seller/orders`, `/seller/orders/:id`; shared parts in `features/orders/` (`status.ts` = labels + UI copy of the transition table). Reviews (6b): `src/services/reviews.ts`, `src/types/reviews.ts`, `features/reviews/` (`ReviewDialog` = create or edit, `StarInput` = native radios, `ProductReviewsSection` on the product page, `AccountReviews` on `/account`), read-only `components/shared/Stars`; buyer order items get a "Beri ulasan" / "Ubah ulasan" action when the order is delivered (`OrderDetailLayout` `itemAction`). `useAsyncResource` lets a page show a mutation's response without reloading; toasts via Sonner (`<Toaster>` in `App.tsx`). `ProductImage` is `w-full`: wrap it in a sized element. Loading states use content-shaped skeletons from `components/shared/Skeletons` (`LoadingRegion` = one `role="status"` label with the bars hidden from screen readers, `ItemRowSkeleton`, `TableSkeleton` with the real column headers, `StatTileSkeleton`, `LineSkeleton`), not single grey blocks. Address book on `/account` (`features/addresses/`, shadcn Dialog + AlertDialog). `CategorySelect` is a Radix Select used through RHF `<Controller>` and is mounted only after categories load. Logout navigates with `flushSync` before clearing the user (PROJECT_STATE §11). `applyServerError` maps API keys like `variants[0].price` to RHF paths `variants.0.price`. Vite proxies `/api` (changeOrigin **false**, required by the CSRF Origin/Host check) and exactly `/health`, `/ready` to `127.0.0.1:8081` (Compose publishes ports on loopback only).
- Docker: `backend/Dockerfile` (builds every `cmd/*`: `/app/api`, `/app/seed-admin`, `/app/seed-dev`), `docker-compose.yml` (`postgres:18` with volume `pgdata` at `/var/lib/postgresql` as PG18 requires, host port **5433**; `api`; `migrate`; `migrate-test`). The developer's `.env` holds real credentials — never print it, never commit it.
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
- Accounts & data: `make seed-admin email=... name="..."` (password prompted, or `ADMIN_PASSWORD` env; needs the api image from `make up`); `make seed-dev` (demo catalog through the real services; refuses production; no-op if categories exist)
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

Route groups: public (`auth/register|login|logout`, `products`, `products/:id/related`, `products/:id/reviews`, `categories`, `stores/:slug`), customer (`me`, `cart`, `cart/items`, `checkout`, `checkout/:id`, `orders`, `orders/:id`, `orders/:id/cancel`, `payments/:id/simulate`, `reviews`), seller (`seller/store` [POST opens a store; any signed-in user], `seller/products` [+ `/:id`, `/:id/variants`, `/:id/variants/:variantId`, `/:id/variants/:variantId/stock`], `seller/orders`, `seller/orders/:id/status`, `seller/inventory`), admin (`admin/users`, `admin/users/:id/status`, `admin/stores`, `admin/orders`, `admin/audit-logs`). Endpoint details may be adjusted, but keep REST conventions and ownership rules.

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
0.5. **UI-first catalog prototype** (§17): homepage, product list/search, product detail, store page. *(Done; mocks replaced by the real API in Phase 2.)*
1. **Identity & security**: register/login/logout, Argon2id, sessions, auth middleware, RBAC, CSRF, authorization tests.
2. **Catalog**: categories, stores, products, variants, images, inventory basics, public listing, search/pagination/filter/sort.
3. **Marketplace UI** on real APIs (homepage, nav, cards, detail, store page, search).
4. **Cart & inventory**: cart, stock validation, movements, reservation.
5. **Checkout & orders**: addresses, shipping sim, transactional multi-store checkout, order lifecycle, payment simulation, cancellation + stock release.
6. **Reviews & frontend completion**: reviews, buyer/seller/admin dashboards.
7. **Hardening**: integration + E2E tests, security review, performance/index review, OpenAPI docs, deploy prep.

Scope status: Phases 0, 0.5, 1, 2, 4, 5 and 6 are committed; the shadcn/ui migration is committed; Phase 7 has a written plan (PLAN.md) awaiting approval. Phase 3 (marketplace UI on real APIs) is largely covered already because the prototype UI was wired to the Phase 2 API. Each later phase starts only after explicit developer approval; stop at the end of each phase for review.

## 15. Where the two source briefs disagreed (resolved — newer brief wins)

Router Chi→**Gin**; migrations Goose→**golang-migrate**; auth JWT→**cookie sessions**; hashing bcrypt/Argon2id→**Argon2id**; health `/api/v1/health`→**`/health` + `/ready`**; response envelope gained `success`; role "buyer"→**customer**; stock on `products`→**`inventory` + movements per variant**; single order per checkout→**kept multi-vendor `checkouts` → per-store `orders`** (old brief's rule, not in the new one). If you hit another conflict, flag it rather than silently picking.

## 16. Working agreement

- Before implementing: inspect the existing code, identify affected files, explain the approach and its security/regression implications, and **wait for approval** on substantial changes. Don't overwrite existing config without inspecting it.
- During: small coherent steps, scoped to the requested feature, follow existing conventions, migrations for schema, backend validation, tests for business-critical behaviour, no unnecessary abstractions or dependencies, never hardcode secrets, never bypass security mechanisms.
- After: review the diff, run formatting/lint/static analysis and relevant tests, check migrations up/down, check the frontend compiles, verify real API behaviour, list remaining risks, summarise modified files. **Never claim done because it compiles; never silently skip a failing test — report failures with their output.**
- **Never commit, push, or deploy automatically**; never touch production infrastructure or expose credentials. When a feature is complete, propose a commit message and let the developer decide. Explain any destructive database operation before running it.

## 17. UI-first catalog prototype (completed)

The developer chose to see the marketplace interface before backend business logic, so the catalog pages were first built on typed mock data (phase 0.5) and then switched to the real API in Phase 2 by rewriting only `src/services/catalog.ts`; pages and components did not change. `src/mocks/` and the `?simulate=` helper were deleted.

Lessons that still apply to new UI work:

- Pages and components get data through feature hooks → `src/services/`; nothing else calls the API or holds fixtures.
- Types in `src/types/` mirror the Go DTOs; update both together.
- Every data view implements loading, empty, error and success states.
- Controls for features without a backend (cart, checkout) stay visibly disabled; nothing pretends to be a real operation.
