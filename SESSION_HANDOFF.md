# SESSION_HANDOFF.md

Updated 2026-11-16, after **Phase 6e (admin UI)** was built (uncommitted; 6a–6d committed), for a fresh Claude Code session.

**Read order:** this file → [CLAUDE.md](CLAUDE.md) (rules, loaded automatically) → [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md) (checkpoints, decision log, known issues, failed approaches) → [PLAN.md](PLAN.md) (Phase 4 as built + Phase 5 open questions).

**Evidence labels:** **[verified]** = command run in the session that produced this file and its output read · **[verified earlier]** = observed earlier in the same long session, not re-run · **[not verified]** = from reading code or conversation only.

---

## 1. Current project understanding

**What it is.** A learning project: a multi-vendor marketplace (Tokopedia-style UX, original branding) with a Go backend, React frontend and PostgreSQL. The developer is **learning Go and backend engineering**: keep code explicit and explain important flows (request → handler → service → repository → SQL; why each constraint, lock and transaction exists). Talk to the developer in **Indonesian**; repo docs are English.

**Stack.** Go 1.27.3 + Gin + pgx/pgxpool + sqlc + golang-migrate · PostgreSQL 18 · React 19 + TypeScript + Vite + Tailwind 4 + shadcn/ui (Radix) + react-router-dom + react-hook-form + zod · Docker Compose (postgres + api; frontend on the host with `npm run dev`). Modular monolith: `router → middleware → handler → service → repository → sqlc → pgxpool`.

**Working agreement (CLAUDE.md §16):** plan and wait for approval before substantial work; one phase at a time and stop at the end for review; **never commit, push or deploy** (propose commit messages); never print or commit `.env`; report failures honestly; don't say "done" without running the checks.

**Where we are.**

| Phase | State |
|---|---|
| 0 Foundation, 0.5 UI prototype, 1 Identity | committed, reviewed |
| 2 Catalog | committed (`2838782`, `d0c2a54`; docs `41679b9`) |
| 3 Marketplace UI | mostly delivered by Phase 2; polish open |
| 4 Cart | committed (`7e57829`, `f312b0b`; docs `0d63dfc`) |
| UI migration to shadcn/ui | committed (`db15fef`; docs `83b34a7`) |
| Dev housekeeping (loopback ports, Vite target, `.gitignore`) | committed (`ac4e163`); Phase 5 plan `d4dde07` |
| 5 Checkout & orders | committed: 5a–5d as before, 5e `33e046e`, docs `6f10765`; homepage carousel `932065a` |
| **6 Reviews & dashboards** | plan approved 2026-11-15 (PLAN.md Q1–Q15 defaults); 6a committed (`491953f`, `f8ab642`); 6b committed (`6f01036`, `b5fe739`); wishlist practice feature committed (`3d722a3`, `12bbd7f`); 6c committed (`0677906`, `6e18967`, docs `9716ed9`); 6d committed (`4ed91de`, `e00c95f`, docs `07bdc43`); **6e admin UI done and verified, uncommitted** (PROJECT_STATE §1n); that closes Phase 6 |

---

## 2. Completed work (since Phase 4)

| Area | Evidence |
|---|---|
| Phase 4 cart (backend + frontend) | committed; evidence in PROJECT_STATE §1b |
| shadcn/ui migration: `components.json`, `@/` alias, theme tokens, `src/components/ui/*`, Radix Select for category/sort, account `DropdownMenu`, logout with `navigate('/', { flushSync: true })` | committed `db15fef`; [verified earlier] build + lint clean, E2E auth 15/15, catalog 17/17, cart 16/16; logout race 9/10 wrong before the fix, 10/10 right after |
| Compose ports on `127.0.0.1`; Vite `API_TARGET` = `http://127.0.0.1:8081`; root `.gitignore` `/node_modules/` | [verified] containers recreated (volume kept, 30 products), `/ready` 200 direct and via Vite :5190, login POST via Vite 200, API not answering on a non-loopback adapter; **uncommitted**, commit command in PROJECT_STATE §1d |

`docs/e2e-reference/` is git-ignored by the developer (local-only copies of the three E2E scripts); they were updated for the shadcn UI.

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

