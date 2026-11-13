# PHASES.md — roadmap

Rewritten at the 2026-11-13 handoff. The first version carried hour estimates and design details that were later superseded; those were removed. **Authoritative sources:** rules → [CLAUDE.md](CLAUDE.md); verified state and checkpoints → [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md); the next task → [PLAN.md](PLAN.md) and [SESSION_HANDOFF.md](SESSION_HANDOFF.md).

Status legend: ✅ implemented **and verified** · 🟡 implemented, awaiting developer review · ⬜ not started. "Verified" means the checks listed in `docs/PROJECT_STATE.md` §1 were run.

| Phase | Scope | Status |
|---|---|---|
| 0 Foundation | `backend/` layout, config, pgxpool, JSON logging, middleware, graceful shutdown, `/health` `/ready`, golang-migrate, sqlc, Makefile, Compose, status page | ✅ |
| 0.5 UI prototype | Catalog pages on typed mock data (UI-first, developer's choice) | ✅ (superseded: mocks removed in Phase 2) |
| 1 Identity & security | users/roles/sessions; register, login, logout, `/me`; Argon2id; cookie sessions; CSRF layers; rate limits; `seed-admin` | ✅ |
| 2 Catalog | categories (2 levels), stores, products, variants, image URLs, inventory + ledger, public listing/search, seller dashboard, `seed-dev` | 🟡 (not yet committed) |
| 3 Marketplace UI on real APIs | homepage, nav, cards, detail, store page, search | 🟡 mostly delivered by Phase 2 wiring; remaining: polish, 390 px mobile check, banner/"recommended" refinement |
| 4 Cart & inventory | carts, cart items, stock validation on add (no reservation), grouped-by-store view, cart UI | ⬜ **next** — plan in [PLAN.md](PLAN.md) |
| 5 Checkout & orders | addresses, shipping simulation, one-transaction multi-store checkout (checkout → per-store orders), `available → reserved`, snapshots, idempotency key, order lifecycle, payment simulation, cancel/expiry release | ⬜ |
| 6 Reviews & dashboards | reviews (purchased only, one per purchase), rating aggregates, buyer/seller/admin dashboards, category admin UI | ⬜ |
| 7 Hardening | Playwright + Vitest suites, security review, `EXPLAIN` / index review, OpenAPI, CI, pinned images, session cleanup, deploy prep | ⬜ |

## Rules for moving between phases

1. A phase starts only after explicit developer approval; stop at the end of each phase for review (CLAUDE.md §14, §16).
2. Before calling a phase done: run the verify commands in `docs/PROJECT_STATE.md` §8, record a checkpoint row, and update the docs in the same change.
3. The developer commits; Claude proposes messages only.

## Deviations from the original plans (decided during implementation)

- Router **Gin**, migrations **golang-migrate**, auth **cookie sessions** (not Chi / Goose / JWT) — CLAUDE.md §15.
- Health endpoints live at `/health` and `/ready`, outside `/api/v1`.
- There is **no `/auth/me`**: use `GET /api/v1/me`.
- Opening a store is `POST /api/v1/seller/store` (singular) and is open to any signed-in user; it grants the `seller` role.
- Product detail is addressed by store: `GET /api/v1/stores/:slug/products/:productSlug` (not `/products/:id`).
- Money travels as decimal strings (`"349000.00"`).
- Rating sort was dropped until reviews exist (Phase 6).
- Mock data was a Phase 0.5 device only; it no longer exists.
