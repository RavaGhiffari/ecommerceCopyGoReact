# PROJECT_STATE.md — living documentation + checkpoints

Where the project stands, how it fits together, and a trail of verified checkpoints.
**[CLAUDE.md](../CLAUDE.md) holds the rules** (conventions, domain rules, working agreement); **this file holds the state** (what exists, what was verified, what is next). Update this file at the end of every phase (see [How to update](#how-to-update-this-file)).

|  |  |
|---|---|
| Snapshot date | 2026-11-13 |
| Last commit | `41679b9` Add project state, session handoff and refresh roadmap docs (Phase 2 = `2838782` + `d0c2a54`) |
| Uncommitted | Phase 4 (cart) + doc updates — **not yet committed** (commit commands in §1) |
| Current phase | Phase 4 implemented and verified, **awaiting developer review** |
| Next phase | Phase 5 — Checkout & orders (needs a plan and answers to its open questions first; see §10) |

---

## 1. Checkpoints

A checkpoint is a state where **everything listed under "Verified" was actually run and passed**. Restoring one means `git checkout <commit>`, `make migrate-up`, and the verify commands in §8.

| # | Name | Commit | Status | Verified |
|---|---|---|---|---|
| CP-0 | Foundation | `0d45a70`…`b92f582`, `c4a5582` | ✅ reviewed | docker compose up; `/health` 200, `/ready` 200, and 503 with the DB stopped; migrate up→down→up; sqlc generate; unit + integration tests; frontend build/lint; graceful shutdown in logs |
| CP-1 | Identity & security | `759a82e`, `316c1dc` | ✅ reviewed | 13 auth integration tests on a real DB; curl register/login/me/logout/cross-site; 15-step browser E2E; logs contain no password/hash/token |
| CP-0.5 | UI prototype on mocks | `830c33d` (+ docs `d30b4ef`) | ✅ reviewed, superseded | 7 pages screenshotted incl. loading/empty/error; no component imported `mocks/`. Mocks were deleted in CP-2 |
| CP-2 | Catalog | `2838782` (backend), `d0c2a54` (frontend), docs `41679b9` | ✅ committed by the developer, who then approved Phase 4 | migrate up→down(4)→up; 13 unit packages; 12 catalog integration tests; concurrency test **fails without `FOR UPDATE`** (proved) and passes with it; ledger invariant = 0 mismatches; 17-step seller/catalog browser E2E + auth E2E re-run (no regression) |
| CP-4 | Cart | **uncommitted** | 🟡 awaiting review | see §1b |

**Commit CP-4** (the developer commits; Claude never does). Suggested split — run `git reset -q` first if anything is staged:

```bash
git add -A -- backend
git commit -m "Add Phase 4 cart: carts, cart items, stock-checked add/update/remove" -m "- Migration 000009 (one cart per user, 1..99 per line, unique variant per cart)" -m "- Cart row lock serialises changes; adding never reserves stock" -m "- Lines that became unbuyable stay in the cart with an issue flag" -m "- Unit and integration tests; also gofmt for slug_test.go" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -A -- frontend
git commit -m "Add cart page, header badge and add-to-cart" -m "- CartProvider keeps the server's cart; totals never computed client-side" -m "- /cart grouped by store with quantity controls and issue fixes" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -A -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs
git commit -m "Update project docs for Phase 4" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

`docs/e2e-reference/` is git-ignored by the developer's choice, so the third commit only picks up `docs/PROJECT_STATE.md`. After committing, put the hashes in the CP-4 row.

### 1a. Re-verified at the handoff session (2026-11-13)

Commands actually run in that session, against the working tree described above:

| Check | Result |
|---|---|
| `git status` | 62 paths at verification time (66 after the handoff docs were written): 27 M, 31 ??, 4 D (the deleted `frontend/src/mocks/*` + `services/simulate.ts`); HEAD = `316c1dc` |
| `docker compose ps` | `api` up, `postgres` healthy |
| `migrate version` on dev DB and on `marketplace_test` | both `8`, not dirty |
| `make test` | 13 packages `ok` |
| `make test-integration` | 14 packages `ok` (adds `modules/health`, which only has an integration test) |
| Test functions in source | 83 (`grep "^func Test"`); 3 files carry `//go:build integration` |
| `npm run build` / `npm run lint` | both clean; no `mocks/` or `simulate` references left in `frontend/src` |
| Ledger invariant on dev DB | 0 mismatches |
| `make seed-dev` on the seeded dev DB | prints "categories already exist … nothing to do" (idempotent) |
| Dev DB content | 15 users, 8 stores, 27 products, 47 variants, 23 categories, 45 inventory movements, 4 sessions (0 expired) |
| API log sample | all 38 request lines show `client_ip` `172.18.0.1` (Docker gateway) → see §9 |

**Not re-run at handoff** (observed earlier in the session; treat as "was true at CP-2", not re-confirmed): the concurrency test failing without `FOR UPDATE`; the curl flows; the two browser E2E scripts; graceful shutdown; screenshots.

### 1b. Verified at CP-4 (cart), 2026-11-13

| Check | Result |
|---|---|
| Migration `000009` | up → down 1 → up on the dev DB; `make test-db` applied it to `marketplace_test`; CHECK rejects quantity 0, UNIQUE rejects a second line for the same variant (tried by hand in psql) |
| `gofmt`, `go vet` (incl. `-tags=integration`) | clean after `make fmt` (it also fixed `slug_test.go`, committed unformatted in CP-2) |
| `make test` | 14 packages `ok` (new: `modules/cart`) |
| `make test-integration` | 15 packages `ok`; 12 cart tests listed by `-v` and passing |
| Lock check | `TestCart_ConcurrentAddsOfTheSameVariant` **failed** with the cart-row lock removed (quantity 1 instead of 10) and passed 5/5 with it restored |
| `npm run build` / `npm run lint` | clean |
| Browser E2E (headless Edge, Vite :5190, real API + dev DB) | `cart-flow.mjs` 16/16; `auth-flow.mjs` 15/15 and `catalog-flow.mjs` 17/17 re-run, no regression; 0 uncaught page errors |
| Dev DB after E2E | ledger mismatches 0; `SUM(reserved_quantity)` = 0 (the cart reserved nothing); the demo stock changed by the E2E was restored (Kaos M = 80) |

Test functions in source: 98 (was 83).

---

## 2. What exists, by phase

| Phase | Scope | State |
|---|---|---|
| 0 Foundation | `backend/` layout, config, pgxpool, JSON logging, request ID, CORS, graceful shutdown, `/health` `/ready`, golang-migrate, sqlc, Makefile, Compose, status page | ✅ |
| 0.5 UI prototype | Catalog pages on typed mock data | ✅ (now on real API) |
| 1 Identity | users, roles, sessions; register/login/logout; Argon2id; CSRF layers; rate limits; `/me`; `seed-admin` | ✅ |
| 2 Catalog | categories (2 levels), stores, products, variants, images (URLs), inventory + ledger, public listing/search, seller dashboard, `seed-dev` | ✅ |
| 3 Marketplace UI | Homepage, nav, cards, detail, store page, search on real APIs | ≈ done as a side effect of CP-2; polish (banner, popular/recommended refinements, mobile check) remains |
| 4 Cart | carts + items, stock-checked add/update/remove (no reservation), issue flags, `/cart` page, header badge, add-to-cart | 🟡 implemented + verified, review pending |
| 5 Checkout & orders | addresses, shipping sim, transactional multi-store checkout, order lifecycle, payment sim | ⬜ |
| 6 Reviews & dashboards | reviews, rating aggregates, buyer/seller/admin dashboards | ⬜ |
| 7 Hardening | integration + E2E suites, security review, query/index review, OpenAPI, deploy | ⬜ |

---

## 3. Architecture map

```
Browser ──(Vite proxy /api,/health,/ready; Host kept)──► Gin API :8081 ──► PostgreSQL 18
 React 19 + TS                                            │
 features/ pages/ services/ ── fetch ──────────────────► router → middleware → handler → service → repository → sqlc → pgxpool
```

Backend request path (`backend/internal/`):

- `router/router.go` — the whole API surface in one file. Middleware order: `RequestID → Logging → Recovery → CORS → Timeout → SameOrigin → Authenticate`; per-route `RequireAuth` / `RequireRole`.
- `modules/<name>/` — `handler.go` (HTTP only) → `service.go` (rules, transactions) → `repository.go` (sqlc wrapper, maps DB errors to domain errors) + `dto.go` (JSON contract) + `model.go`.
- `database/queries/` — **generated** by sqlc from `db/queries/*.sql`; never edit. `database.WithTx` runs a function in a transaction; repositories accept `queries.DBTX` (pool **or** tx).
- `platform/` — `password`, `token`, `ratelimit`, `validation`, `authctx`, `request`, `response`, `logger`, `slug`.

Frontend (`frontend/src/`): `pages/` (routes) → `features/<area>/` (hooks, forms, views) → `services/` (**only** place that calls the API) → `lib/api.ts` (envelope-aware fetch, `ApiError`). `types/` mirrors Go DTOs by hand.

Frontend routes (`src/app/router.tsx`): `/`, `/products`, `/categories/:categorySlug`, `/stores/:storeSlug`, `/stores/:storeSlug/products/:productSlug`, `/login`, `/register`, `/account` (RequireAuth), `/seller` (RequireAuth; opens a store or shows the dashboard), `/seller/products/new` and `/seller/products/:id` (RequireSeller), `/cart` (RequireAuth), `/status`, `*`. "Tambah ke keranjang" works (anonymous → `/login?next=…`); **"Beli sekarang" and "Lanjut ke checkout" stay disabled** until Phase 5. `CartProvider` (inside `AuthProvider`) holds the server's cart for the badge, product page and `/cart`.

Runtime dependencies — Go: gin, pgx/v5, pgx-shopspring-decimal, shopspring/decimal, google/uuid, x/crypto (argon2), x/text (slug NFD), testify (tests). Frontend: react 19, react-router-dom, react-hook-form, zod 4, @hookform/resolvers, lucide-react, tailwindcss 4 (+ vite plugin). Pinned tool images: `sqlc/sqlc:1.29.0`, `migrate/migrate:v4.18.3`; `golang:1.27-alpine` and `postgres:18` are floating tags.

---

## 4. API surface (as of CP-4)

Base `/api/v1`. Envelope: `{success, data, message}` / `{success:false, error:{code,message,details}}`.

| Group | Routes |
|---|---|
| Infra (no prefix) | `GET /health` (no DB), `GET /ready` (DB query) |
| Public | `POST /auth/register`, `/auth/login`, `/auth/logout` · `GET /categories` · `GET /products?q&category&store&sort&page&pageSize` · `GET /products/:id/related` · `GET /stores/:slug` · `GET /stores/:slug/products/:productSlug` |
| Signed in | `GET`/`PATCH /me` · `POST /seller/store` (opens store + grants seller role) · `GET /cart` · `POST /cart/items` `{variantId, quantity}` (adds to an existing line) · `PATCH /cart/items/:id` `{quantity}` · `DELETE /cart/items/:id` — every cart route returns the whole cart |
| Seller (`seller` role) | `GET`/`PATCH /seller/store` · `GET`/`POST /seller/products` · `GET`/`PATCH`/`DELETE(archive) /seller/products/:id` · `POST …/variants` · `PATCH …/variants/:variantId` · `PUT …/variants/:variantId/stock` |
| Admin (`admin` role) | `POST /admin/categories` · `PUT /admin/categories/:id` |

Status codes in use: 200 201 204 400 401 403 404 409 413 415 422 429 500 503. Error codes are SCREAMING_SNAKE (`EMAIL_TAKEN`, `SLUG_TAKEN`, `SKU_TAKEN`, `STORE_EXISTS`, `RATE_LIMITED`, `ORIGIN_REJECTED`, …). Cart adds: `CART_ITEM_NOT_FOUND` 404, `VARIANT_NOT_FOUND` 404, `VARIANT_UNAVAILABLE` 409, `OUT_OF_STOCK` 409, `QUANTITY_EXCEEDS_STOCK` 409 (details `available`, `inCart`), `OWN_PRODUCT` 409; quantity outside 1..99 → 422.

Cart response: `{ stores: [{ store, items: [{ id, quantity, variantId, variantName, sku, productId, productSlug, productName, image|null, unitPrice, lineTotal, availableQuantity, issue|null }], subtotal }], itemCount, subtotal, hasIssues }`. `issue` ∈ `unavailable` (archived product / suspended store), `out_of_stock`, `insufficient_stock`; flagged lines are excluded from both subtotals. `itemCount` = total units.

---

## 5. Data model (migrations 000001–000009)

| Table | Notes |
|---|---|
| `users` | UUID; `email` lower-case CHECK + unique `LOWER(email)`; status active/suspended |
| `roles`, `user_roles` | roles seeded: customer, seller, admin |
| `user_sessions` | `token_hash` BYTEA(32) unique; only the hash is stored |
| `categories` | `parent_id`; **two levels** (service check + trigger `categories_max_two_levels`) |
| `stores` | `owner_id` UNIQUE (one store per seller); slug immutable |
| `products` | `UNIQUE(store_id, slug)`; generated `search` tsvector + GIN; status active/archived; `sold_count` |
| `product_variants` | `price NUMERIC(12,2) ≥ 0`; global unique SKU; `UNIQUE(product_id, name)` |
| `product_images` | URLs only (`^https?://` CHECK) |
| `inventory` | PK = variant; `available_quantity`, `reserved_quantity` both CHECK ≥ 0 |
| `inventory_movements` | append-only ledger; deltas, type, reason, reference, actor |
| `carts` | one per user (`user_id` UNIQUE, cascade on user delete); created on the first add |
| `cart_items` | `quantity` CHECK 1..99; `UNIQUE(cart_id, variant_id)`; variant FK RESTRICT |

**Invariants worth re-checking after any inventory change**

```sql
-- must return 0: ledger explains every stock level
SELECT count(*) FROM inventory i
LEFT JOIN (SELECT variant_id, SUM(available_delta) s FROM inventory_movements GROUP BY variant_id) m USING (variant_id)
WHERE COALESCE(m.s,0) <> i.available_quantity;
```

Until Phase 5 also: `SELECT SUM(reserved_quantity) FROM inventory` must be 0 — nothing reserves stock yet (the cart must not).

Not yet created (planned): `user_addresses`, `checkouts`, `orders`, `order_items`, `order_status_histories`, `payments`, `payment_events`, `reviews`, `shipping_methods`, `shipments`, `audit_logs`.

---

## 6. Decision log

Decisions made so far and **why** (newest first). Reverse one only deliberately.

| Decision | Why |
|---|---|
| Money = `NUMERIC` ↔ `decimal.Decimal` ↔ JSON string `"349000.00"` | floats corrupt money; JSON numbers become JS floats |
| Variant price is authoritative; product has none | one source of truth; listings show min/max |
| Products only on leaf categories | avoids products parked on browse-only parents |
| Stock change = lock row `FOR UPDATE` → update → ledger row, one tx | proven necessary: without the lock the ledger drifts under concurrency |
| Ledger is append-only, `SUM(delta) = quantity` | auditable stock |
| Open store + grant seller role in **one** transaction | no seller without a store, no store without its role |
| Seller routes take the store from the session; queries filter `store_id` | foreign IDs return 404, nothing to probe |
| Sessions: opaque token in HttpOnly cookie, only SHA-256 stored; rotate on login | DB leak ≠ session theft; defeats fixation |
| CSRF = SameSite=Lax + Origin/Host check + JSON-only bodies (no double-submit token) | enough for same-site SPA behind a proxy; revisit for cross-site deployment |
| Vite proxy uses `changeOrigin:false` for `/api` | CSRF check compares `Origin` with `Host`; rewriting Host breaks it |
| Argon2id, OWASP-minimum parameters, configurable | tests use cheaper params |
| Search = Postgres FTS `'simple'` + prefix terms from sanitised words; no trigram | keeps tsquery injection-proof; no Indonesian dictionary exists |
| Product URLs are store-scoped | slug uniqueness is per store |
| Gin, golang-migrate, cookie sessions (newer brief) over Chi, Goose, JWT | resolved conflict between the two source briefs (CLAUDE.md §15) |
| sqlc overrides: uuid → `google/uuid` (nullable → pointer), numeric → decimal | no driver types leak into services |
| Services never import Gin; handlers map domain errors to HTTP | testability, clean layering |
| Integration tests run only against `*_test` DB and TRUNCATE it | dev data can't be wiped by tests |
| One store per seller (`stores.owner_id` UNIQUE); the `seller` role is granted when the store is opened | simplest ownership model; revisit explicitly if multi-store sellers are wanted |
| Admins exist only via `make seed-admin` (password from prompt or `ADMIN_PASSWORD`, never a flag) | no admin sign-up endpoint, no default credentials |
| `register` always creates a `customer`; the body may not contain `role` (unknown fields are rejected) | client can never choose a role |
| Unknown email and wrong password return the same 401; a dummy Argon2 verify equalises timing; "suspended" is revealed only after a correct password | prevents account enumeration |
| Suspension is enforced inside the session lookup SQL | a suspended user is logged out on the next request, no cleanup job needed |
| Login/register rate limits: 5 failures / 15 min per IP+email (reset on success); 10 sign-ups / hour per IP | brute-force and mass-signup friction; in memory (see §9) |
| Session TTL 24 h (`SESSION_TTL`), register logs the user in | developer's choice for the benchmark phase |
| Password rules: 8–128 characters counted as characters, no composition rules | NIST-style; length matters most |
| Categories: a parent filter also returns its children's products | "Elektronik" must show "Audio" items |
| Public visibility = product active ∧ store active ∧ ≥ 1 variant | same rule in list, detail and store counts |
| `seed-dev` creates data through the real services (not raw SQL) | demo data obeys validation and writes ledger rows |
| Dev DB is named `main` (developer's `.env`); `.env.example` says `marketplace_dev`; `marketplace_test` is created by `make test-db` | do not hardcode either name in scripts |
| Frontend `RequireAuth` / `RequireSeller` are UX only; API enforces access | CLAUDE.md §6 |
| Logout navigates away first, then clears the user | otherwise a protected page's guard redirects to `/login?next=…` and overrides the navigation |
| Frontend forms: React Hook Form + Zod, server field errors mapped to RHF paths (`variants[0].price` → `variants.0.price`) | one error path for client and server validation |
| Cart (Phase 4, approved defaults Q1–Q5): login required; quantity per line `min(stock, 99)`; sellers cannot add their own products (`OWN_PRODUCT`); stock problems are flags in `GET /cart`, errors only on add/update; badge count comes from `GET /cart` | developer approved the proposed defaults |
| A cart is not a reservation: no `inventory` writes, no ledger rows; prices shown are always current | CLAUDE.md §7; checkout (Phase 5) snapshots prices and reserves stock under lock |
| Every cart mutation runs in one tx that first locks the cart row (`INSERT … ON CONFLICT DO UPDATE` / `SELECT … FOR UPDATE`) | proven: without it concurrent adds lose updates |
| Unbuyable lines stay in the cart with `issue`, excluded from subtotals; they can only be removed (unavailable) or reduced (insufficient stock) | the user sees why, instead of items silently vanishing |
| Cart endpoints return the whole cart, including after delete (200, not 204) | the UI never computes totals; one round trip |
| Cart item ownership: update/delete filter by the caller's cart (`WHERE id = $1 AND cart_id = $2` / join on `carts.user_id`) | another user's item id is a 404 |
| Frontend `CartProvider` keys the cart by user **object** (new key per login) | a re-login always refetches instead of showing a cached cart |

---

## 7. Test inventory

- **Unit** (`make test`, no DB): config, middleware (request ID, recovery, timeout, CORS, auth, same-origin), response, logger, password, token, ratelimit, request/JSON, slug, products validation + search query, auth validation, users model, cart model (issue classification, grouping, exact totals). 98 test functions in total (counted in source), 14 unit packages + `modules/health` integration only; integration-tagged files: `modules/health/integration_test.go`, `router/auth_integration_test.go`, `router/catalog_integration_test.go`, `router/cart_integration_test.go`.
- **Integration** (`make test-integration`, `-tags=integration`, DB `marketplace_test`): health against a real DB; 13 auth flows; 12 catalog flows (ownership, uniqueness, validation, ledger, concurrency, search/sort/filter/pagination, archive); 12 cart flows (login required, empty cart, grouping/totals, not-a-reservation, quantity/stock rules, own product, archived + stock-drop flags, current price, ownership 404s, persistence across sessions, concurrent adds).
- **Browser E2E**: three `playwright-core` scripts (`auth-flow.mjs` 15 checks, `catalog-flow.mjs` 17, `cart-flow.mjs` 16) run by hand against Vite on port 5190 + the real API and dev DB. Copies live in `docs/e2e-reference/`, which the developer **git-ignored** — they exist only on this machine. They create throw-away accounts in the dev DB; `cart-flow.mjs` temporarily lowers a demo stock and restores it. A proper Playwright suite is a Phase 7 deliverable.
- **Frontend unit tests (Vitest)**: none yet.

---

## 8. Run, verify, restore

```bash
make up                 # build + start postgres and api
make migrate-up         # apply migrations to the dev DB (named `main` on this machine)
make seed-dev           # demo catalog (no-op if categories exist)
make test               # unit tests
make test-integration   # creates/migrates marketplace_test, runs integration tests
cd frontend && npm run build && npm run lint
cd frontend && npm run dev -- --port 5174    # open http://localhost:5174  (or http://[::1]:5174)
curl http://localhost:8081/health ; curl http://localhost:8081/ready
```

Demo logins (local only): `penjual1@contoh.id` … `penjual4@contoh.id` / `penjual-123`. Admins: `make seed-admin email=… name="…"`.

**API configuration** (all optional except `DATABASE_URL`; defaults in `internal/config/config.go`): `APP_ENV` (development|production), `APP_PORT` (8081), `LOG_LEVEL` (info), `CORS_ALLOWED_ORIGINS` (`http://localhost:5173`), `REQUEST_TIMEOUT` (10s), `SHUTDOWN_TIMEOUT` (15s), `SESSION_TTL` (24h), `ARGON2_MEMORY_KIB` (19456) / `ARGON2_ITERATIONS` (2) / `ARGON2_PARALLELISM` (1), `DB_MAX_CONNS` (10) / `DB_MIN_CONNS` (2) / `DB_MAX_CONN_LIFETIME` / `DB_MAX_CONN_IDLE_TIME` / `DB_HEALTH_CHECK_PERIOD` / `DB_CONNECT_TIMEOUT`. Compose forwards only `APP_ENV`, `APP_PORT`, `LOG_LEVEL`, `CORS_ALLOWED_ORIGINS`, `SESSION_TTL`, `DATABASE_URL`; the rest need a compose change to be overridden.

**Environment quirks** (this machine): Go is not installed on the host — everything Go/sqlc runs in Docker via the Makefile; GNU make came from scoop; Git Bash path conversion is disabled in the Makefile (`MSYS_NO_PATHCONV=1`); local PostgreSQL 18 owns port 5432 (Compose uses **5433**); another app owns IPv4 `:5173` — use `--port 5174` or `http://[::1]:5173`. Never kill a process by port without checking its command line.

---

## 9. Known gaps, risks, tech debt

| Item | Impact | When |
|---|---|---|
| Rate limiter is in memory | wrong behind >1 API instance | before scaling out |
| No CSRF token (relies on SameSite + Origin) | fine for same-site; weaker if frontend is hosted on another site | Phase 7 review |
| Dev DB contains leftover test accounts/stores (`seller-…@contoh.id`, `admin@contoh.id`, `e2e-…`) | clutter only | cleanup offered, not done |
| Categories manageable only via admin API (no UI) | admin must use curl | Phase 6 |
| Images are URLs only; no upload, no hotlink protection | seller can point at any host | decide with storage plan |
| `sold_count` only seeded; ratings are 0 | "popular" is demo data | Phases 5–6 |
| Reservation columns exist but nothing uses `reserved_quantity` yet | — | Phase 5 |
| Right after "Keluar", the header badge keeps the old count until the logout request returns (the page navigates first by design) | cosmetic, milliseconds | accept, or clear the cart state optimistically |
| Cart issue flags are recomputed on every `GET /cart`, but the page does not refresh by itself | a stock change by the seller shows up on the next load/mutation | acceptable; checkout re-validates |
| `ListCartLines` joins six tables + a LATERAL image lookup; no `EXPLAIN` yet | fine for small carts | Phase 7 index review |
| E2E reference scripts are git-ignored (`/docs/e2e-reference`) | they are lost if this machine is lost | promote into a real Playwright suite (Phase 7) |
| No `/auth/me`; no email verification; no password reset | account recovery impossible | backlog |
| No audit log table yet | admin actions untracked | Phase 6/7 |
| Frontend has no automated tests | regressions caught manually | Phase 7 |
| Mobile layout checked only down to ~500 px | 390 px unverified (headless limit) | Phase 3 polish |
| Money display converts strings to JS numbers | fine for display; never sum client-side for charging | rule for Phase 4–5 |
| **Rate limiter keys on `c.ClientIP()`, and every request reaches the API from the Docker gateway `172.18.0.1`** (verified in logs) while `SetTrustedProxies(nil)` ignores `X-Forwarded-For` | in dev all users share one budget (10 sign-ups/hour!); in production behind a reverse proxy the same happens unless trusted proxies are configured | **before Phase 5 / deploy**; sign-up-heavy E2E runs can hit it |
| Expired sessions are deleted only when *the same user* logs in again | `user_sessions` grows for users who never return | cleanup job in Phase 7 |
| Cookie `Secure` flag is set only when `APP_ENV=production`; that path has **no test** | a production misconfiguration would go unnoticed | Phase 7 |
| No `EXPLAIN` / load check of `ListPublicProducts` (LATERAL joins + window count) | indexes exist but are unproven at scale | Phase 7 |
| `AddVariant` checks the 20-variant cap before inserting (check-then-insert) | two concurrent requests can exceed 20 | low priority |
| Search uses the `simple` config without `unaccent` | "kemeja" ≠ "kémeja"; no stemming | acceptable for now |
| `.env.example` lists only 9 of the 17 config variables | people don't discover optional tuning | document-only; defaults are safe |
| `frontend/README.md` is still the Vite template | misleading | replace with a short pointer to docs |
| `golang:1.27-alpine` / `postgres:18` tags float | builds can change under you | pin in Phase 7 |
| No CI | nothing runs tests on push | Phase 7 |
| CP-4 uncommitted | verified work exists only in the working tree | **commit first** (§1) |
| **Compose publishes Postgres (`0.0.0.0:5433`) and the API (`0.0.0.0:8081`) on all interfaces, and the developer's `.env` DB password is only 4 characters** (length checked; value not recorded) | anyone on the same network can attempt to log in to the dev database | developer decision: use `127.0.0.1:${POSTGRES_HOST_PORT:-5433}:5432` in compose and a longer password; **not changed** at handoff |

---

## 10. Next steps (proposed)

The dependency-checked version, with the exact next task, is in [SESSION_HANDOFF.md](../SESSION_HANDOFF.md); the Phase 5 questions are in [PLAN.md](../PLAN.md).

1. Developer reviews CP-4 and commits it (§1).
2. Before Phase 5: decide the rate-limiter/trusted-proxy question (§9) and the dev DB exposure (compose binds `0.0.0.0`).
3. **Phase 5 — Checkout & orders**: write the plan first (PLAN.md lists the open questions), get approval, then: `user_addresses`, shipping simulation, one-transaction checkout (lock inventory rows, `available → reserved` + ledger, per-store orders, price/name/SKU snapshots, idempotency key), order lifecycle + status history, payment simulation (consume / release reservations), cancellation and expiry. Concurrency test for overselling is mandatory.

---

## 11. Failed approaches and lessons (so they are not repeated)

| What happened | Cause | What was done / lesson |
|---|---|---|
| `docker run … -w /src` failed with `C:/Program Files/Git/src` | Git Bash rewrites arguments that look like POSIX paths | `export MSYS_NO_PATHCONV := 1` in the Makefile; use it in any ad-hoc `docker run` from Git Bash |
| `go`, `docker`, `make` "not found" in the agent shell | not on PATH (Go not installed at all) | Go/sqlc/migrate run in containers; Docker CLI path is in CLAUDE.md; `make` via scoop |
| `seed-admin` test created an admin in the **dev** DB | a `sed` rewrote `…/marketplace_dev?` but the dev DB is called `main` | never assume the dev DB name; check `.env` keys, not values; test accounts were left behind |
| A process on port 5173 was killed and turned out to be the developer's Laravel app | killed by port without reading its command line | always inspect `CommandLine` first; use port 5190 for agent-run Vite |
| sqlc generated `pgtype.UUID` / `pgtype.Numeric` | default driver types | `overrides` in `sqlc.yaml` (uuid → google/uuid, nullable uuid → pointer, numeric → decimal) |
| sqlc typed a LEFT JOIN LATERAL image column as non-null `string` | cannot infer nullability through LATERAL | `COALESCE(...)::text` in the query; read generated code for every new join |
| `@id` used twice in a query generated `*uuid.UUID` params | untyped parameter inference | cast: `@id::uuid` |
| Code used `queries.GetStoreByOwnerRow` that does not exist | sqlc returns the model type when a query selects every column | check generated names before writing repositories |
| Logout from `/account` landed on `/login` | user cleared before navigation, guard redirected | navigate first, then log out |
| Duplicate-email message appeared twice | field error + form-level error both shown | form-level message only when no field error was placed |
| Product editor showed "Pilih kategori" for a saved category | RHF default value written before the `<option>`s existed | mount the registered `<select>` only after categories load; regression check in E2E |
| First product-detail curl returned 404 | test used a guessed slug; slugs derive from the full name ("…-motif-parang") | fetch slugs from the API instead of guessing |
| Rewriting a CLAUDE.md section ran two interpreters (unclosed heredoc) | shell mistake | prefer the Edit tool for doc changes |
| Early PHASES.md carried hour estimates and `/auth/me` | written before decisions were made | PHASES.md was rewritten at handoff; do not reintroduce estimates |
| Chi vs Gin / JWT vs cookies were argued in chat | two briefs conflicted | resolved in CLAUDE.md §15 (newer brief wins); the developer approved Gin + cookie sessions |
| `slug_test.go` was committed unformatted in CP-2 | `gofmt -l` ran before that file was written | run `make fmt` as the **last** step before reporting a phase done |
| Two cart test expectations were wrong (3 × 100000.50 written as 301501.50; 2 × 59000.50 as 119001.00) | hand arithmetic in the test | the code was right; confirm the actual value (`-run X -v`) before "fixing" either side |
| Cart E2E saw the old badge count right after "Keluar" | the test read the badge before the logout request returned | wait for the "Masuk" link (logged-out header) before asserting |

---

## How to update this file

At the end of each phase (or any risky change):

1. Run the verify commands in §8; only then add a **Checkpoint** row in §1 with the commit hash and what was *actually* run.
2. Update the snapshot header, §2 status, and §4/§5 if routes or tables changed.
3. Add new decisions to §6 (with the reason); move fixed items out of §9.
4. Keep CLAUDE.md for rules and this file for state — don't duplicate long text between them; link instead.
