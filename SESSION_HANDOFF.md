# SESSION_HANDOFF.md

Written 2026-11-13 for a **fresh Claude Code session**. Nothing here was implemented during the handoff itself; the handoff only verified, documented and tidied docs.

**Read order:** this file → [CLAUDE.md](CLAUDE.md) (rules, loaded automatically) → [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md) (checkpoints, decision log, known issues, failed approaches) → [PLAN.md](PLAN.md) (next task).

**Evidence labels used below**
- **[verified now]** — a command was run in the handoff session and its output was read.
- **[verified earlier]** — observed earlier in the same long session; not re-run at handoff.
- **[not verified]** — believed from reading code or from conversation only.

---

## 1. Current project understanding

**What it is.** A learning project: a multi-vendor marketplace (Tokopedia-style UX, original branding) with Go backend + React frontend + PostgreSQL. The developer is **learning Go and backend engineering**, so code must be explicit and readable, and important flows need explanations (request → handler → service → repository → SQL, why each constraint/transaction exists). Responses to the developer are in **Indonesian**; repo documentation is in English.

**Stack.** Go 1.27.3 + Gin + pgx/pgxpool + sqlc + golang-migrate · PostgreSQL 18 · React 19 + TypeScript + Vite + Tailwind 4 + react-router-dom + react-hook-form + zod · Docker Compose (postgres + api; the frontend runs on the host with `npm run dev`). Modular monolith: `router → middleware → handler → service → repository → sqlc → pgxpool`.

**Working agreement (from CLAUDE.md §16, non-negotiable):** propose a plan and wait for approval before substantial work; one phase at a time, stop at each phase end for review; **never commit, push or deploy** (propose commit messages); never print or commit `.env`; report failures honestly; don't claim "done" without running checks.

**Roles:** customer / seller / admin. Sellers are customers who opened a store. Admins exist only via `make seed-admin`.

**Where we are.** Phases 0, 0.5 (UI prototype) and 1 (identity) are done and reviewed by the developer. **Phase 2 (catalog) is implemented, verified and awaiting review — and is not committed.** Next is Phase 4 (cart); Phase 3 (marketplace UI) was largely delivered as a side effect of wiring the prototype UI to the Phase 2 API.

---

## 2. Completed work

| Area | State | Evidence |
|---|---|---|
| Foundation (config, pool, middleware, graceful shutdown, health/ready, migrations, sqlc, Makefile, Compose) | committed (`0d45a70`, `b92f582`, `c4a5582`) | [verified now] via `make test`/`test-integration`; shutdown + 503 on DB stop [verified earlier] |
| Identity: users, roles, sessions, Argon2id, CSRF layers, rate limits, `/me`, `seed-admin` | committed (`759a82e`, `316c1dc`) | [verified now] 13 auth integration tests pass; browser E2E [verified earlier] |
| Catalog backend: categories (2 levels), stores, products, variants, images, inventory + ledger, public listing/search, seller API | **uncommitted** | [verified now] migrations at v8 on dev + test DBs; `make test` 13 packages ok; `make test-integration` 14 packages ok; ledger invariant 0; 83 test functions |
| Catalog + seller frontend on the real API; mocks removed | **uncommitted** | [verified now] `npm run build` + `npm run lint` clean, no mock references; browser E2E 17 + 15 checks [verified earlier] |
| `make seed-dev` demo data | uncommitted | [verified now] idempotent on the seeded dev DB |
| Docs: CLAUDE.md, docs/PROJECT_STATE.md, PHASES.md, PLAN.md, this file, docs/e2e-reference/ | uncommitted | rewritten/checked at handoff |

Working tree: **66 changed paths** (30 modified, 32 untracked, 4 deleted; it was 62 before the handoff docs were added) on top of `HEAD = 316c1dc`. Suggested commits are in `docs/PROJECT_STATE.md` §1.

**Dev database right now** [verified now]: 15 users, 8 stores, 27 products, 47 variants, 23 categories, 45 inventory movements, 4 sessions. It contains **leftover test accounts** from E2E runs (`seller-…@contoh.id`, `e2e-…@contoh.id`, `demo@contoh.id`, `admin@contoh.id` created by an accidental `seed-admin` test). Demo sellers: `penjual1..4@contoh.id` / `penjual-123` (local only).

---

## 3. Important technical decisions and rationale

Full table: `docs/PROJECT_STATE.md` §6. The ones a new session is most likely to trip over:

