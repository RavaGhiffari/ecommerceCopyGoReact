# PLAN.md — Phase 4: Cart (PROPOSED, NOT YET APPROVED)

This replaces the Phase 0–2 plan, which is complete (see [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md) §1, §6). Nothing below is implemented. Rules live in [CLAUDE.md](CLAUDE.md).

## Preconditions (blockers)

1. **CP-2 committed.** The catalog work is verified but only exists in the working tree (66 uncommitted paths). Commit commands: `docs/PROJECT_STATE.md` §1.
2. **Developer approval of this plan**, and answers to the open questions below.

## Goal

A signed-in customer keeps a persistent cart across sessions. Adding validates stock **but does not reserve it** (CLAUDE.md §7); the cart shows **current** prices; items are grouped by store so Phase 5 can create one order per store.

## Open questions (need an answer before coding)

| # | Question | Proposed default |
|---|---|---|
| Q1 | Cart for anonymous visitors (merge on login) or login required? | **Login required** — simplest, matches "each buyer has an independent cart" |
| Q2 | Maximum quantity per line? | `min(stock, 99)` |
| Q3 | May a seller add their own store's products? | **No** → 409 `OWN_PRODUCT` |
| Q4 | Lines whose stock dropped below the cart quantity: error or warning? | **Warning flag in `GET /cart`**, hard error only on add/update |
| Q5 | Header badge: separate lightweight endpoint or part of `GET /cart`? | Part of `GET /cart` (`itemCount`), refetched after mutations |

## Backend

**Migration `000009_create_cart_tables`** (up + down; verify up → down → up on dev and test DBs):

- `carts(id uuid pk, user_id uuid not null unique references users on delete cascade, created_at, updated_at)` + `set_updated_at` trigger.
- `cart_items(id uuid pk, cart_id uuid not null references carts on delete cascade, variant_id uuid not null references product_variants on delete restrict, quantity integer not null check (quantity > 0 and quantity <= 99), created_at, updated_at, unique (cart_id, variant_id))`.

**Queries** (`db/queries/cart.sql`, then `make sqlc`; remember `::uuid` casts and `COALESCE` on LATERAL columns):

- `GetOrCreateCart` (INSERT … ON CONFLICT (user_id) DO UPDATE … RETURNING).
- `UpsertCartItem` (INSERT … ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = …) so concurrent adds of the same variant yield one row.
- `ListCartLines`: join cart_items → variants → products → stores (+ first image, + `inventory.available_quantity`); filters nothing, so the service can flag lines that became unavailable (archived product, suspended store, out of stock).
- `UpdateCartItemQuantity` / `DeleteCartItem`: **both filter by the caller's cart** (`WHERE id = $1 AND cart_id = $2`) → another user's item id is a 404.

**Module `modules/cart/`** (`handler.go`, `service.go`, `repository.go`, `dto.go`, `model.go`), following the products module pattern:

- `GET /api/v1/cart` → `{ stores: [{ store, items: [{ id, variant, product, quantity, unitPrice, lineTotal, available, issue? }], subtotal }], itemCount, subtotal }`. Totals are computed server-side with `decimal` and sent as strings.
- `POST /api/v1/cart/items` `{ variantId, quantity }` → validates: variant exists and is **publicly visible** (active product, active store), quantity ≥ 1, resulting quantity ≤ `min(stock, 99)`, not the caller's own store (Q3). Adds to an existing line.
- `PATCH /api/v1/cart/items/:id` `{ quantity }`; `DELETE /api/v1/cart/items/:id` (204).
- Routes go in the `signedIn` group of `router/router.go`. Sellers are also customers, so no role gate.
- New error codes: `CART_ITEM_NOT_FOUND`, `OUT_OF_STOCK`, `QUANTITY_EXCEEDS_STOCK`, `VARIANT_UNAVAILABLE`, `OWN_PRODUCT`.

## Frontend

- `types/cart.ts` (mirror of the DTOs), `services/cart.ts` (`apiRequest`), `features/cart/` (hook + provider for the item count, `CartLine`, `StoreGroup`, `CartSummary`).
- `/cart` page behind `RequireAuth`: store groups, quantity steppers, remove, subtotal, loading / empty / error states; checkout button stays **disabled** until Phase 5.
- Enable "Tambah ke keranjang" in `features/catalog/ProductDetailView.tsx` (send the selected variant + quantity; anonymous → `/login?next=…`); enable the cart button + badge in `components/layout/Header.tsx`.
- Show server field/code errors via the existing `applyServerError` / `ApiError` pattern.

## Tests (must exist before calling it done)

- Unit: quantity/price/line-total calculations, DTO shape.
- Integration (`marketplace_test`): add → increments; cap by stock; archived / suspended-store variants rejected; cannot exceed 99; own-store product rejected; update and delete; **user B cannot update or delete user A's item (404)**; price shown follows a seller's later price change; groups and subtotal per store; **concurrent adds of the same variant produce one row with the summed quantity**; stock reduced after adding → line flagged, `GET` still 200.
- Browser: extend `docs/e2e-reference/` style script (register → add two products from two stores → cart groups → change quantity → remove).

## Acceptance criteria

1. up → down → up on dev and test DBs; `make test` and `make test-integration` pass; `npm run build` and `npm run lint` pass.
2. A cart persists across logout/login.
3. Cart totals equal the sum of current variant prices × quantities, computed server-side.
4. No cart route can read or change another user's cart.
5. Docs updated: `docs/PROJECT_STATE.md` (new checkpoint CP-4), CLAUDE.md §2, `SESSION_HANDOFF.md`.

## Out of scope

Reservation of stock, checkout, orders, shipping, payment, anonymous carts (unless Q1 changes), coupons.

## Risks

- Price/stock shown in the cart can be stale by checkout time → Phase 5 must re-validate everything inside its transaction.
- `ListCartLines` joins several tables; check the plan with `EXPLAIN` on seeded data (Phase 7 will repeat this at scale).
- The rate limiter shares one IP bucket in dev (known issue, `docs/PROJECT_STATE.md` §9) — irrelevant to cart routes, which are not rate limited.
