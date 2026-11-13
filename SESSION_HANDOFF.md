# SESSION_HANDOFF.md

Updated 2026-11-13, at the end of **Phase 4 (cart)**, for a fresh Claude Code session.

**Read order:** this file → [CLAUDE.md](CLAUDE.md) (rules, loaded automatically) → [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md) (checkpoints, decision log, known issues, failed approaches) → [PLAN.md](PLAN.md) (Phase 4 as built + Phase 5 open questions).

**Evidence labels:** **[verified]** = command run in the session that produced this file and its output read · **[verified earlier]** = observed earlier in the same long session, not re-run · **[not verified]** = from reading code or conversation only.

---

## 1. Current project understanding

**What it is.** A learning project: a multi-vendor marketplace (Tokopedia-style UX, original branding) with a Go backend, React frontend and PostgreSQL. The developer is **learning Go and backend engineering**: keep code explicit and explain important flows (request → handler → service → repository → SQL; why each constraint, lock and transaction exists). Talk to the developer in **Indonesian**; repo docs are English.

**Stack.** Go 1.27.3 + Gin + pgx/pgxpool + sqlc + golang-migrate · PostgreSQL 18 · React 19 + TypeScript + Vite + Tailwind 4 + react-router-dom + react-hook-form + zod · Docker Compose (postgres + api; frontend on the host with `npm run dev`). Modular monolith: `router → middleware → handler → service → repository → sqlc → pgxpool`.

**Working agreement (CLAUDE.md §16):** plan and wait for approval before substantial work; one phase at a time and stop at the end for review; **never commit, push or deploy** (propose commit messages); never print or commit `.env`; report failures honestly; don't say "done" without running the checks.

**Where we are.**

| Phase | State |
|---|---|
| 0 Foundation, 0.5 UI prototype, 1 Identity | committed, reviewed |
| 2 Catalog | committed (`2838782`, `d0c2a54`; docs `41679b9`) |
| 3 Marketplace UI | mostly delivered by Phase 2; polish open |
| **4 Cart** | **implemented and verified, uncommitted, awaiting review** |
| 5 Checkout & orders | next — needs answers to PLAN.md Q1–Q8 and a written plan |

---

## 2. Completed work (Phase 4 in this session)

| Area | Evidence |
|---|---|
| Migration `000009` (carts, cart_items) | [verified] up → down → up on dev DB; applied to `marketplace_test`; CHECK 1..99 and UNIQUE(cart, variant) tried by hand |
| `modules/cart` + 4 routes | [verified] `make test` 14 packages ok; `make test-integration` 15 packages ok; 12 cart integration tests listed and passing |
| Cart-row lock | [verified] concurrency test fails without it (quantity 1 instead of 10), passes 5/5 with it |
| Frontend `/cart`, badge, add-to-cart, `CartProvider` | [verified] `npm run build`, `npm run lint` clean; browser E2E `cart-flow.mjs` 16/16; auth 15/15 and catalog 17/17 re-run |
| Invariants on dev DB | [verified] ledger mismatches 0; `SUM(reserved_quantity)` 0; demo stock changed by E2E restored |
| Docs | PROJECT_STATE (§1 CP-4 + commit commands, §1b evidence, §4/§5/§6/§7/§9/§10/§11), PLAN, PHASES, CLAUDE.md §2/§14, this file |

Working tree: Phase 4 + docs **uncommitted** on top of `41679b9`. Commit commands: PROJECT_STATE §1. `docs/e2e-reference/` is git-ignored by the developer (local-only copies of the three E2E scripts, incl. `cart-flow.mjs`).

Test functions in source: 98 [verified].

---

## 3. Important technical decisions (most likely to matter next)

Full log: PROJECT_STATE §6.