## 4. Files and modules relevant to the next task (Phase 6c — seller dashboard)

> Update 2026-11-15: Phase 5 is complete and 6a/6b are done; the wishlist practice feature is committed (`3d722a3`, `12bbd7f`; PLAN.md §6.4b, migration `000013`, so the admin audit-log migration is now `000014`). The next task is **6c**; the table below is the original Phase 5 file list, still useful as a map of the modules 6c reads (`orders`, `reviews`, `products`).

Files the work builds on:

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
| `frontend/src/pages/CartPage.tsx` | disabled "Lanjut ke checkout" button (shadcn Card/Button) |
| `frontend/src/components/ui/` | shadcn components; Phase 5 adds `sonner` (toast) and `alert-dialog` (developer decision). After `npx shadcn add`, fix `from "cn"` → `from "@/lib/utils"` |
| `frontend/src/features/catalog/ProductDetailView.tsx` | disabled "Beli sekarang" |
| `frontend/src/features/cart/CartProvider.tsx`, `services/cart.ts`, `types/cart.ts` | cart state to clear/refresh after checkout |
| CLAUDE.md §7 | non-negotiable checkout rules |

Dependencies: developer answers Q1–Q8; approval of the written plan; Docker up (`make up`), dev DB on host port 5433.

---

## 5. Known limitations and unresolved issues

Full tables: PROJECT_STATE §9 (debt/risks) and §11 (failed approaches).

**Unresolved**
- **Rate limiter shares one IP bucket in dev**: every request reaches the API from the Docker gateway (`172.18.0.1`, [verified earlier]); register is limited to 10/hour per IP, so sign-up-heavy E2E runs can hit 429 (restart the `api` container to reset). Needs a trusted-proxy decision before deploy.
- Rate limiter in memory (single instance); expired sessions only cleaned when the same user logs in; `Secure` cookie path untested; no `EXPLAIN` on catalog/cart queries; ratings 0 and `sold_count` demo-only; no admin category UI; mobile < 500 px unverified.
- Dev DB holds many throw-away accounts from E2E runs (`e2e-…`, `seller-…`, `buyer-…@contoh.id`, `admin@contoh.id`, `demo@contoh.id`).
- JS bundle 672 kB (gzip 208 kB) after Radix; Vite warns about >500 kB chunks → route-level code splitting in Phase 7.
- Cosmetic: right after "Keluar" the badge keeps the old count until the logout request returns.

**Technical debt**: no frontend unit tests; E2E scripts are hand-run and git-ignored; no CI; floating `golang:1.27-alpine` / `postgres:18` tags; `frontend/README.md` is the Vite template.

**Not verified**: more than one API instance, reverse proxy, HTTPS/`APP_ENV=production`; load on catalog/cart queries.

---

## 6. Recommended next steps

1. **Developer reviews and commits 6e** (PROJECT_STATE §1n has the commit commands).
2. **Next — Phase 7 (hardening)**: write a plan (PHASES.md row 7 and PROJECT_STATE §9/§10 list the candidates) and wait for approval. Local admin for the UI: `make seed-admin email=... name=...`; the E2E admin `e2e-admin@contoh.id` exists in the dev DB (password kept outside the repo).
3. *(Done in 6e, kept for reference)* 6e admin UI (PLAN.md §6.1): `RequireAdmin` guard (UX only), `/admin` (stats tiles), `/admin/users`, `/admin/stores` (search, status filter, suspend/reactivate with AlertDialog), `/admin/categories` (list + create/edit on the existing API), `/admin/orders` (+ detail, reuse `OrderDetailLayout`), `/admin/reviews` (hide/unhide), `/admin/audit-logs`; header link for admins; types in `src/types/admin.ts` from PROJECT_STATE §4; loading states with `components/shared/Skeletons`. To log in as admin locally: `make seed-admin email=... name=...` (the existing `admin@contoh.id` password is not in the repo). Then stop for review.

**First 10 minutes in a new session**

```bash
git status --short ; git log --oneline | head -4     # housekeeping uncommitted (or committed by now) on top of 83b34a7
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
