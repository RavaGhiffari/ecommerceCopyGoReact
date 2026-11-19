# PROJECT_STATE.md — living documentation + checkpoints

Where the project stands, how it fits together, and a trail of verified checkpoints.
**[CLAUDE.md](../CLAUDE.md) holds the rules** (conventions, domain rules, working agreement); **this file holds the state** (what exists, what was verified, what is next). Update this file at the end of every phase (see [How to update](#how-to-update-this-file)).

|  |  |
|---|---|
| Snapshot date | 2026-11-19 |
| Last commit | `0cb0fc8` (Phase 7c docs; 7c = `1a0e2e3`, `ee43cab`); `d8cac67` (7b CI workflow; `main` was 8 commits ahead of `origin` when checked, so not pushed); `b20d7ea` (Phase 7a docs; 7a = `bce9df0`, `cdc43ad`); `6b6d7d7` (Phase 7 plan docs); `d5b8577` (Phase 6e docs; 6e = `2c44965`; 6d docs `07bdc43`; 6d = `e00c95f`, demo orders in seed-dev `4ed91de`; loading skeletons `019fdbb`; Phase 6c = `0677906`, `6e18967`, docs `9716ed9`; wishlist practice feature `3d722a3` + `12bbd7f`; brand name `a1c4a27`; Phase 6b = `6f01036`, docs `b5fe739`; homepage carousel `932065a`; Phase 5e = `33e046e`, docs `6f10765`; Phase 5d = `2cf29fb`, docs `1dde9e0`; 5c = `870bcbe`, docs `c9caf97` — committed with the backend message by mistake, content is the docs; 5b = `9eec081`, docs `98bff00`; 5a = `f7ef72f`, `7c1d441`, docs `5efc212`; shadcn/ui migration = `db15fef`) |
| Uncommitted | **Slice 7d — security pass** (`docs/SECURITY_REVIEW.md`, `TRUSTED_PROXIES`, session cleanup, Secure-cookie test, security headers, race-free variant cap, page upper bound, `cn`/`shadcn` cleanup) + doc updates — see §1r |
| Current phase | **Phase 7 (hardening)**: plan approved 2026-11-17 with the Q1–Q8 defaults; 7a and 7c committed; 7b committed, **GitHub run not yet seen**; **7d done and verified, awaiting review** |
| Next | developer reviews and commits 7d; pushes and confirms green CI jobs + one red run; then 7e |

---

## 1. Checkpoints

A checkpoint is a state where **everything listed under "Verified" was actually run and passed**. Restoring one means `git checkout <commit>`, `make migrate-up`, and the verify commands in §8.

| # | Name | Commit | Status | Verified |
|---|---|---|---|---|
| CP-0 | Foundation | `0d45a70`…`b92f582`, `c4a5582` | ✅ reviewed | docker compose up; `/health` 200, `/ready` 200, and 503 with the DB stopped; migrate up→down→up; sqlc generate; unit + integration tests; frontend build/lint; graceful shutdown in logs |
| CP-1 | Identity & security | `759a82e`, `316c1dc` | ✅ reviewed | 13 auth integration tests on a real DB; curl register/login/me/logout/cross-site; 15-step browser E2E; logs contain no password/hash/token |
| CP-0.5 | UI prototype on mocks | `830c33d` (+ docs `d30b4ef`) | ✅ reviewed, superseded | 7 pages screenshotted incl. loading/empty/error; no component imported `mocks/`. Mocks were deleted in CP-2 |
| CP-2 | Catalog | `2838782` (backend), `d0c2a54` (frontend), docs `41679b9` | ✅ committed by the developer, who then approved Phase 4 | migrate up→down(4)→up; 13 unit packages; 12 catalog integration tests; concurrency test **fails without `FOR UPDATE`** (proved) and passes with it; ledger invariant = 0 mismatches; 17-step seller/catalog browser E2E + auth E2E re-run (no regression) |
| CP-4 | Cart | `7e57829`, `f312b0b`, docs `0d63dfc` | ✅ committed by the developer | see §1b |
| CP-UI | shadcn/ui migration (frontend only) | `db15fef`, docs `83b34a7` | ✅ committed by the developer | see §1c |
| CP-5a | Phase 5a addresses | `f7ef72f`, `7c1d441`, docs `5efc212` | ✅ committed by the developer | see §1e |
| CP-5b | Phase 5b checkout core | `9eec081`, docs `98bff00` | ✅ committed by the developer | see §1f |
| CP-5c | Phase 5c payment & lifecycle | `870bcbe`, docs `c9caf97` | ✅ committed by the developer | see §1g |
| CP-5d | Phase 5d seller orders | `2cf29fb`, docs `1dde9e0` | ✅ committed by the developer | see §1h |
| CP-5e | Phase 5e checkout & order UI | `33e046e`, docs `6f10765` | ✅ committed by the developer | see §1i |
| CP-6a | Phase 6a reviews API | `491953f`, docs `f8ab642` | ✅ committed by the developer | see §1j |
| CP-6b | Phase 6b reviews UI | `6f01036`, docs `b5fe739` | ✅ committed by the developer | see §1k |
| — | Wishlist (practice feature, PLAN.md §6.4b) | `3d722a3`, `12bbd7f` | ✅ committed by the developer | its 7 integration tests pass in the §1l run; no separate checkpoint |
| CP-6c | Phase 6c seller dashboard | `0677906`, `6e18967`, docs `9716ed9` | ✅ committed by the developer | see §1l |
| CP-6d | Phase 6d admin API (+ demo orders in seed-dev) | `4ed91de`, `e00c95f`, docs `07bdc43` | ✅ committed by the developer | see §1m |
| CP-6e | Phase 6e admin UI | `2c44965`, docs `d5b8577` | ✅ committed by the developer | see §1n |
| CP-7a | Phase 7a test suites | `bce9df0`, `cdc43ad`, docs `b20d7ea` | ✅ committed by the developer | see §1o |
| CP-7b | Phase 7b CI | `d8cac67` | 🟡 committed locally; GitHub run not seen | see §1p |
| CP-7c | Phase 7c performance & indexes | `1a0e2e3`, `ee43cab`, docs `0cb0fc8` | ✅ committed by the developer | see §1q |
| CP-7d | Phase 7d security pass | **uncommitted** | 🟡 awaiting review | see §1r |

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

### 1c. Verified at CP-UI (shadcn/ui migration), 2026-11-13

Developer decisions: Radix `Select` (not a styled native select), account menu as `DropdownMenu`, toast (`Sonner`) and `AlertDialog` deferred to Phase 5, migration done before Phase 5 planning.

| Check | Result |
|---|---|
| Setup | `components.json` (style `radix-nova`, CSS variables), `@/*` alias in `tsconfig.json`, `tsconfig.app.json`, `vite.config.ts`; `src/lib/utils.ts` (`cn` = `clsx` + `tailwind-merge`); theme tokens in `src/index.css` with `--primary`/`--ring` = emerald-600 |
| `npm run build` / `npm run lint` | clean (lint rule `only-export-components` off for `src/components/ui/**` only) |
| Browser E2E (Edge headless, Vite :5190, real API + dev DB) | `auth-flow.mjs` 15/15, `catalog-flow.mjs` 17/17 (incl. category shown after reload with Radix Select), `cart-flow.mjs` 16/16; scripts updated for the account menu and Radix Select |
| Logout race (see §11) | from `/account`: original code ended on `/login` 9 of 10 runs; with `navigate('/', { flushSync: true })` on `/` 10 of 10 |
| Dev DB after E2E | ledger mismatches 0; `SUM(reserved_quantity)` 0. The E2E scripts accidentally ran twice (one output discarded), so there are extra throw-away accounts |
| Screenshots | home, cart, product editor, open account menu, open category and sort selects, checked by eye |

Not checked: mobile widths, dark mode (tokens exist, no toggle; not a goal), keyboard-only walkthrough beyond what Radix provides.

Committed by the developer as `db15fef` (frontend) and `83b34a7` (docs).

### 1d. Dev housekeeping, 2026-11-14 (committed `ac4e163`)

| Change | Why | Verified |
|---|---|---|
| `docker-compose.yml`: ports published as `127.0.0.1:${POSTGRES_HOST_PORT:-5433}:5432` and `127.0.0.1:${API_HOST_PORT:-8081}:8081` | the dev DB and API were reachable from the local network (`0.0.0.0`) | `docker compose up -d` recreated both containers (volume kept, 30 products still there); `docker compose ps` shows `127.0.0.1:` bindings; `/ready` 200 on `127.0.0.1:8081`; a request to the host's non-loopback adapter address on :8081 gets no answer |
| `frontend/vite.config.ts`: `API_TARGET` = `http://127.0.0.1:8081` (was `localhost`) | Node may resolve `localhost` to `::1`, and Docker now listens on IPv4 loopback only | through Vite on :5190: `/ready` 200, `GET /api/v1/categories` 200, `POST /api/v1/auth/login` with a same-origin `Origin` 200 (CSRF Origin/Host check still passes because `changeOrigin` stays false) |
| Root `.gitignore`: `/node_modules/` | an accidental `npm install` at the repo root produced ~860 untracked files on 2026-11-13 | `git status` clean apart from the files listed here |

DB password: the developer set a 19-character `POSTGRES_PASSWORD` in `.env` (length checked, value never read). The existing volume still had the old password, so the API failed with `password authentication failed` until the role was updated from inside the container (`ALTER USER … PASSWORD :'pw'` over the local socket, with the value taken from the container's own environment). Afterwards `/ready` 200, 30 products and 35 users intact. Lesson in §11.

Suggested commit:

```bash
git add -- docker-compose.yml frontend/vite.config.ts .gitignore CLAUDE.md docs/PROJECT_STATE.md SESSION_HANDOFF.md
git commit -m "Bind dev ports to loopback and refresh handoff docs" -m "- Compose publishes Postgres and the API on 127.0.0.1 only" -m "- Vite proxies to 127.0.0.1:8081 so it does not depend on how localhost resolves" -m "- Ignore a stray root node_modules" -m "- SESSION_HANDOFF and PROJECT_STATE reflect the committed Phase 4 and shadcn/ui migration" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1e. Verified at CP-5a (addresses), 2026-11-14

| Check | Result |
|---|---|
| Migration `000010_create_user_addresses` | dev DB up → down → up (version 10); `marketplace_test` migrated by `make test-integration` |
| `make fmt` / `make vet` / `make test` | clean; new unit package `modules/addresses` (phone normalisation, field rules) |
| `make test-integration` | all packages ok; 10 new address tests: login required, default rules, validation + normalisation + strict JSON, partial update (PATCH cannot touch `isDefault`), delete promotes newest, limit 10 → 409, other user → 404 on every route, the DB refuses a 2nd default and a bad phone without the service, 2 concurrency tests |
| Lock proven necessary | with `LockUser` stubbed out, both concurrency tests failed in 3 of 3 runs: concurrent creates gave **500** (the partial unique index rejected a second default) instead of 409, and concurrent default switches gave 500 instead of 200. Restored afterwards |
| curl against the rebuilt API | create (phone normalised), list, 422 with every field, delete 204, empty list |
| Frontend | `npm run build` clean (JS 691 kB, +19 kB for Radix Dialog/AlertDialog), `oxlint` exit 0; shadcn `dialog` + `alert-dialog` added (declined overwriting `button.tsx`; `cn` import fixed) |
| Browser E2E | new `address-flow.mjs` 11/11 (empty state, client validation, first = default, new default, switch default, edit prefilled, AlertDialog cancel/confirm, promote on delete, reload, cleanup, no page errors); `auth-flow.mjs` 15/15 re-run (ran twice by accident: 2 extra throw-away accounts); screenshots checked by eye |

Not checked: mobile widths of the dialog; keyboard-only walkthrough beyond the Radix defaults.

**Commit CP-5a** (done: `f7ef72f`, `7c1d441`, `5efc212`):

```bash
git add -- backend/db/migrations/000010_create_user_addresses.up.sql backend/db/migrations/000010_create_user_addresses.down.sql backend/db/queries/addresses.sql backend/internal/database/queries backend/internal/modules/addresses backend/internal/router backend/cmd/api/main.go
git commit -m "Add Phase 5a address book API" -m "- user_addresses with a partial unique index (one default per user)" -m "- /me/addresses CRUD + set default; per-user row lock keeps the limit of 10 and the default switch race-free" -m "- Integration tests incl. ownership 404s and concurrency" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- frontend
git commit -m "Add address book to the account page" -m "- shadcn Dialog form and AlertDialog delete confirmation" -m "- Types, service and zod schema mirroring the API rules" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 5a" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1f. Verified at CP-5b (checkout core), 2026-11-14

| Check | Result |
|---|---|
| Migration `000011_create_checkout_tables` | dev DB up → down → up (version 11); `marketplace_test` migrated by `make test-integration` |
| `make fmt` / `make vet` / `make test` | clean; new unit packages `modules/shipping` (fees, Same Day rule, city compare), `modules/orders` (order number format, WIB date), `modules/checkout` (quote grouping, exact decimal totals, issue flags incl. own product, shipping validation, request hash), plus config (`CHECKOUT_TTL`, `SHIPPING_FEE_*`) |
| `make test-integration` | all packages ok; 12 new checkout tests: request rules (login, Idempotency-Key, strict JSON, every store needs a method), quote writes nothing, happy path with two stores (totals, orders, snapshots, reservation ledger, cart emptied, read endpoints), snapshots survive later product/store/address changes, idempotent repeat + reused key 409 + keys are per user, 5 concurrent submits with one key → 1×201 + 4×200, stock lowered → 409 with **every table count unchanged and the cart intact**, archived product / suspended store → 409, own store → 409, empty cart / foreign address / Same Day to another city, other buyers and the seller get 404, **10 buyers racing for 3 units → exactly 3×201 + 7×409, available 0, reserved 3, ledger invariant 0**, 5 rounds of two buyers with shared variants in opposite cart order → no deadlock |
| Concurrency tests repeated | the three concurrency tests passed 5 runs in a row (`-count=5`) |
| Inventory lock proven necessary | with `inventory.LockMany` removed, the 10-vs-3 test still sold exactly 3 (the conditional `UPDATE … WHERE available_quantity >= q` held) but 2–3 buyers per run got **500** instead of 409, in 3 of 3 runs. The lock gives the correct answer; the conditional update is the safety net. Restored afterwards |
| curl against the rebuilt API (dev DB, demo login `penjual3`) | quote (Jakarta Barat store → no Same Day), missing key 400, place 201, repeat 200, `GET /orders` shows `PK-20261114-…`, empty cart 422; ledger invariant on the dev DB = 0. The smoke address was deleted afterwards (the order kept its copy) |
| Frontend | unchanged in 5b (only the CORS allow-list gained `Idempotency-Key`, backend side) |

**Left in the dev DB on purpose:** one `pending_payment` checkout of `penjual3@contoh.id` holding 1 reserved unit. The 5c sweeper expired it on its first start (log `expired checkouts count=1`).

**Committed** as `9eec081` + `98bff00` with:

```bash
git add -- backend
git commit -m "Add Phase 5b checkout core" -m "- Migration 000011: checkouts, orders, order_items, order_status_histories, payments, payment_events; restock ledger type" -m "- POST /checkout/quote, POST /checkout (Idempotency-Key, one transaction: cart lock, inventory rows locked in variant order, server-side prices, per-store orders with snapshots, reservation + ledger, cart cleared)" -m "- GET /checkout/:id, GET /orders, GET /orders/:id (buyer only); shipping module (flat fees, Same Day same city)" -m "- Integration tests incl. rollback proof, idempotency, 10 buyers racing for 3 units, shared-variant deadlock check" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 5b" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1g. Verified at CP-5c (payment & lifecycle), 2026-11-14

| Check | Result |
|---|---|
| Migrations | none in 5c (000011 already had every table and the `restock` ledger type) |
| `make fmt` / `make vet` / `make test` | clean; new unit tests: the full order transition table (7 statuses × 7 × 3 actors, final statuses final), config `CHECKOUT_SWEEP_INTERVAL` (default 1m, `0s` rejected) |
| `make test-integration` | all packages ok; 9 new lifecycle tests: payment success (orders paid, reserved consumed, `sold_count` +, consumption ledger, history actor NULL; **same `eventId` again → 200 no-op**, same id with another outcome → 409 `PAYMENT_EVENT_CONFLICT`, new event after paid → 409 `CHECKOUT_NOT_PENDING`); payment failure (cancelled/failed, stock released); validation 422, other user 404, no login 401; payment after `expires_at` → 409 `CHECKOUT_EXPIRED` **and the expiry is committed** (stock released, no event stored); the sweeper expires only due checkouts and a second sweep is a no-op; buyer cancel (one unpaid order → 409 with `checkoutId`, other user 404, twice → 409, paying a cancelled checkout → 409); refund of a paid order (restock with `reference_type = 'order'`, `sold_count` −, the other store's order untouched, twice → 409, `processing` → 409 `{from: processing, to: refunded}`); **4 concurrent lifecycle requests (2 payments, 1 failure, 1 cancel) → exactly 1×200 + 3×409, 5 rounds**; **payment racing the sweeper on an expired checkout → payment always 409, exactly one release per line, 5 rounds**; ledger invariant 0 everywhere |
| Concurrency tests repeated | the 2 new and 3 existing concurrency tests passed 5 runs in a row (`-count=5`) |
| Checkout lock proven necessary | with `FOR UPDATE` removed from `LockUserCheckout` / `LockPaymentCheckout`, the 4-request race returned `{200:1, 500:3}` in 3 of 3 runs (the conditional `UPDATE … WHERE status = …` still kept the data correct, but the losers crashed instead of getting 409). Restored afterwards |
| Rebuilt API on the dev DB | the sweeper ran at start-up and expired the 5b leftover checkout; graceful restart logs `shutting down` → `shutdown complete` with no sweeper error |
| HTTP smoke test (`penjual3`, a product of another store) | place 201 → simulate success 200 `paid` → same event 200 "already processed" → new event 409 `CHECKOUT_NOT_PENDING`; `sold_count` 21000 → 21002; refund 200 `refunded`, again 409 `INVALID_STATUS_TRANSITION`; failure → `cancelled/failed`; cancel one unpaid order 409; cancel checkout 200, again 409. Afterwards stock 300 → 300 and `sold_count` back to 21000; dev ledger invariant 0, `SUM(reserved_quantity)` 0. Smoke address deleted |
| Frontend | unchanged in 5c |

**Committed** as `870bcbe` + `c9caf97` with:

```bash
git add -- backend
git commit -m "Add Phase 5c payment and order lifecycle" -m "- POST /payments/:id/simulate (idempotent by eventId), POST /checkout/:id/cancel, POST /orders/:id/cancel (paid -> refunded, restock)" -m "- Every lifecycle change locks the checkout row first; orders change status only through the transition table in orders/transitions.go" -m "- inventory Release / Consume / Restock with ledger rows; sold_count on payment and refund" -m "- Expiry sweeper in the API process (CHECKOUT_SWEEP_INTERVAL, FOR UPDATE SKIP LOCKED), stopped before the pool on shutdown" -m "- Integration tests incl. idempotent events, expiry, refund, racing lifecycle requests and payment vs sweeper" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 5c" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1h. Verified at CP-5d (seller orders), 2026-11-14

| Check | Result |
|---|---|
| Migrations | none (served by `orders_store_status_created_idx` from 000011) |
| `make fmt` / `make vet` / `make test` | clean; new unit tests for the seller request validation (status required / known, note ≤ 500 runes, list filter and page size) |
| `make test-integration` | all packages ok; 4 new tests: each store lists only its own orders (newest first, `?status=` filter, pagination, unknown status 422, detail with items/address/history, the other store's order and a malformed id → 404, buyer → 403, anonymous → 401); fulfilment paid → processing → shipped → delivered with a custom note, the buyer sees the full history, the seller is the actor, the buyer can no longer cancel; transition rules (unpaid → processing, skipping processing, `refunded`, `cancelled` → 409 with `{from, to}`; unknown/missing status 422; unknown JSON field 400; other store 404; nothing changed by refused requests; `delivered` is final); **seller processing racing the buyer's cancel → exactly one 200 + one 409, restock only when the refund won, 5 rounds** (both sides won in practice: refund 3–4 of 5) |
| Race tests repeated | the new race test and the two 5c/5b race tests passed 5 runs in a row |
| Order row lock proven necessary | with `FOR UPDATE` removed from `LockBuyerOrder` / `LockStoreOrder`, the loser got **500** instead of 409 in every round (the `WHERE status = @from_status` guard kept the data right). Restored and regenerated |
| HTTP smoke test (buyer `penjual3`, seller `penjual1` / Gadget Nusantara) | seller processing an unpaid order 409; after payment the order is in `?status=paid`; the buyer's own store gets 404 for it; skipping to shipped 409 `{from: paid, to: shipped}`; processing → shipped (note "Resi SIM-123") → delivered 200; buyer cancel 409; buyer history `pending_payment → paid → processing → shipped → delivered`. Dev ledger invariant 0 |
| Frontend | unchanged in 5d |

**Left in the dev DB on purpose:** one delivered order (`penjual3` bought 1 × "Kabel USB-C Braided" from Gadget Nusantara), so the seller list has data for 5e. Its stock and `sold_count` changed by 1 like a real sale.

**Committed** as `2cf29fb` + `1dde9e0` with:

```bash
git add -- backend
git commit -m "Add Phase 5d seller orders" -m "- GET /seller/orders?status=&page=&pageSize=, GET /seller/orders/:id, PATCH /seller/orders/:id/status {status, note?}" -m "- Store resolved from the session; every query filters by store_id, so another store's order is 404" -m "- Status changes lock the order row and go through the transition table as BySeller (paid -> processing -> shipped -> delivered)" -m "- Integration tests incl. ownership, transition rules and seller processing racing the buyer's cancel" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 5d" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1i. Verified at CP-5e (checkout, payment and order UI), 2026-11-15

| Check | Result |
|---|---|
| Backend | unchanged in 5e (no Go files, no migrations) |
| `npm run build` (`tsc -b && vite build`) / `npm run lint` (oxlint) | clean (Vite's existing "chunk > 500 kB" warning only) |
| New dependency | `sonner` (toast; the shadcn wrapper in `components/ui/sonner.tsx` is hand-written without `next-themes`, the app has no theme switch). shadcn `radio-group` added (uses the existing `radix-ui`) |
| Browser E2E `checkout-flow.mjs` (Edge headless, Vite :5190, real API + dev DB; buyer `penjual3`, sellers `penjual1`/`penjual2`) | **16/16**, 0 page errors, 3 consecutive runs: add to cart + "Beli sekarang" → /checkout; add a Surabaya address from the checkout page (selected); two store groups, Same Day offered only by Batik Kita (Surabaya), not by Gadget Nusantara (Jakarta Barat); choosing Same Day re-quotes and the summary equals `POST /checkout/quote`; **double-clicking "Buat pesanan" creates exactly one checkout with 2 orders** (order count +2); cart badge empties; simulated payment → "Lunas", both orders "Dibayar"; "Pesanan saya" lists both; seller `penjual1` filters "Perlu diproses", processes, ships with a resi note; buyer sees "Dikirim" + resi in the timeline and no cancel button; buyer refunds the other (paid) order; `penjual1` opening Batik Kita's order gets `ORDER_NOT_FOUND`; a new unpaid checkout is cancelled from the payment page; "Lanjut ke checkout" works from /cart |
| Regression E2E | `auth-flow` 15/15, `catalog-flow` 17/17, `cart-flow` 16/16, `address-flow` 11/11 in one full run. Two old scripts had timing bugs, fixed in the scripts (not the app): `cart-flow` clicked the header "Daftar" link (no `?next` on /login) when the card had not rendered yet; `address-flow` waited for `getByText('Utama')`, which also matches "Jadikan utama". Later repeats of `auth`/`catalog`/`cart-flow` hit `RATE_LIMITED` (register 10/hour per IP), expected backend behaviour |
| Visual check (screenshots) | fixed during the slice: order item images rendered full width (`ProductImage` is `w-full`; the parent must set the width, as in CartPage); default history notes are English in the DB and are shown in Indonesian via `historyNote()`; the payment card's "Dibayar" time label was renamed "Waktu bayar" (it clashed with the status badge); the stale homepage line "Keranjang dan checkout segera hadir" was replaced |
| Dev DB ledger invariant | 0 mismatches after all runs |

**Left in the dev DB:** the E2E orders of `penjual3` (one shipped order from Gadget Nusantara per run, refunded Batik Kita orders, cancelled checkouts). Two paid orders left by the first, failed run were refunded through the API so their stock returned.

**Committed** as `33e046e` + `6f10765` with:

```bash
git add -- frontend
git commit -m "Add Phase 5e checkout and order UI" -m "- /checkout: address picker (+ add new), per-store shipping radio, totals only from POST /checkout/quote, Idempotency-Key generated once per visit" -m "- /checkout/:id payment page: countdown to expiresAt, simulated success/failure (one eventId per outcome per visit), cancel with AlertDialog" -m "- /orders, /orders/:id (timeline, refund before processing), /seller/orders (status filter), /seller/orders/:id (next-status action with note)" -m "- Sonner toasts, shadcn radio-group, cart and Beli sekarang buttons enabled, TS types mirrored from checkout/orders DTOs" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 5e" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1j. Verified at CP-6a (reviews API), 2026-11-15

| Check | Result |
|---|---|
| Migration `000012_create_reviews` | dev DB (`main`) 11 → up 12 → down 11 → up 12; `marketplace_test` migrated by `make test-integration`. By hand in psql (rolled back): `rating = 6` refused by `reviews_rating_check`, `rating_total > rating_count × 5` refused by `products_rating_check`, a second review of an item refused by `reviews_order_item_key` |
| `make fmt` / `make vet` / `make test` | clean (`fmt` re-aligned two struct literals); new unit packages `platform/rating` (rounding) and `modules/reviews` (rating range, trimmed comment, 1000 runes not bytes, name masking incl. multi-byte and blank names) |
| `make test-integration` | all packages ok; 5 new tests: only the buyer's **delivered** items (paid and refunded → 409 with `orderStatus`; other buyer, the seller → 404; anonymous 401; 422 lists `orderItemId` + `rating`; unknown `productId` field 400; comment trimmed; product taken from the order; second review 409; pending list, my reviews, `review` on the order item, product and store ratings); public list + summary + edits (average 3.7, distribution, newest first, masked name, no email / user id in the body, pagination, edit moves the counters by the delta, comment-only edit keeps them, 6 refused edits change nothing, a hidden review leaves list and aggregate and its edit does not touch the counters, archived product → 404); `sort=rating` (best average first, unreviewed last); **20 concurrent reviews of one product → count 20, total 60**; **the same item reviewed twice at once → one 201 + one 409, 5 rounds**; rating invariant 0 in every test. The catalog test that expected `sort=rating` → 422 now checks an unknown value instead |
| Concurrency tests repeated | both new race tests passed 5 runs in a row (`-count=5`) |
| Atomic counter update proven necessary | with `AdjustProductRating` replaced by read-then-write in Go, the 20-review race ended with **count 6 / total 15–23** instead of 20 / 60 in 3 of 3 runs. Restored afterwards; the review tests pass again |
| Rebuilt API on the dev DB (`penjual3`) | `GET /me/reviews/pending` → 1 item (the delivered "Kabel USB-C Braided" from 5d); review 201, again 409 `REVIEW_EXISTS`; public list shows "Agus W***", summary 5.0 / 1 with distribution; `sort=rating` puts it first; Gadget Nusantara store rating 5.0 / 1; pending 0 afterwards. Rating invariant 0, stock ledger invariant 0 |
| Frontend | unchanged in 6a; new response fields (`rating` values, `review` on order items) are additive |

**Left in the dev DB on purpose:** one 5-star review by `penjual3@contoh.id` of "Kabel USB-C Braided" (Gadget Nusantara), as data for 6b.

**Committed** as `491953f` + `f8ab642` with:

```bash
git add -- backend
git commit -m "Add Phase 6a reviews API" -m "- Migration 000012: reviews (one per order item, rating 1-5, visible/hidden) and products.rating_count / rating_total with a CHECK" -m "- POST /reviews (only items of the caller's delivered orders), PATCH /reviews/:id, GET /products/:id/reviews (summary, distribution, masked names), GET /me/reviews, GET /me/reviews/pending" -m "- Counters updated atomically in the review transaction; real ratings on product list/detail and store page; sort=rating; order items carry their review" -m "- Seller product list sorted by created_at so counter updates no longer reorder it" -m "- Integration tests incl. ownership, invariant checks and two race tests" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 6a" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1k. Verified at CP-6b (reviews UI), 2026-11-15

| Check | Result |
|---|---|
| Backend (additive) | `ListOrderItems` also returns `r.comment`; order items carry `review: {id, rating, comment}`; `make fmt` / `make vet` / `make test-integration` all ok (the 6a test asserts the new field) |
| `npm run build` / `npm run lint` | clean (Vite's existing "chunk > 500 kB" warning only); two oxlint warnings from the first draft fixed (`STAR_LABELS` no longer exported from a component file, `watch` → `useWatch`) |
| Browser E2E `review-flow.mjs` (Edge headless, Vite :5190, real API + dev DB; buyer `penjual3`, seller `penjual1`) | **11/11**, 0 page errors: pending item on `/account`; the dialog refuses 0 stars; 4 stars + comment → toast, the item moves to "Ulasan saya"; product page shows it with "Agus W***", summary +1, no `@` in the section; the title rating jumps to `#ulasan`; `/orders/:id` "Ubah ulasan" opens prefilled (4 stars + comment), edit to 2 stars → toast, stars updated without a page reload; product page shows "· diubah" and the new text; an undelivered order has no review button; "Rating tertinggi" sort; a second review via the API → 409 |
| Bug found by the E2E | the visually hidden radio (`sr-only` = `position: absolute`) sat outside its star because the `<label>` was not `relative`; focusing it could scroll the page. Fixed (`relative` on the label); positions checked in the browser (each input inside its star). Two other failures were selector bugs in the script (clicking the hidden input during the dialog animation instead of the star; `/diubah/` also matching the comment) |
| Regression E2E | `address-flow` 11/11 (account page), `checkout-flow` 16/16 (order pages) |
| Visual check (screenshots) | product review block (summary, distribution bars, list), `/account` review cards, order page with stars + "Ubah ulasan" |
| Dev DB | rating invariant 0, stock ledger invariant 0 |

**Left in the dev DB:** 4 E2E reviews of "Kabel USB-C Braided" by `penjual3` (one per run; texts start with "Ulasan E2E"), on orders delivered for the purpose, plus the orders from `checkout-flow`. Reviews cannot be deleted by design; hiding them is a 6d admin action.

**Commit CP-6b** (the developer commits):

```bash
git add -- backend frontend
git commit -m "Add Phase 6b reviews UI" -m "- Review dialog (star picker on native radios, RHF + zod, 1000-character counter) from /orders/:id and /account" -m "- /account: 'Menunggu ulasan' and 'Ulasan saya' (hidden-by-admin note); product page: summary, distribution and paginated reviews" -m "- 'Rating tertinggi' sort option; rating next to the title links to the reviews" -m "- Order items carry the review comment so editing starts from the saved text" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 6b" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1l. Verified at CP-6c (seller dashboard), 2026-11-15

| Check | Result |
|---|---|
| `make sqlc` / `make fmt` / `make vet` / `make test` | ok |
| `make test-integration` | all packages ok (router 11 s), including the wishlist tests; the 3 new tests run by name with `-v`: `TestSellerStats_EmptyStore`, `TestSellerStats_MatchesHandComputedNumbers`, `TestSellerStats_Access` all PASS |
| What the fixture test proves | 5 checkouts for store A (delivered, pending, refunded, paid with a second product, paid but backdated 40 days): status counts; revenue 30 d `450002.00` / all `650003.00` (items only, pending + refunded excluded, shipping excluded); units 9 / 11; top products ordered by units with the old and refunded orders left out; rating 4.5 from the counters; store B's numbers contain only its own orders and not A's rating |
| `npm run build` / `npm run lint` | clean (existing "chunk > 500 kB" warning only; bundle 800 kB / gzip 240 kB) |
| Real API (`make up`, curl against :8081) | `/ready` 200; `penjual1` login 200 → `GET /seller/stats` 200 with all 7 statuses at 0, `"0.00"` money, `topProducts: []`; anonymous → 401 |
| **Not verified** | the dashboard in a browser (no screenshot / E2E run); the dev DB has one order only, owned by an account whose password is not in the repo, so the panel was not seen with real numbers |

**Commit CP-6c** (the developer commits):

```bash
git add -- backend frontend
git commit -m "Add Phase 6c seller dashboard" -m "- GET /seller/stats: orders per status, revenue and units (30 days and all time), top 5 products, store rating; store from the session" -m "- Revenue = items_subtotal of paid/processing/shipped/delivered orders; computed on read, no new table" -m "- /seller: stat tiles, orders per status (links to the filtered list), best sellers table" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 6c" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```


### 1m. Verified at CP-6d (admin API), 2026-11-16

| Check | Result |
|---|---|
| `make sqlc` / `make fmt` / `make vet` / `make test` | ok (gofmt reformatted 4 new files) |
| `make test-integration` | all packages ok; the 10 admin tests run by name with `-v`, all PASS: every admin route 401 anonymous / 403 customer / 403 seller (11 routes); suspend → session dead at once, `ACCOUNT_SUSPENDED` on login, sessions deleted, repeat = no second audit row; reactivate → old session still dead, new login works; self / other admin / unknown / bad body refused with **0 audit rows**; store suspend → out of `/products`, cart line `unavailable`, store page 404, the existing order still moves to processing; hide → average 3 → 4, public list 1 item, author still sees it, rating invariant 0; unhide → back, `edited: false`; author edit while hidden leaves the counters alone; stats match the fixture (users 3/1, stores 1/1, 1 public product, GMV 30 d `200001.00` / all `350001.00`); platform orders list/filter/detail; audit log newest first with before/after details; audit row rolls back with a failing action; FK refuses an unknown actor |
| Migration 000015 is needed (proved) | with the test DB at version 14, `TestAdmin_HideAndUnhideReview` **fails** (`edited` true after unhide); at 15 it passes |
| Migrations | test DB down 2 → up 2 clean; dev DB migrated to 15 |
| Real API (`make up`) | `/admin/stats` anonymous 401, demo seller 403 on GET and PATCH. The dev DB admin (`admin@contoh.id`) was created by an E2E script and its password is not in the repo, so admin success paths were checked by the tests only |
| `seed-dev` demo orders | ran on the dev DB: "categories already exist; skipping the demo catalog", 8 demo checkouts created; a second run is a no-op. `penjual1` stats: every status filled, revenue 30 d `1482000.00` (= 349000 + 258000 + 117000 + 259000 + 499000, hand-checked) / all `2180000.00`, rating 4.5 (2). Dev DB stock-ledger and rating invariants: 0 mismatches |
| Frontend skeletons (committed `019fdbb`) | `npm run build` / `npm run lint` clean; **not seen in a browser** (look with DevTools throttling) |

**Commits CP-6d** (the developer commits; two separate changes, then docs):

```bash
git add -- backend/cmd/seed-dev Makefile
git commit -m "Seed demo orders and reviews in seed-dev" -m "- Separate, run-once step: skipped when the demo buyer exists, so an already seeded DB still gets it" -m "- Goes through cart, checkout, payment, seller transitions and reviews; only created_at is moved into the past" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- backend
git commit -m "Add Phase 6d admin API" -m "- Migration 000014 audit_logs (append-only, written in the same transaction as each admin action)" -m "- Migration 000015: reviews.updated_at only on rating/comment changes, so moderation does not mark a review as edited" -m "- Users and stores: list, search, suspend/reactivate (not self, not another admin; suspend ends sessions)" -m "- Review hide/unhide moves the rating counters; platform orders (read-only); platform stats; audit log; category writes audited" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 6d" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1n. Verified at CP-6e (admin UI), 2026-11-16

| Check | Result |
|---|---|
| `npm run build` / `npm run lint` | clean (bundle 838 kB, gzip 248 kB; the >500 kB warning is the known Phase 7 item) |
| `make fmt` / `make vet` | clean (only `cmd/seed-dev/orders.go` changed on the backend) |
| Browser E2E `admin-flow.mjs` (headless Edge, real API + dev DB, admin from `make seed-admin`) | **16/16 PASS**: seller sees "Halaman khusus admin" and no header link; anonymous `/admin/stores` → `/login?next=…`; admin header link → dashboard tiles; "Selesai" on the dashboard → `/admin/orders?status=delivered`; order detail read-only; user search, own row "(kamu)" without action; suspend: AlertDialog, "Batal" changes nothing, confirm → "Ditangguhkan" and the user's open session answers 401 at once; reactivate; store suspend → public store 404, no link in the row, reactivate → 200; status filter from the URL; category edit (sortOrder 777) and revert; category form: client errors (empty name, bad slug) and server 409 `SLUG_TAKEN` on the slug field, nothing created; review hide (AlertDialog) → "Disembunyikan" → show again; audit log shows all 7 action labels and "N sesi diakhiri"; 390 px: no horizontal page scroll on users and reviews (tables scroll inside their card); no uncaught page errors |
| Screenshots | dashboard, audit log, users and reviews at 390 px, read in the session |
| Bug found and fixed in the 6d demo seed | `seed-dev` moved only `created_at` of demo reviews into the past, so `updated_at > created_at` and every demo review showed "(diubah)". Seed now moves `updated_at` too; the 4 existing demo reviews in the dev DB were corrected with one `UPDATE … SET updated_at = created_at` limited to `pembeli.demo@contoh.id` (allowed by 000015: the trigger only fires on rating/comment) |
| Left in the dev DB | admin `e2e-admin@contoh.id` (random password, kept outside the repo) and the audit rows of the E2E runs (append-only by design). Every other change was reverted by the script |
| Not verified | keyboard-only walk through the dialogs (Radix handles focus, not re-checked by hand); dark mode; Safari/Firefox |

**Committed** as `2c44965` + `d5b8577` with:

```bash
git add -- frontend/src backend/cmd/seed-dev/orders.go
git commit -m "Add Phase 6e admin UI" -m "- /admin behind RequireAdmin (UX only): dashboard, users, stores, categories, orders (+ read-only detail), reviews, audit log" -m "- Search, status filter and page in the URL; suspend and hide behind AlertDialog; rows updated from the PATCH answer" -m "- Category create/edit dialog (RHF + zod, server field errors on the fields); header link for admins" -m "- seed-dev: move demo reviews' updated_at with created_at so they do not show as edited" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 6e" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1o. Verified at CP-7a (test suites), 2026-11-17

| Check | Result |
|---|---|
| `make fmt` / `make vet` / `make test` | ok (gofmt re-aligned `cmd/api/main.go`); new `TestLoad_RateLimits` pins the production defaults 5 / 15 min / 10 / 1 h and rejects 0, negative and non-numbers |
| `make test-integration` | all packages ok (router 12 s); new `TestRegister_RateLimitedPerIP` (3rd sign-up → 429 `RATE_LIMITED` + `Retry-After`, nothing created). There was no register-limit test before |
| `npm run build` / `npm run lint` | clean; bundle unchanged (838 kB, tests are not bundled); `tsc -b` now also type-checks `e2e/` (`tsconfig.e2e.json`) and `vitest.config.ts` |
| `npm run test` (Vitest 5 + RTL + jsdom) | **11 files, 107 tests pass**, twice in a row: format, `lib/api` (envelope, 204, network error, abort, non-JSON), `safeNextPath`, server error translation + RHF field mapping, every zod schema's edges, order-status UI copy, URL parsing, the 3 route guards, `ProductCard`, `CartPage` (loading / error / empty / issue flags / blocked checkout), `OrdersPage` (loading / empty / error / list) |
| Unit tests catch a regression (proved) | removing the `//` check from `safeNextPath` → `falls back to / for //evil.example` fails; restored |
| `make e2e` (Playwright 1.65, Edge, fresh `marketplace_e2e` each run) | **3 consecutive full runs, 10/10 tests each, ~1.5 min per run**: auth, catalog (public + seller journey), cart, address, checkout, review, wishlist, admin (admin spec ran, not skipped), and a new 390 px check of home / listing / product / cart / checkout (no horizontal scroll). Every spec also fails on any uncaught page error |
| Dev DB untouched (proved) | `main`: users 52, orders 14, reviews 4, audit rows 22, ledger rows 168, newest user 2026-11-16 — identical before and after the 3 runs |
| E2E DB after a run | stock + reserved ledger invariant 0 mismatches |
| Reruns without a reset | review + wishlist specs re-run on the same E2E DB: pass (every spec signs up its own customer) |
| Not verified | CI (7b); Firefox/Safari; the suite with `retries` (local runs use 0) |

Fixed during the slice (test code, not the app): two ambiguous selectors (`1 ulasan` also matched the screen-reader text of the rating bars; the t-shirt page has hearts on "Produk terkait" cards too) and one timing bug (the admin spec read a skeleton row's text). Findings for later slices: `npm audit` reports 4 moderate advisories, all through `shadcn` → `ts-morph` → `minimatch` (7d).

**Commit CP-7a** (the developer commits):

```bash
git add -- backend Makefile docker-compose.yml scripts/reset-e2e-db.sh .env.example
git commit -m "Add E2E stack and configurable rate limits" -m "- RATE_LIMIT_* config (defaults = the old hard-coded limits, pinned by a test); register limit now has an integration test" -m "- Compose profile e2e: api-e2e on :8082 with its own database marketplace_e2e and raised limits" -m "- make e2e / e2e-reset / e2e-up / e2e-down: drop, migrate, seed-dev, seed-admin with a random password, run Playwright" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- frontend
git commit -m "Add Vitest and Playwright suites" -m "- Vitest + React Testing Library: api client, formatting, schemas, server errors, guards, ProductCard, CartPage, OrdersPage (107 tests)" -m "- Playwright suite in frontend/e2e ported from the hand-run scripts: auth, catalog, cart, address, checkout, review, wishlist, admin, 390 px" -m "- Fresh customer per spec, page-error watch on every page, Vite on :5191 proxying to api-e2e" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- CLAUDE.md PLAN.md PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 7a" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### 1p. CP-7b (CI), written 2026-11-18

| Check | Result |
|---|---|
| `.github/workflows/ci.yml` parses | [verified] YAML loads; jobs `backend`, `frontend`, `e2e`; triggers push, pull_request (main), workflow_dispatch |
| Review against the repo | [verified] `.env.example`, `scripts/reset-e2e-db.sh`, `make e2e-reset` / `e2e-up`, `npm run e2e` exist; `playwright.config.ts` switches to bundled Chromium when `CI` is set; sqlc 1.29.0 and migrate v4.18.3 match the Makefile; Go version read from `go.mod` |
| Jobs | `backend`: gofmt, `go vet` (+ `-tags=integration`), `sqlc diff`, migrate up → down → up, unit tests, integration tests (`-p 1`, Postgres 18 service), `govulncheck` (non-blocking). `frontend`: `npm ci`, lint, Vitest, build, `npm audit --omit=dev` (non-blocking). `e2e`: only on `main` / manual; random `.env` password, `make e2e-reset e2e-up`, Playwright, logs + report artifact on failure |
| **Not verified** | **No run on GitHub yet.** `actionlint` was not run (image pull was denied). Linux-specific risks in the `e2e` job: `make e2e-reset` (seed-dev, seed-admin via `docker compose run`), Docker network/healthcheck timing |

Done when (PLAN.md §7.1): a pushed branch shows all jobs green, and a deliberately broken test turns it red (shown once, then reverted).

**Commit CP-7b** (the developer commits and pushes):

```bash
git add -- .github/workflows/ci.yml
git commit -m "Add GitHub Actions CI" -m "- backend: gofmt, vet, sqlc diff, migrations up/down/up, unit + integration tests on Postgres 18" -m "- frontend: lint, Vitest, build; npm audit and govulncheck non-blocking" -m "- e2e: Compose e2e stack + Playwright on main and manual runs" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"

git add -- PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 7b" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"

git push -u origin main   # or a branch + PR; watch the Actions tab
```

### 1q. Verified at CP-7c (performance & indexes), 2026-11-18

**Method.** `make seed-load` drops and recreates a separate database `marketplace_perf` (never dev/test/e2e), migrates it and bulk-loads 5 200 users, 200 stores, 50 000 products, 200 000 variants + inventory, 75 000 images, 100 000 orders / 200 000 order items and 16 666 reviews in ~25 s (`scripts/seed-load.sql`, deterministic ids). `make perf-explain` PREPAREs every hot query with the same text as `backend/db/queries/` and runs `EXPLAIN (ANALYZE, BUFFERS) EXECUTE` (as pgx does); `make perf-explain generic=1` forces generic plans. Numbers below are a **warm** run (run it twice; the first run after a restart is slower). Plans: `docs/perf/plans-7c-final.txt` (final), `plans-7c-before-indexes.txt` (after the query rewrite, before migration 000016), `plans-7c-original-catalog.txt` (the original catalog query; first run, partly cold, but 1.3 M buffer hits, so it was CPU-bound, not a cache artefact).

| Finding | Evidence | Fix |
|---|---|---|
| **`ListPublicProducts` built every product before `LIMIT`** because of `COUNT(*) OVER ()` | 47 500 rows × (variant aggregate + image lookup): **2 195 ms** newest, 1 665 ms popular, 1 385 ms rating | the total moved to a new `CountPublicProducts` (no sort, no per-row lookups); the page query walks `products_status_created_idx` and stops after one page: newest **2.7 ms**, popular 6 ms, category 0.9 ms, store page 1.7 ms, related 0.9 ms. Page + count together ≈ 50 ms (the count is 47 ms) |
| Price sort needed every product's price before page 1 | LATERAL per product: **1 227 ms**; image lookup inside the sorted query: ~1 010 ms | separate `ListPublicProductsByPrice`: one `GROUP BY` over all variants + a `page` CTE so images are fetched for 24 rows only: **213 ms** (category 167, search 157). The repository converts its row type into the other (identical columns) |
| Admin lists had no index for `ORDER BY created_at DESC` | Parallel Seq Scan over 100 000 orders + top-N sort: orders 32 ms, orders `?status=` 14 ms, reviews 34 ms | migration `000016`: `orders_created_idx (created_at DESC, id)`, `orders_status_created_idx (status, created_at DESC, id)`, `reviews_created_idx (created_at DESC, id)` → **0.040 ms, 0.049 ms, 0.116 ms** |
| Bundle was one 838 kB chunk (Vite warning) | `npm run build` | route `lazy` for cart, checkout, payment, orders, account, wishlist, seller and admin (guards stay eager; admin layout and pages lazy) + vendor groups (`react`, `router`, `forms`): entry chunk **190 kB** (gzip 57 kB), `react` 219 kB, `router` 95 kB, `forms` 133 kB, ~40 route chunks of 2–18 kB; **warning gone**. `HydrateFallback` (`app/FirstLoad.tsx`) covers a bookmarked lazy URL |

Recorded, needs nothing: `ListCartLines` 0.06 ms, `ListOrderItems` 0.4 ms, buyer orders 0.05 ms, seller orders 1.1 ms (0.07 ms with `?status=`), seller counts 0.08 ms, seller status counts 0.09 ms, top products 0.7 ms, store rating 0.5 ms, review distribution 0.1 ms, public reviews 1.4 ms, pending reviews 0.09 ms.

**Accepted (recorded, not fixed)**

| Query | Time | Why it stays |
|---|---|---|
| `sort=rating` | 85 ms | sorts all 47 000 products by a computed average; an expression index is possible but it is a minority sort |
| Price sorts | 157–213 ms | the real fix is a denormalised `min_price` on `products` kept in sync by variant writes: a design decision, not an index |
| `CountPublicProducts` (47 ms); admin stats (public-product count 43 ms, GMV 13–16 ms); `CountAdminOrders` 11 ms | | aggregates over the whole table; admin-only or one extra query per page; cache later if needed |
| Seller revenue | 45 ms for a store with 500 orders | one `order_items` lookup per order (linear in that store's orders); fine at demo scale |
| `ListSellerOrders` without status | 1.1 ms | sorts one store's orders; a seller with 50 000 orders would want `(store_id, created_at DESC, id)` |

**Generic plans.** With `generic=1` the catalog query cannot use the index (1.6 s): the sort key is a `CASE` on a parameter, which the planner can only fold when the value is a constant. pgx's default is `plan_cache_mode=auto`, which keeps custom plans while the generic plan's estimated cost is much higher than the custom average (it is, here). Not reproduced end to end against the API.

| Check | Result |
|---|---|
| `make sqlc` | regenerated; `products.sql.go` changed only for the catalog queries |
| `make fmt` / `make vet` / `make test` | ok |
| `make test-integration` | all packages ok (router 17.9 s); `TestPublicListing_SortSearchFilterPaginate` covers price asc/desc, page 2, total 3 and "total is still reported past the last page" through the new count and price query |
| Migration 000016 | up on `marketplace_perf` and on the test DB ok; **down → up not run** (CI's backend job does it) |
| `npm run build` / `lint` / `test` | clean, no chunk warning; 107 tests pass |
| `make e2e` | 10/10 twice after the router change; after adding `HydrateFallback` the "No `HydrateFallback`" console warning is gone |
| Not verified | the API under real load (only SQL plans); generic-plan behaviour through pgx; lazy chunks over a slow network |

Pre-existing, not from this slice: the Playwright run logs "Select is changing from uncontrolled to controlled" warnings from a shadcn Select.

**Commit CP-7c** (the developer commits):

```bash
git add -- backend scripts Makefile docker-compose.yml docs/perf
git commit -m "Speed up catalog and admin list queries" -m "- ListPublicProducts no longer counts with a window function (2.2 s -> 3 ms); CountPublicProducts added" -m "- ListPublicProductsByPrice: grouped aggregate + page CTE (1.2 s -> 0.2 s)" -m "- migration 000016: indexes for the admin order and review lists (30 ms -> 0.1 ms)" -m "- make seed-load / perf-explain on a separate marketplace_perf database" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"

git add -- frontend
git commit -m "Split the frontend bundle by route" -m "- route-level lazy loading for cart, checkout, orders, seller and admin; vendor chunks; HydrateFallback" -m "- entry chunk 838 kB -> 190 kB, Vite size warning gone" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"

git add -- PHASES.md SESSION_HANDOFF.md docs/PROJECT_STATE.md
git commit -m "Update project docs for Phase 7c" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

### 1r. Verified at CP-7d (security pass), 2026-11-19

The findings table, OWASP mapping and decisions are in [SECURITY_REVIEW.md](SECURITY_REVIEW.md). In short:

| Change | Files | Proof |
|---|---|---|
| `TRUSTED_PROXIES` (IPs/CIDRs; empty = trust nobody; `0.0.0.0/0` and `::/0` refused) | `config/config.go`, `router/router.go` | `TestClientIP_TrustedProxies`, `TestLoad_TrustedProxies`, `TestRegister_RateLimitKeysOnTheRealClientBehindATrustedProxy` |
| Expired-session cleanup: query `DeleteExpiredSessions` (batched), `auth/cleanup.go` (`CleanupExpiredSessions`, `RunSessionCleanup`), its own goroutine in `main.go` (`sync.WaitGroup`, waited for before `pool.Close`), `SESSION_CLEANUP_BATCH` (500) | `db/queries/sessions.sql`, `auth/`, `cmd/api/main.go` | `TestCleanupExpiredSessions`; on `api-e2e` start 26 expired sessions → 0. No index (measured: 5–10 ms per run at 208 k sessions) |
| `config.SecureCookies()`, used by `main.go` and the test app | `config/config.go` | `TestSecureCookies_FollowAppEnv`, `TestSessionCookie_SecureOnlyInProduction` |
| `middleware.SecurityHeaders` (nosniff, DENY, CSP `default-src 'none'; frame-ancestors 'none'`, Referrer-Policy, `Cache-Control: no-store`) right after RequestID | `middleware/security_headers.go`, `router/router.go` | `TestSecurityHeaders_OnSuccessAndOnAbort`, `TestSecurityHeaders_OnEveryKindOfResponse`; `curl -i` on the dev API shows all five |
| `AddVariant`: lock the product row, then count, inside the tx | `db/queries/products.sql` (`LockStoreProduct`, `CountProductVariants`), `products/repository.go`, `products/service_seller.go` | `TestAddVariant_CapHoldsUnderConcurrency`: **26 variants on the old code (3/3 runs)**, 20 with the fix (3/3) |
| **New finding:** no `page` upper bound in orders/admin → negative OFFSET → 500 | `orders/service.go`, `admin/service.go` (`MaxPage = 10_000`); messages in `products`, `reviews` corrected to 6000 | `TestEveryListEndpoint_RejectsOutOfRangePaging` (12 endpoints): 6 returned 500 before the fix |
| `cn` removed, `shadcn` → devDependencies | `frontend/package.json`, `package-lock.json` | `npm audit --omit=dev`: 4 moderate → 0; the lockfile changes only `dev` flags and the `cn` entry |
| `.env.example` documents `TRUSTED_PROXIES`, `SESSION_CLEANUP_BATCH` | `.env.example` | — |

| Check | Result |
|---|---|
| `make sqlc` | only `products.sql.go` and `sessions.sql.go` changed |
| `make fmt` / `make vet` / `go vet -tags=integration` | ok |
| `make test` | 20 packages ok |
| `make test-integration` | all packages ok (router 26 s) |
| `npm run build` / `lint` / `test` | clean, entry chunk 190 kB, 107 tests |
| `make e2e` | 10/10, then `npm run e2e` again 10/10 |
| No migration | none needed (no new index) |
| Not verified | `TRUSTED_PROXIES` through a real nginx (7e); Secure cookie over real HTTPS; cleanup with several API instances |

Deviation from the plan: session cleanup is its own goroutine owned by `auth`, not a second task inside the checkout sweeper (keeps `checkout` free of session knowledge); it shares the interval and the shutdown wait. Added beyond the plan: `Cache-Control: no-store` and the page upper bound (a found bug).

**Commit CP-7d** (the developer commits):

```bash
git add -- backend .env.example
git commit -m "Harden the API: proxies, sessions, headers, limits" -m "- TRUSTED_PROXIES so ClientIP (rate limits) is the real client behind nginx" -m "- background cleanup of expired sessions (batched, SESSION_CLEANUP_BATCH)" -m "- security headers on every response; Secure cookie tested in production" -m "- AddVariant locks the product before counting: the 20-variant cap held 26 under concurrency" -m "- page upper bound on order/admin lists (negative OFFSET gave a 500)" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- frontend/package.json frontend/package-lock.json
git commit -m "Drop unused cn package, move shadcn to devDependencies" -m "npm audit --omit=dev: 4 moderate -> 0" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"

git add -- docs/SECURITY_REVIEW.md docs/PROJECT_STATE.md PLAN.md PHASES.md SESSION_HANDOFF.md CLAUDE.md
git commit -m "Add security review and update project docs for Phase 7d" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

---

## 2. What exists, by phase

| Phase | Scope | State |
|---|---|---|
| 0 Foundation | `backend/` layout, config, pgxpool, JSON logging, request ID, CORS, graceful shutdown, `/health` `/ready`, golang-migrate, sqlc, Makefile, Compose, status page | ✅ |
| 0.5 UI prototype | Catalog pages on typed mock data | ✅ (now on real API) |
| 1 Identity | users, roles, sessions; register/login/logout; Argon2id; CSRF layers; rate limits; `/me`; `seed-admin` | ✅ |
| 2 Catalog | categories (2 levels), stores, products, variants, images (URLs), inventory + ledger, public listing/search, seller dashboard, `seed-dev` | ✅ |
| 3 Marketplace UI | Homepage, nav, cards, detail, store page, search on real APIs | ≈ done as a side effect of CP-2; polish (banner, popular/recommended refinements, mobile check) remains |
| 4 Cart | carts + items, stock-checked add/update/remove (no reservation), issue flags, `/cart` page, header badge, add-to-cart | ✅ committed |
| 5 Checkout & orders | addresses, shipping sim, transactional multi-store checkout, order lifecycle, payment sim, checkout/payment/order UI | ✅ committed |
| 6 Reviews & dashboards | reviews, rating aggregates, buyer/seller/admin dashboards | ✅ 6a–6e committed |
| 7 Hardening | integration + E2E suites, security review, query/index review, OpenAPI, deploy | 🟡 plan approved; 7a committed; 7b (CI) committed, GitHub run not seen; 7c (performance & indexes) done, awaiting review; 7d–7e open |

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

Frontend routes (`src/app/router.tsx`): `/`, `/products`, `/categories/:categorySlug`, `/stores/:storeSlug`, `/stores/:storeSlug/products/:productSlug`, `/login`, `/register`, `/account` (RequireAuth), `/seller` (RequireAuth; opens a store or shows the dashboard), `/seller/products/new` and `/seller/products/:id` (RequireSeller), `/cart` (RequireAuth), `/admin` with `users`, `stores`, `categories`, `orders`, `orders/:id`, `reviews`, `audit-logs` (6e; `RequireAdmin` + `AdminLayout` section bar), `/status`, `*`. "Tambah ke keranjang" works (anonymous → `/login?next=…`); **"Beli sekarang" and "Lanjut ke checkout" stay disabled** until Phase 5. `CartProvider` (inside `AuthProvider`) holds the server's cart for the badge, product page and `/cart`.

Runtime dependencies — Go: gin, pgx/v5, pgx-shopspring-decimal, shopspring/decimal, google/uuid, x/crypto (argon2), x/text (slug NFD), testify (tests). Frontend: react 19, react-router-dom, react-hook-form, zod 4, @hookform/resolvers, lucide-react, tailwindcss 4.3.5 (+ vite plugin), and for **shadcn/ui** (developer's choice): `radix-ui` (primitives), `class-variance-authority` (variants), `clsx` + `tailwind-merge` (`cn()`), `tw-animate-css` (open/close animations), `shadcn` (its `tailwind.css` is imported by `index.css`, so it stays a runtime dependency). The npm package `cn` and the Geist font were removed.

**shadcn/ui components** live in `src/components/ui/` (generated, then owned by us). Add more with `npx shadcn add <name>`. **The CLI writes `import { cn } from "cn"` into new files** even though `components.json` points at `@/lib/utils`. After every `add`, run `sed -i 's#from "cn"#from "@/lib/utils"#' src/components/ui/*.tsx`; otherwise the build fails. The wrappers in `components/shared/` (`TextField`, `TextAreaField`, `FieldMessage`, `StateViews`, `Pagination`) keep their old props, so pages did not change. `CategorySelect` is now controlled (`value`/`onChange`/`ref`) and must be used through RHF `<Controller>`. Pinned tool images: `sqlc/sqlc:1.29.0`, `migrate/migrate:v4.18.3`; `golang:1.27-alpine` and `postgres:18` are floating tags.

---

## 4. API surface (as of CP-6d)

Base `/api/v1`. Envelope: `{success, data, message}` / `{success:false, error:{code,message,details}}`.

| Group | Routes |
|---|---|
| Infra (no prefix) | `GET /health` (no DB), `GET /ready` (DB query) |
| Public | `POST /auth/register`, `/auth/login`, `/auth/logout` · `GET /categories` · `GET /products?q&category&store&sort&page&pageSize` (sort `popular\|newest\|price_asc\|price_desc\|rating`) · `GET /products/:id/related` · `GET /products/:id/reviews?page&pageSize` (6a: `{summary: {average, count, distribution: {"1".."5"}}, items: [{id, rating, comment, reviewerName (masked), variantName, edited, createdAt}], page…}`; non-public product → 404 `PRODUCT_NOT_FOUND`) · `GET /stores/:slug` · `GET /stores/:slug/products/:productSlug` |
| Signed in | `GET`/`PATCH /me` · `GET`/`POST /me/addresses` · `PATCH`/`DELETE /me/addresses/:id` · `POST /me/addresses/:id/default` · `POST /seller/store` (opens store + grants seller role) · `GET /cart` · `POST /cart/items` `{variantId, quantity}` (adds to an existing line) · `PATCH /cart/items/:id` `{quantity}` · `DELETE /cart/items/:id` — every cart route returns the whole cart · `POST /checkout/quote` `{addressId, shipping:{<storeId>: regular\|express\|same_day}}` (read-only; missing store = regular) · `POST /checkout` (same body + header `Idempotency-Key: <uuid>`; 201 new, 200 repeat) · `GET /checkout/:id` · `POST /checkout/:id/cancel` (only while `pending_payment`; returns the checkout) · `POST /payments/:id/simulate` `{outcome: success\|failure, eventId: uuid}` (plays the gateway webhook; the same `eventId` again is a no-op 200; returns the checkout) · `GET /orders?page&pageSize` · `GET /orders/:id` (with `history`) · `POST /orders/:id/cancel` (a `paid` order → `refunded`, stock restocked; returns the order) · `POST /reviews` `{orderItemId, rating 1..5, comment? ≤ 1000}` (6a; 201) · `PATCH /reviews/:id` `{rating?, comment?}` · `GET /me/reviews?page&pageSize` (own, hidden included, with `item` ref) · `GET /me/reviews/pending?page&pageSize` (delivered items without a review) |
| Seller (`seller` role) | `GET`/`PATCH /seller/store` · `GET`/`POST /seller/products` · `GET`/`PATCH`/`DELETE(archive) /seller/products/:id` · `POST …/variants` · `PATCH …/variants/:variantId` · `PUT …/variants/:variantId/stock` · `GET /seller/orders?status&page&pageSize` (own store only, newest first) · `GET /seller/orders/:id` (with `history`) · `PATCH /seller/orders/:id/status` `{status, note?}` (paid → processing → shipped → delivered; returns the order) · `GET /seller/stats` (6c: `{ordersByStatus: {<all 7 statuses>: n}, revenue: {last30Days, allTime}, unitsSold: {last30Days, allTime}, topProducts: [{productId, name, status, units, revenue}] (≤ 5, last 30 days), rating: {average, count}, since}`; revenue = `items_subtotal` of paid/processing/shipped/delivered orders, window on `orders.created_at`; `STORE_REQUIRED` 409 without a store) |
| Signed in (wishlist, practice) | `GET /me/wishlist?page&pageSize` · `GET /me/wishlist/ids` · `PUT /me/wishlist/:productId` (idempotent) · `DELETE /me/wishlist/:productId` |
| Admin (`admin` role) | `POST /admin/categories` · `PUT /admin/categories/:id` (both write an audit row since 6d) · `GET /admin/stats` (`{users: {active, suspended}, stores: {active, suspended}, publicProducts, ordersByStatus, gmv: {last30Days, allTime}, since}`) · `GET /admin/users?q&status&page&pageSize` (`q` matches email or name, `%` and `_` literal; items `{id, email, name, status, roles[], createdAt}`) · `PATCH /admin/users/:id/status` `{status: active\|suspended}` (409 `CANNOT_CHANGE_SELF`, `CANNOT_CHANGE_ADMIN`; 404 `USER_NOT_FOUND`; suspend deletes the user's sessions; same status = 200 without an audit row) · `GET /admin/stores?q&status&page&pageSize` (`q` matches name, slug or owner email; items with `owner {id, name, email}` and `activeProductCount`) · `PATCH /admin/stores/:id/status` (404 `STORE_NOT_FOUND`) · `GET /admin/orders?status&page&pageSize` and `GET /admin/orders/:id` (every store, read-only, same Order shape) · `GET /admin/reviews?status&page&pageSize` (items `{id, rating, comment, status, edited, product {id, name}, storeName, reviewer {name, email}, createdAt}`) · `PATCH /admin/reviews/:id/status` `{status: visible\|hidden}` (returns the review; 404 `REVIEW_NOT_FOUND`) · `GET /admin/audit-logs?page&pageSize` (`{id, action, targetType, targetId, details, actor {id, name, email}, createdAt}`, newest first). Default page size 20, max 50 |

Status codes in use: 200 201 204 400 401 403 404 409 413 415 422 429 500 503. Error codes are SCREAMING_SNAKE (`EMAIL_TAKEN`, `SLUG_TAKEN`, `SKU_TAKEN`, `STORE_EXISTS`, `RATE_LIMITED`, `ORIGIN_REJECTED`, …). Cart adds: `CART_ITEM_NOT_FOUND` 404, `VARIANT_NOT_FOUND` 404, `VARIANT_UNAVAILABLE` 409, `OUT_OF_STOCK` 409, `QUANTITY_EXCEEDS_STOCK` 409 (details `available`, `inCart`), `OWN_PRODUCT` 409; quantity outside 1..99 → 422. Addresses: `ADDRESS_NOT_FOUND` 404 (also for someone else's id or a malformed id), `ADDRESS_LIMIT_REACHED` 409; create 201, update/default 200 with the address, delete 204. Checkout: `IDEMPOTENCY_KEY_REQUIRED` 400 (missing or not a UUID), `CART_HAS_ISSUES` 409 (details `items[]` with `itemId, variantId, productName, variantName, quantity, availableQuantity, issue`; issue adds `own_product`), `IDEMPOTENCY_KEY_REUSED` 409, `CART_EMPTY` 422, `SHIPPING_METHOD_NOT_ALLOWED` 422 (details keyed `shipping.<storeId>`), `ADDRESS_NOT_FOUND` 404, `CHECKOUT_NOT_FOUND` 404, `ORDER_NOT_FOUND` 404 (also another buyer's order or a malformed id). Lifecycle (5c): `CHECKOUT_NOT_PENDING` 409, `CHECKOUT_EXPIRED` 409 (the checkout was closed as expired by this request), `PAYMENT_NOT_FOUND` 404, `PAYMENT_EVENT_CONFLICT` 409 (event id already used for another payment or outcome), `INVALID_STATUS_TRANSITION` 409 (details `{from, to}`, or `{from: "pending_payment", checkoutId}` when an unpaid order is cancelled on its own). Seller orders (5d): `ORDER_NOT_FOUND` 404 for another store's order, `STORE_REQUIRED` 409 without a store, `INVALID_STATUS_TRANSITION` 409 for any change the table does not allow the seller, 422 for an unknown status or a note over 500 characters.

Reviews (6a): `ORDER_ITEM_NOT_FOUND` 404 (also another buyer's item or a seller asking), `REVIEW_NOT_ALLOWED` 409 (details `{orderStatus}`; only `delivered` qualifies), `REVIEW_EXISTS` 409, `REVIEW_NOT_FOUND` 404 (another user's or malformed id), `PRODUCT_NOT_FOUND` 404, 422 for rating / comment / ids / paging. Ratings in product summaries, product detail and the store page are real since 6a (`average` rounded to one decimal, `count`); buyer and seller order items carry `review: {id, rating, comment} | null` (`comment` since 6b).

Lifecycle states: checkout `pending_payment → paid | cancelled | expired`; payment `pending → succeeded | failed | cancelled | expired`; on failure, cancel or expiry every order becomes `cancelled`.

Checkout response: `{ id, status, itemsTotal, shippingTotal, grandTotal, expiresAt, paidAt, cancelledAt, createdAt, payment: { id, reference (SIM-…), provider, amount, status, expiresAt, paidAt }, orders: [Order] }`. Order: `{ id, checkoutId, orderNumber (PK-YYYYMMDD-XXXXXX), status, store {id, slug, name (snapshot)}, shippingMethod, shippingFee, itemsSubtotal, total, shippingAddress {…snapshot}, items: [{ id, variantId, productId, productSlug, productName, variantName, sku, unitPrice, quantity, lineTotal, image }], history? , createdAt, updatedAt }`. Quote: `{ address, stores: [{ store, items (cart line shape with itemId), shippingOptions: [{method, fee}], shippingMethod, shippingFee, subtotal, total }], itemsTotal, shippingTotal, grandTotal, hasIssues }`.

Cart response: `{ stores: [{ store, items: [{ id, quantity, variantId, variantName, sku, productId, productSlug, productName, image|null, unitPrice, lineTotal, availableQuantity, issue|null }], subtotal }], itemCount, subtotal, hasIssues }`. `issue` ∈ `unavailable` (archived product / suspended store), `out_of_stock`, `insufficient_stock`; flagged lines are excluded from both subtotals. `itemCount` = total units.

---

## 5. Data model (migrations 000001–000015; 000013 wishlist_items, 000014 audit_logs, 000015 the reviews trigger fires only on rating/comment changes)

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
| `user_addresses` | cascade on user delete (orders will keep a copy); phone CHECK `^\+?[0-9]{8,15}$`, postal code 5 digits; partial unique index `(user_id) WHERE is_default` = at most one default; the service keeps exactly one while addresses exist; max 10 per user |
| `checkouts` | `UNIQUE (user_id, idempotency_key)`; `request_hash` 32 bytes; status pending_payment/paid/cancelled/expired; `grand_total = items_total + shipping_total` CHECK; `expires_at` from the DB clock; partial index on `expires_at` for pending ones |
| `orders` | one per store per checkout (`UNIQUE (checkout_id, store_id)`); `order_number` UNIQUE + format CHECK; 7 statuses; shipping method CHECK; `total = items_subtotal + shipping_fee` CHECK; store name + full address snapshot |
| `order_items` | product/variant name, SKU, unit price snapshots; `line_total = unit_price × quantity` CHECK; quantity 1..99 |
| `order_status_histories` | append-only; `from_status` NULL at creation; `actor_id` NULL = system |
| `payments` | one per checkout; `reference` UNIQUE; provider `simulated`; status pending/succeeded/failed/expired/cancelled |
| `payment_events` | `event_id` UNIQUE (duplicate deliveries are no-ops); created in 5b, used from 5c |
| `reviews` (000012) | `order_item_id` UNIQUE (one per purchase); `product_id`, `store_id`, `user_id` copied from the order; rating 1..5 CHECK; comment ≤ 1000 chars; status visible/hidden; all FKs RESTRICT; partial index `(product_id, created_at DESC) WHERE visible`, index `(user_id, created_at DESC)` |
| `products.rating_count`, `rating_total` (000012) | denormalised counters of VISIBLE reviews; CHECK `rating_count ≥ 0 AND rating_total BETWEEN rating_count AND rating_count × 5` |

All FKs of the checkout tables are `ON DELETE RESTRICT` (history). `inventory_movements.movement_type` also allows `restock` since 000011.

**Invariants worth re-checking after any inventory change**

```sql
-- must return 0: ledger explains every stock level
SELECT count(*) FROM inventory i
LEFT JOIN (SELECT variant_id, SUM(available_delta) s FROM inventory_movements GROUP BY variant_id) m USING (variant_id)
WHERE COALESCE(m.s,0) <> i.available_quantity;
```

Since 5b the same must hold for `reserved_quantity` vs `SUM(reserved_delta)` (the integration helper `ledgerMismatches` checks both). `SUM(reserved_quantity)` equals the units held by `pending_payment` checkouts. Since 5c every ledger type is written: `reservation` (place), `consumption` (paid), `release` (failure, cancel, expiry; reference = checkout) and `restock` (refund; reference = order).

Since 6a also: for every product, `rating_count = COUNT(*)` and `rating_total = SUM(rating)` of its `visible` reviews (integration helper `ratingMismatches`).

Not yet created (planned): `audit_logs` (6d), `shipping_methods`, `shipments`.

---

## 6. Decision log

Decisions made so far and **why** (newest first). Reverse one only deliberately.

| Decision | Why |
|---|---|
| (6b) The review dialog always sends rating **and** comment, starting from the saved review (order items carry its comment) | a dialog that started empty would have overwritten the old comment with "" on a rating-only change |
| (6b) Star picker = visually hidden native radios inside `relative` labels | keyboard (Tab + arrows) and screen readers for free; proven: without `relative` the hidden input sat outside its star |
| (6b) After saving a review, the order page refetches quietly (`getMyOrder` → `replace`) instead of `reload()` | no skeleton flash; the items then carry the new review |
| (6a) One review per **order item** of a delivered order (`order_item_id UNIQUE`); the author may edit, nobody deletes; no review window | developer approved Q1–Q3; buying twice gives two reviews |
| (6a) Product rating = denormalised `rating_count` / `rating_total`, changed in the review transaction with `UPDATE … SET x = x + delta` | listings need the rating on every card without a join; proven: a read-then-write in Go lost 14 of 20 concurrent reviews |
| (6a) No lock on the order when reviewing | `delivered` is final in the transition table, so the checked status cannot change; the unique index handles concurrent duplicates (`ON CONFLICT DO NOTHING` → 409) |
| (6a) Store rating = sum over all of its products, archived included, computed on read | archived products were real purchases; no second counter to keep in sync |
| (6a) Public review list returns the name masked ("Agus W***"), never email or user id | reviews are public, identity is not |
| (6a) Seller product list sorted by `created_at DESC` (Q15) | counter updates (sales, ratings) fire the `updated_at` trigger and used to move a product to the top of the seller's list |
| (6a) Buyer/seller order items carry `review: {id, rating} \| null` (LEFT JOIN in `ListOrderItems`) | the 6b order page can show "Beri ulasan" / "Ubah ulasan" without one request per item |
| (5e) `/checkout` sends exactly the shipping methods of the last quote it displayed, not the raw radio state | the order the buyer confirms is the quote they saw; `POST /checkout` requires a method for every store, the quote defaults missing ones to regular |
| (5e) `Idempotency-Key` = `useState(() => crypto.randomUUID())`, once per visit of /checkout | double click / retry reuses it (server returns the same checkout); a new visit is a new attempt |
| (5e) Changing the address resets the shipping choices to regular | Same Day depends on the destination city; avoids a 422 from a stale choice |
| (5e) The payment page keeps one `eventId` per outcome per visit | clicking again resends the same gateway event (harmless repeat), like a real gateway retry |
| (5e) Which status buttons to show (seller next step, buyer refund) comes from a UI copy of the transition table (`features/orders/status.ts`) | display only; the server still decides, and a 409 reloads the order with a toast |
| (5e) Default history notes stay English in the DB and are translated in the UI (`historyNote`) | API stays English like its other messages; seller-typed notes are shown as written |
| (5d) Seller order routes resolve the store from the session and filter every query by `store_id`; status changes lock the order row (`LockStoreOrder`, the same row as the buyer's refund) and go through `Transition` as `BySeller` | ownership cannot be bypassed by changing an id; "seller starts processing" vs "buyer cancels" serialise (proven: without the lock the loser got 500) |
| (5d) The seller sends the target status (`{status}`), not an action name; any known status is accepted by validation and the table decides (409) | one endpoint, one rule set; 422 only for statuses that do not exist |
| (5d) Sellers see every status of their orders, including unpaid and cancelled ones, with an optional `?status=` filter | nothing hidden; the UI can default to "paid" |
| (5c) Payment simulation, cancel and expiry live in the `checkout` package (`lifecycle.go`), not a separate `payments` module | one payment per checkout (Q4) and every payment outcome is a checkout state change; a separate package would have to export checkout's transaction internals |
| (5c) Every lifecycle change locks the checkout row first; lock order checkout → orders (id) → inventory (variant_id) → products (id) → payment | "pay" vs "cancel" vs "expire" serialise and exactly one wins (proven: without the lock the losers got 500) |
| (5c) Order statuses change only through `orders.Repository.Transition` and the table in `orders/transitions.go`, which also says WHO may make each change (system / buyer / seller) | one place to read and test the lifecycle; a seller can never refund, a buyer can never ship |
| (5c) Payment failure cancels the checkout and releases the stock (Q9 default); expiry sets checkout `expired`, payment `expired`, orders `cancelled` | simple and predictable; `orders` has no `expired` status |
| (5c) An expired checkout touched by pay/cancel is closed as expired and committed, then 409 `CHECKOUT_EXPIRED` | the stock comes back immediately instead of waiting for the sweeper; an error inside the transaction would have rolled the close back, so the error is returned after commit |
| (5c) Payment events: check under the checkout lock, then `INSERT … ON CONFLICT (event_id) DO NOTHING` | a redelivered event is a no-op 200; the same id used for something else is 409 instead of a 500 |
| (5c) Cancelling one unpaid order → 409 with `checkoutId`; only paid orders are cancelled per order (refund) | one payment covers the whole checkout (Q8) |
| (5c) A refund lowers `sold_count`; the payment row stays `succeeded` | refunded units were not sold; refunds are simulated per order and there is no partial-refund state on payments |
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
| UI components = shadcn/ui (style `radix-nova`), brand colour as `--primary` (emerald-600), system font | consistent, accessible primitives; one place to change the look. Geist font from `shadcn init` dropped (no unrequested dependency) |
| Selects use Radix `Select` (developer's choice) over a styled native `<select>` | consistent look; costs: RHF `Controller` instead of `register`, and E2E opens the list and picks an option instead of `selectOption` |
| Account actions live in a `DropdownMenu` (name + email, "Akun saya", seller link on small screens, "Keluar") | developer's choice; cleaner header |
| Logout runs `await navigate('/', { flushSync: true })` before `logout()` | proven: without flushSync the route change (a transition) lost to `setUser(null)` 9 of 10 times and the guard sent the user to `/login` |
| Toast (`Sonner`) and `AlertDialog` are deferred to Phase 5 | developer's choice |

---

## 7. Test inventory

- **Unit** (`make test`, no DB): config, middleware (request ID, recovery, timeout, CORS, auth, same-origin), response, logger, password, token, ratelimit, request/JSON, slug, products validation + search query, auth validation, users model, cart model (issue classification, grouping, exact totals). 98 test functions in total (counted in source), 14 unit packages + `modules/health` integration only; integration-tagged files: `modules/health/integration_test.go`, `router/auth_integration_test.go`, `router/catalog_integration_test.go`, `router/cart_integration_test.go`, `router/addresses_integration_test.go`, `router/checkout_integration_test.go`. Since CP-5a also the `modules/addresses` unit package; since CP-5b `modules/shipping`, `modules/orders`, `modules/checkout`; since CP-5c the order transition table (`orders/transitions_test.go`) and `router/lifecycle_integration_test.go`; since CP-5d `orders/service_seller_test.go` and `router/seller_orders_integration_test.go`; since CP-6a `platform/rating`, `modules/reviews/model_test.go` and `router/reviews_integration_test.go`. 165 test functions in source at CP-6a.
- **Integration** (`make test-integration`, `-tags=integration`, DB `marketplace_test`): health against a real DB; 13 auth flows; 12 catalog flows (ownership, uniqueness, validation, ledger, concurrency, search/sort/filter/pagination, archive); 12 cart flows (login required, empty cart, grouping/totals, not-a-reservation, quantity/stock rules, own product, archived + stock-drop flags, current price, ownership 404s, persistence across sessions, concurrent adds); 10 address flows (see §1e); 12 checkout flows incl. rollback, idempotency and two concurrency tests (see §1f); 9 lifecycle flows incl. idempotent payment events, expiry, refund and two race tests (see §1g); 4 seller order flows incl. ownership, transition rules and the processing-vs-cancel race (see §1h); 5 review flows incl. ownership, the rating invariant and two race tests (see §1j).
- **Browser E2E (since 7a)**: `@playwright/test` suite in `frontend/e2e/` (10 tests in 9 specs, `test.step` per check), run by `make e2e` against `api-e2e` (:8082) on the `marketplace_e2e` DB, which is dropped, migrated and seeded before every full run. `e2e/support/fixtures.ts`: `pageErrors` (auto; any uncaught page error fails the test), `actor()` (second browser, e.g. the seller), `api` (`Api` helper with the page's cookies and a same-origin `Origin`), `customer` (fresh sign-up per test). One worker: specs share demo accounts (admin suspends `penjual4`; cart lowers a demo stock and restores it in `finally`). The old hand-run scripts in the git-ignored `docs/e2e-reference/` are superseded.
- **Frontend unit/component tests (since 7a)**: Vitest 5 + React Testing Library + jsdom, `src/**/*.test.ts(x)`, setup `src/test/setup.ts`, builders in `src/test/fixtures.ts`. 107 tests (see §1o). Services are mocked with `vi.mock('../services/…')`; contexts are provided with their `*Context.Provider` and a fixture value.

---

## 8. Run, verify, restore

```bash
make up                 # build + start postgres and api
make migrate-up         # apply migrations to the dev DB (named `main` on this machine)
make seed-dev           # demo catalog (no-op if categories exist)
make test               # unit tests
make test-integration   # creates/migrates marketplace_test, runs integration tests
cd frontend && npm run build && npm run lint && npm run test   # Git Bash: cd /D/... (upper-case drive, see §11)
make e2e                # fresh marketplace_e2e + api-e2e (:8082) + Playwright (Vite :5191)
cd frontend && npm run dev -- --port 5174    # open http://localhost:5174  (or http://[::1]:5174)
curl http://localhost:8081/health ; curl http://localhost:8081/ready
```

Demo logins (local only): `penjual1@contoh.id` … `penjual4@contoh.id` / `penjual-123`. Admins: `make seed-admin email=… name="…"`.

**API configuration** (all optional except `DATABASE_URL`; defaults in `internal/config/config.go`): `APP_ENV` (development|production), `APP_PORT` (8081), `LOG_LEVEL` (info), `CORS_ALLOWED_ORIGINS` (`http://localhost:5173`), `REQUEST_TIMEOUT` (10s), `SHUTDOWN_TIMEOUT` (15s), `SESSION_TTL` (24h), `ARGON2_MEMORY_KIB` (19456) / `ARGON2_ITERATIONS` (2) / `ARGON2_PARALLELISM` (1), `RATE_LIMIT_LOGIN_FAILURES` (5) / `RATE_LIMIT_LOGIN_WINDOW` (15m) / `RATE_LIMIT_REGISTRATIONS` (10) / `RATE_LIMIT_REGISTER_WINDOW` (1h) (7a; ≥ 1, only `api-e2e` raises them), `DB_MAX_CONNS` (10) / `DB_MIN_CONNS` (2) / `DB_MAX_CONN_LIFETIME` / `DB_MAX_CONN_IDLE_TIME` / `DB_HEALTH_CHECK_PERIOD` / `DB_CONNECT_TIMEOUT`. Compose forwards only `APP_ENV`, `APP_PORT`, `LOG_LEVEL`, `CORS_ALLOWED_ORIGINS`, `SESSION_TTL`, `DATABASE_URL`; the rest need a compose change to be overridden.

**Environment quirks** (this machine): Go is not installed on the host — everything Go/sqlc runs in Docker via the Makefile; GNU make came from scoop; Git Bash path conversion is disabled in the Makefile (`MSYS_NO_PATHCONV=1`); local PostgreSQL 18 owns port 5432 (Compose uses **5433**); another app owns IPv4 `:5173` — use `--port 5174` or `http://[::1]:5173`. Never kill a process by port without checking its command line.

---

## 9. Known gaps, risks, tech debt

| Item | Impact | When |
|---|---|---|
| Rate limiter is in memory | wrong behind >1 API instance | before scaling out |
| No CSRF token (relies on SameSite + Origin) | fine for same-site; weaker if frontend is hosted on another site | Phase 7 review |
| Dev DB contains leftover test accounts/stores (`seller-…@contoh.id`, `admin@contoh.id`, `e2e-…`) | clutter only | cleanup offered, not done |
| Images are URLs only; no upload, no hotlink protection | seller can point at any host | decide with storage plan |
| `sold_count` changes with real payments and refunds since 5c, but the demo values are seeded; ratings are real since 6a but `seed-dev` creates no reviews | "popular" is mostly demo data; almost every product shows 0 ratings | seed reviews later if wanted |
| Updating `sold_count` or the rating counters fires the `products` `updated_at` trigger | `updated_at` means "last change of anything", not "last edit by the seller"; the seller list no longer sorts by it (6a) | accept, or skip the trigger for counter-only updates |
| No index for `sort=rating` (computed `rating_total / rating_count`) | full sort of the matching products | Phase 7 `EXPLAIN` |
| `GET /checkout/:id` does not expire anything itself; the sweeper runs every `CHECKOUT_SWEEP_INTERVAL` (1m) | a checkout can show `pending_payment` up to ~1 minute after `expiresAt` (pay/cancel still refuse it) | handled in the UI: at 0:00 the pay buttons are disabled and "Muat ulang status" is offered |
| The countdown uses the browser clock | a skewed client clock shows the wrong remaining time | server decides anyway (`CHECKOUT_EXPIRED`); could send server time with the response |
| Frontend bundle is one ~840 kB chunk (admin pages included for everyone) | slower first load | route-level `lazy()` in Phase 7; `/admin/*` is the obvious first split |
| The sweeper runs inside the API process | with several instances each runs one; `SKIP LOCKED` prevents double work, but that setup is untested | before scaling out |
| A payment event that arrives after expiry is not stored | no audit trail of late notifications | when a real gateway arrives (also signature checks) |
| Sellers cannot cancel a paid order (e.g. an item turned out to be missing); only the buyer can, before processing | the seller has to ask the buyer | add `paid → cancelled` by seller with restock + refund if needed |
| The optional `?status=` filter is written as `(@status IS NULL OR status = @status)` | measured in 7c (§1q): custom plans use the indexes (0.07 ms seller, 0.05 ms admin) | resolved |
| `ListCheckoutLines` and `ListOrderItems` use LATERAL image lookups | measured in 7c: 0.4 ms for order items | resolved |
| Right after "Keluar", the header badge keeps the old count until the logout request returns (the page navigates first by design) | cosmetic, milliseconds | accept, or clear the cart state optimistically |
| Cart issue flags are recomputed on every `GET /cart`, but the page does not refresh by itself | a stock change by the seller shows up on the next load/mutation | acceptable; checkout re-validates |
| `ListCartLines` joins six tables + a LATERAL image lookup | measured in 7c: 0.06 ms | resolved |
| Test suites exist (7a); CI workflow written (7b) but not yet run on GitHub | until the first green run, CI is unproven | push and watch the Actions tab |
| `npm audit`: 4 moderate advisories via `shadcn` → `ts-morph` → `minimatch` | fixed in 7d: `shadcn` is a devDependency, `npm audit --omit=dev` = 0 (§1r) | resolved |
| No `/auth/me`; no email verification; no password reset | account recovery impossible | backlog |
| Mobile layout checked only down to ~500 px | 390 px unverified (headless limit) | Phase 3 polish |
| Money display converts strings to JS numbers | fine for display; never sum client-side for charging | rule for Phase 4–5 |
| Rate limiter keys on `c.ClientIP()`; in dev every request comes from the Docker gateway `172.18.0.1` | dev: all users share one budget (by design; the Vite proxy sends no `X-Forwarded-For`). Production: `TRUSTED_PROXIES` added in 7d (§1r) | set it to nginx in the 7e prod compose |
| Expired sessions were deleted only when the same user logged in again | fixed in 7d: background cleanup (§1r) | resolved |
| Cookie `Secure` flag path had no test | fixed in 7d: `TestSessionCookie_SecureOnlyInProduction` (§1r) | resolved; real HTTPS still unverified (7e) |
| Seller and admin order lists, admin user/store/audit lists had no `page` upper bound (negative OFFSET → 500) | fixed in 7d: `MaxPage = 10_000` (§1r) | resolved |
| `ListPublicProducts` at scale | fixed in 7c (window count removed, 2.2 s → 3 ms); price sort 0.2 s and rating sort 85 ms accepted (§1q) | `min_price` denormalisation if it ever matters |
| `AddVariant` checked the 20-variant cap before inserting | fixed in 7d: lock the product, then count (26 → 20 under concurrency, §1r) | resolved |
| Search uses the `simple` config without `unaccent` | "kemeja" ≠ "kémeja"; no stemming | acceptable for now |
| `.env.example` lists the main variables plus commented tuning (7d added `TRUSTED_PROXIES`, `SESSION_CLEANUP_BATCH`); the checkout/shipping ones are still missing | people don't discover optional tuning | document-only; defaults are safe |
| `frontend/README.md` is still the Vite template | misleading | replace with a short pointer to docs |
| `golang:1.27-alpine` / `postgres:18` tags float | builds can change under you | pin in Phase 7 |
| CI not yet proven | `ci.yml` written in 7b, no GitHub run yet | see §1p |
| JS bundle was one 838 kB chunk | fixed in 7c: entry 190 kB + vendor chunks + route chunks (§1q) | resolved |

---

## 10. Next steps (proposed)

The dependency-checked version, with the exact next task, is in [SESSION_HANDOFF.md](../SESSION_HANDOFF.md); the Phase 5 questions are in [PLAN.md](../PLAN.md).

1. Developer reviews and commits 7d (§1r has the commands).
2. Developer pushes and checks the Actions tab (all jobs green; one deliberately broken test shows red, then revert; the first run also proves migration 16 on CI's up → down → up). `docs/e2e-reference/` (git-ignored, local only) is superseded and can be deleted.
3. 7e — API docs & deploy prep (PLAN.md "Phase 7" §7.1): include `TRUSTED_PROXIES` = nginx, the SPA CSP and HSTS in nginx (SECURITY_REVIEW §4).

---

## 11. Failed approaches and lessons (so they are not repeated)

| What happened | Cause | What was done / lesson |
|---|---|---|
| `docker run … -w /src` failed with `C:/Program Files/Git/src` | Git Bash rewrites arguments that look like POSIX paths | `export MSYS_NO_PATHCONV := 1` in the Makefile; use it in any ad-hoc `docker run` from Git Bash |
| `go`, `docker`, `make` "not found" in the agent shell | not on PATH (Go not installed at all) | Go/sqlc/migrate run in containers; Docker CLI path is in CLAUDE.md; `make` via scoop |
| `seed-admin` test created an admin in the **dev** DB | a `sed` rewrote `…/marketplace_dev?` but the dev DB is called `main` | never assume the dev DB name; check `.env` keys, not values; test accounts were left behind |
| A process on port 5173 was killed and turned out to be the developer's Laravel app | killed by port without reading its command line | always inspect `CommandLine` first; use port 5190 for agent-run Vite |
| Running the E2E scripts from the agent's Git Bash: Edge exited at launch (`Target page, context or browser has been closed`) | the Bash tool's sandbox | run `node <script>.mjs` from the PowerShell tool instead |
| `npx shadcn add dialog alert-dialog --yes` wrote nothing | it stopped at an overwrite prompt for the existing `button.tsx` (stdin closed = abort) | pipe `n` answers (`printf 'n\n' | npx shadcn add …`), then fix the `from "cn"` imports |
| After changing `POSTGRES_PASSWORD` in `.env`, the API crash-looped with `password authentication failed` | the postgres image applies `POSTGRES_PASSWORD` only when it initialises an **empty** volume | also run `ALTER USER` inside the container (§1d shows how without printing the value), then restart `api` |
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
| `shadcn init` wrote `import { cn } from "cn"` (an unrelated npm package) and added the Geist font | CLI behaviour with this setup | removed both; fix the import after every `shadcn add` (§3) |
| Logout from `/account` landed on `/login` again after moving "Keluar" into the Radix menu; `await navigate('/')` alone did not fix it (1 of 2 runs still failed) | React Router commits navigations in a transition; the urgent `setUser(null)` could render first, so the guard on the old page redirected | `navigate('/', { flushSync: true })`; measured 9/10 failures before, 0/10 after. For timing bugs, repeat the scenario several times instead of trusting one green run |
| A debug run timed out on sign-up | the register limit (10/hour per IP, shared Docker gateway IP) was used up by repeated E2E/debug runs | debug with existing demo logins; restart the `api` container before a full E2E run. Since 7a the E2E stack has its own API with raised `RATE_LIMIT_*` |
| (7a) Every Vitest file failed with "Vitest failed to find the current suite" (the first run had worked) | Git Bash `cd /d/...` gives the process a lower-case drive (`d:\`); Vitest then loads its runtime twice. `process.chdir` and an explicit `root` in the config did not help (the CLI is already loaded) | run from PowerShell/cmd or Git Bash `cd /D/...`; `vitest.config.ts` now stops at once with that hint |
| (7a) A Vitest hook timed out after 10 s | `beforeEach(() => mock.mockReset())` returns the mock (a function), which Vitest calls as a teardown; it returned a never-settling promise | write hooks with braces: `beforeEach(() => { … })` |
| (7a) Two E2E steps failed with "strict mode violation" on a fresh DB but had passed on the old dev DB | `getByText('1 ulasan')` also matched the rating bars' screen-reader text; the product page has hearts on related-product cards too | `exact: true`; scope to the product `article`. A fresh DB exposes selectors that only worked because of leftover data |
| (7a) The admin spec read an empty category name | `innerText` of the first table row while the skeleton rows were still shown (`innerText` does not wait) | wait for something only a real row has (its "Ubah" button) before reading text |

---

## How to update this file

At the end of each phase (or any risky change):

1. Run the verify commands in §8; only then add a **Checkpoint** row in §1 with the commit hash and what was *actually* run.
2. Update the snapshot header, §2 status, and §4/§5 if routes or tables changed.
3. Add new decisions to §6 (with the reason); move fixed items out of §9.
4. Keep CLAUDE.md for rules and this file for state — don't duplicate long text between them; link instead.