1. **Money** = `NUMERIC(12,2)` ↔ `decimal.Decimal` ↔ JSON string `"349000.00"`; never float; totals only server-side.
2. **Variant price is authoritative**; carts show **current** prices; Phase 5 must snapshot name, SKU and unit price into order items.
3. **Stock change** = `SELECT … FOR UPDATE` on `inventory` → update → ledger row, one tx (`modules/inventory`). Invariant `SUM(available_delta) = available_quantity`. Proven necessary.
4. **Cart is not a reservation.** Nothing writes `reserved_quantity` yet; Phase 5 checkout is where `available → reserved` happens (with ledger rows), under row locks, with conditional updates.
5. **Cart mutations** lock the cart row first (`EnsureCart` = `INSERT … ON CONFLICT DO UPDATE`, or `LockCartByUser` = `SELECT … FOR UPDATE`). Proven necessary.
6. **Ownership by construction**: seller queries filter `store_id` from the session; cart queries filter by the caller's cart/user; foreign IDs → 404. Orders must follow the same pattern (buyer by `user_id`, seller by `store_id`).
7. **Visibility rule** (public catalog and cart add): product active ∧ store active ∧ has variants. In the cart, lines that stop qualifying stay with an `issue` and are excluded from subtotals.
8. **Sessions**: opaque HttpOnly SameSite=Lax cookie `sid`, SHA-256 stored, rotated on login; **CSRF** = SameSite + `SameOrigin` (Origin vs Host) + JSON-only bodies. Vite proxy keeps `changeOrigin:false` for `/api`.
9. **Layering**: services never import Gin; they return plain errors / `validation.Errors`; handlers map to HTTP; repositories translate constraint names; multi-step writes use `database.WithTx` in the service; repositories take `queries.DBTX`.
10. **Tests**: integration tests only against `*_test` DB (`TRUNCATE users, categories CASCADE` also clears carts); `make test-integration` migrates it first.

---

## 4. Files and modules relevant to the next task (Phase 5 — Checkout & orders)

The next task is **planning**, not coding: answer PLAN.md Q1–Q8 with the developer, then write the Phase 5 plan. Files a plan (and later the code) will build on:

| Path | Why |
|---|---|
| `backend/internal/modules/cart/` (`service.go`, `repository.go`, `model.go`) | checkout reads the cart, re-validates every line under lock, then clears purchased lines; `buildCart`/`classify` define "buyable" |
| `backend/db/queries/cart.sql` | `ListCartLines` (current price, stock, status per line) |
| `backend/internal/modules/inventory/inventory.go` | the lock → update → ledger pattern to extend with reservation / release / consumption (`movement_type` already allows them) |
| `backend/db/migrations/000008…` | `inventory.reserved_quantity`, `inventory_movements` (reference columns for `('checkout', id)` / `('order', id)`) |
| `backend/internal/modules/products/` | module pattern, money formatting, error mapping |
| `backend/internal/database/tx.go` | `WithTx` |
| `backend/internal/router/router.go` | new `signedIn` and `seller` routes (`/checkout`, `/orders`, `/seller/orders/:id/status`, `/payments/:id/simulate`) |
| `backend/internal/router/cart_integration_test.go`, `catalog_integration_test.go` | fixtures (`newTestApp`, `newSeller`, `createProduct`, `newCartFixture`) to reuse; concurrency-test pattern |
| `frontend/src/pages/CartPage.tsx` | disabled "Lanjut ke checkout" button |
| `frontend/src/features/catalog/ProductDetailView.tsx` | disabled "Beli sekarang" |
| `frontend/src/features/cart/CartProvider.tsx`, `services/cart.ts`, `types/cart.ts` | cart state to clear/refresh after checkout |
| CLAUDE.md §7 | non-negotiable checkout rules |

Dependencies: CP-4 committed; developer answers Q1–Q8; approval of the written plan; Docker up (`make up`), dev DB on host port 5433.

---

## 5. Known limitations and unresolved issues

Full tables: PROJECT_STATE §9 (debt/risks) and §11 (failed approaches).