1. **Money = `NUMERIC(12,2)` ↔ `decimal.Decimal` ↔ JSON string** (`"349000.00"`). Never float. The pgx codec is registered in `database.NewPool` (`AfterConnect`); sqlc overrides map numeric → decimal, uuid → `google/uuid`.
2. **Variant price is authoritative**; products have no price. Public listings show min/max. A product is public only if active ∧ store active ∧ ≥ 1 variant.
3. **Stock changes** = `SELECT … FOR UPDATE` → update → ledger row, **one transaction** (`modules/inventory`). Invariant: `SUM(available_delta) = available_quantity` per variant. Without the lock a concurrency test fails [verified earlier] — never remove it.
4. **Ownership by construction:** seller routes derive the store from the session and every query filters by `store_id`; other users' IDs return 404. Same pattern must be used for carts/orders (filter by the caller's cart/user).
5. **Sessions:** opaque token in HttpOnly SameSite=Lax cookie `sid`, only SHA-256 stored, rotated at login, deleted at logout; suspension enforced inside the session SQL. **CSRF** = SameSite + `SameOrigin` middleware (Origin vs Host/allow-list) + JSON-only bodies. No double-submit token.
6. **Vite proxy uses `changeOrigin:false` for `/api`** — the Origin/Host CSRF check depends on the browser's Host reaching the API.
7. **Layering:** services never import Gin and return plain errors or `validation.Errors`; handlers map them to HTTP; repositories translate Postgres constraint names (e.g. `products_store_slug_key`) into domain errors and take `queries.DBTX` (pool or tx). Multi-step writes use `database.WithTx` in the **service**.
8. **Response envelope** `{success,data,message}` / `{success:false,error:{code,message,details}}`; validation → 422 with per-field `details` (nested keys like `variants[0].price`); malformed IDs → 404.
9. **Categories** two levels, enforced by service **and** DB trigger; products must use a leaf category; a parent filter includes children.
10. **Search** = PostgreSQL FTS (`simple`, generated tsvector, GIN) with prefix terms built from sanitised words (no tsquery injection). No trigram.
11. **Tests:** integration tests run only against a `*_test` database and `TRUNCATE users, categories CASCADE`; `make test-integration` creates/migrates it.
12. **Frontend:** pages → feature hooks → `services/` (only API callers) → `lib/api.ts`. Auth state in `AuthProvider`/`useAuth`; `RequireAuth`/`RequireSeller` are UX-only. Server field errors map onto form fields via `applyServerError`.
13. Conflicts between the two source briefs were resolved in favour of the newer one (Gin, golang-migrate, cookie sessions, Argon2id, `/health` + `/ready`); multi-vendor `checkouts → per-store orders` was kept from the older brief (CLAUDE.md §15).

---

## 4. Files and modules relevant to the next task (Phase 4 — Cart)

Plan: [PLAN.md](PLAN.md). New files go next to these patterns.

**Backend — read before writing**

| Path | Why |
|---|---|
| `backend/internal/router/router.go` | add cart routes to the `signedIn` group |
| `backend/internal/modules/products/` (`handler.go`, `service_public.go`, `service_seller.go`, `repository.go`, `dto.go`, `model.go`) | the module pattern to copy: validation, ownership, error mapping, DTO/money formatting |
| `backend/internal/modules/inventory/inventory.go` | how stock is read/changed; the cart only **reads** `available_quantity` and must not reserve |
| `backend/db/queries/products.sql` (`ListPublicProducts`, `ListVariantsWithStock`) | public-visibility rule and the LATERAL/`COALESCE` pattern to reuse |
| `backend/db/migrations/000007…`, `000008…` | variant and inventory tables the cart references |
| `backend/internal/middleware/auth.go` + `platform/authctx` | `RequireAuth`, `Principal.UserID` for scoping |
| `backend/internal/platform/{validation,request,response}` | validation errors, strict JSON, envelope and error codes |
| `backend/internal/router/catalog_integration_test.go` | helpers (`newTestApp`, `newSeller`, `createProduct`, `do`) to reuse for cart integration tests |
| `backend/sqlc.yaml`, `Makefile` (`sqlc`, `migrate-*`, `test-integration`) | generation and verification commands |

**Frontend — read before writing**

| Path | Why |
|---|---|
| `frontend/src/features/catalog/ProductDetailView.tsx` | disabled "Tambah ke keranjang" / "Beli sekarang"; has the selected variant + quantity state |
| `frontend/src/components/layout/Header.tsx` | disabled cart button; add the item-count badge here |
| `frontend/src/app/router.tsx` | add `/cart` behind `RequireAuth` |
| `frontend/src/lib/api.ts`, `services/seller.ts` (style), `types/seller.ts` | `apiRequest`, `ApiError.fieldErrors()`, service/type conventions |
| `frontend/src/features/auth/{AuthProvider,authContext,RequireAuth,serverErrors}.ts(x)` | auth state, redirect with `?next=`, error mapping |
| `frontend/src/features/catalog/ProductResults.tsx`, `components/shared/StateViews.tsx` | loading/empty/error patterns |
| `frontend/src/lib/format.ts` | `formatRupiah` accepts decimal strings (display only) |

**Dependencies of the task**
1. CP-2 committed (blocker for a clean baseline).
2. Developer approval of PLAN.md and answers to Q1–Q5 in it.
3. Phase 2 tables exist (they do: migrations 1–8) and auth sessions work (they do).
4. Docker running (`make up`); `make` on PATH; dev DB reachable on host port 5433.

---

## 5. Known limitations and unresolved issues