**Unresolved**
- **Rate limiter shares one IP bucket in dev**: every request reaches the API from the Docker gateway (`172.18.0.1`, [verified earlier]); register is limited to 10/hour per IP, so sign-up-heavy E2E runs can hit 429 (restart the `api` container to reset). Needs a trusted-proxy decision before deploy.
- **Dev DB exposure**: Compose publishes Postgres on `0.0.0.0:5433` and the API on `0.0.0.0:8081`; the developer's `.env` DB password is 4 characters (length only checked). Not changed — developer decision.
- Rate limiter in memory (single instance); expired sessions only cleaned when the same user logs in; `Secure` cookie path untested; no `EXPLAIN` on catalog/cart queries; ratings 0 and `sold_count` demo-only; no admin category UI; mobile < 500 px unverified.
- Dev DB holds many throw-away accounts from E2E runs (`e2e-…`, `seller-…`, `buyer-…@contoh.id`, `admin@contoh.id`, `demo@contoh.id`).
- **Uncommitted, unreviewed frontend deps** in `frontend/package.json` and lock: `shadcn`, `cn`, `class-variance-authority`, `tw-animate-css`, and a Tailwind bump 4.3.3 → 4.3.5. Excluded from the commit procedure. Developer must justify or revert with `git checkout -- frontend/package.json frontend/package-lock.json`.
- **Stray root artifacts** (created 15:51 on 2026-11-13, cause unknown): `package.json`, `package-lock.json`, and `node_modules/` at repo root (about 860 untracked entries). Never run `git add .` or `git add -A` without a path. Delete after confirmation; root `.gitignore` lacks `/node_modules/`.
- Cosmetic: right after "Keluar" the badge keeps the old count until the logout request returns.

**Technical debt**: no frontend unit tests; E2E scripts are hand-run and git-ignored; no CI; floating `golang:1.27-alpine` / `postgres:18` tags; `frontend/README.md` is the Vite template.

**Not verified**: more than one API instance, reverse proxy, HTTPS/`APP_ENV=production`; load on catalog/cart queries.

---

## 6. Recommended next steps

1. **Developer reviews CP-4 and commits it** (PROJECT_STATE §1 has the commands; run `git reset -q` first if anything is staged).
2. Decide the two open infrastructure items: trusted proxy / rate-limit key, and binding Compose ports to `127.0.0.1` + a longer DB password.
3. **Exact next task — Phase 5 planning:** go through PLAN.md Q1–Q8 with the developer (defaults proposed there), then write the Phase 5 plan in PLAN.md: migrations (`user_addresses`, `checkouts`, `orders`, `order_items`, `order_status_histories`, `payments`, `payment_events`), the checkout transaction step by step, reservation/consume/release ledger types, endpoints, tests (overselling concurrency, idempotency, rollback, ownership), UI. **Stop for approval before writing code.**

**First 10 minutes in a new session**

```bash
git status --short ; git log --oneline | head -4     # Phase 4 uncommitted (or committed by now) on top of 41679b9
make ps                                               # postgres healthy, api up; otherwise: make up
make test && make test-integration                    # all ok
cd frontend && npm run build && npm run lint          # clean
```

Then tell the developer (in Indonesian) what you found and ask how to proceed.

---

## 7. Environment notes (this machine)

- Windows 11; Git Bash and PowerShell. **Go is not installed on the host** — Go, sqlc and migrate run in Docker via the Makefile (`MSYS_NO_PATHCONV=1` is set there). `make` 4.4.1 from scoop.
- Docker CLI: `C:\Users\Rava\AppData\Local\Programs\DockerDesktop\resources\bin\docker.exe` (may not be on PATH in agent shells).
- Ports: Compose Postgres **5433**, API **8081**; local PostgreSQL 18 owns **5432**; another project of the developer's owns IPv4 **5173**. Agent-run Vite uses **5190**; the developer uses `npm run dev -- --port 5174`. Before stopping any process by port, read its `CommandLine`.
- Dev DB is named **`main`** (from `.env`); test DB `marketplace_test`. Demo sellers `penjual1..4@contoh.id` / `penjual-123` (local only).