Full tables: `docs/PROJECT_STATE.md` §9 (debt/risks) and §11 (failed approaches). Highlights:

**Unresolved issues**
- **Rate limiter shares one bucket in dev** — every request reaches the API from the Docker gateway `172.18.0.1` [verified now: 38/38 log lines]; with `SetTrustedProxies(nil)` the real client IP is never seen. Register is limited to 10/hour **per IP**, so repeated sign-ups (e.g. E2E runs) can hit 429. Workaround: restart the `api` container (in-memory counters). Real fix needs a trusted-proxy decision; schedule before deploy.
- Rate limiter is in memory (single instance only).
- Expired sessions are only deleted when that user logs in again.
- Cookie `Secure` path (`APP_ENV=production`) has no test.
- **Security hygiene (developer decision, not changed):** Compose publishes Postgres on `0.0.0.0:5433` [verified earlier] and the developer's `.env` DB password is 4 characters long [verified now: length only, value not read into docs]. Suggest binding to `127.0.0.1` and a longer password.
- `ListPublicProducts` never checked with `EXPLAIN`; no load test.
- Seller `AddVariant` 20-variant cap is check-then-insert (race possible, low impact).
- Ratings are always 0 and `sold_count` is demo data (reviews/checkout don't exist).
- No account recovery (email verification, password reset), no audit log, no admin UI for categories.
- Mobile layout verified only down to ~500 px; 390 px not verified (headless Edge limit).

**Technical debt**
- No frontend automated tests; browser E2E exists only as hand-run reference scripts (`docs/e2e-reference/`, not re-run from there, they pollute the dev DB).
- No CI; `golang:1.27-alpine` and `postgres:18` image tags float.
- `.env.example` documents the optional tuning variables only as comments; Compose forwards just 6 of 17 API variables.
- `frontend/README.md` is still the Vite template.

**Failed approaches / traps** (details in §11 of PROJECT_STATE): Git Bash path conversion breaks `docker run -w /src` (Makefile sets `MSYS_NO_PATHCONV=1`); the dev DB is called `main`, not `marketplace_dev` (a `sed` on that name wrote an admin into the dev DB); killing a process by port once killed the developer's Laravel app; sqlc needs `::uuid` casts, `COALESCE` on LATERAL columns and overrides for uuid/numeric; a registered `<select>` must mount after its options exist.

**Not verified**
- Behaviour with more than one API instance, with a reverse proxy, or over HTTPS.
- Anything after the last E2E run on the *committed* tree: the E2E scripts were run on the uncommitted CP-2 tree only.
- The Phase 4 design in PLAN.md is a proposal; it has not been prototyped.

---

## 6. Recommended next steps

1. **Developer reviews CP-2 and commits it** (3 commits suggested in `docs/PROJECT_STATE.md` §1). Decide whether to clean the dev DB test data (`DELETE` of `seller-…`, `e2e-…`, `demo@…`, `admin@…` accounts and their stores/products — explain the destructive step first, and take care that ledger and stock rows reference variants with `ON DELETE RESTRICT`, so delete in dependency order or re-create the dev DB with `make down` + volume removal only if the developer agrees).
2. **Exact next task — Phase 4 "Cart", step 1: answer Q1–Q5, then migration `000009_create_cart_tables`** (carts + cart_items) with up/down verified on dev and test DBs, followed by `db/queries/cart.sql` and `make sqlc`. Then the `modules/cart` backend, its integration tests, and only then the frontend (`/cart`, header badge, enabled add-to-cart). Stop for review at the end of the phase.
3. Before Phase 5 (checkout): decide the trusted-proxy/IP question for rate limiting, and promote the E2E scripts into a real Playwright suite that cleans up after itself.

**First 10 minutes in the new session**

```bash
git status --short | wc -l ; git log --oneline | head -3      # expect 66 uncommitted (or fewer if committed) and HEAD 316c1dc (or newer)
make ps                                                         # postgres healthy, api up; otherwise: make up
make test && make test-integration                              # all ok
cd frontend && npm run build && npm run lint                    # clean
```

Then re-read `docs/PROJECT_STATE.md` §8 for run instructions and §1 for the checkpoint trail, tell the developer the state in Indonesian, and **ask for approval + Q1–Q5 before writing code.**

---

## 7. Environment notes (this machine)

- Windows 11; shells: Git Bash and PowerShell. **Go is not installed on the host** — Go, sqlc and migrate run in Docker via the Makefile. `make` 4.4.1 came from scoop.
- Docker CLI: `C:\Users\Rava\AppData\Local\Programs\DockerDesktop\resources\bin\docker.exe` (may not be on PATH in agent shells).
- Ports: Compose Postgres **5433**; API **8081**; a local PostgreSQL 18 owns **5432**; another project of the developer's (Laravel) owns IPv4 **5173**. Agent-run Vite uses **5190**; developer uses `npm run dev -- --port 5174`. Before stopping any process by port, read its `CommandLine`.
- Credentials live in the developer's `.env` — never print or commit it. Local-only demo credentials are listed in this file and in CLAUDE.md.
