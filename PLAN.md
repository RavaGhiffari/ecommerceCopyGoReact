# PLAN.md

Rules live in [CLAUDE.md](CLAUDE.md); verified state in [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md).

---

## Phase 4 — Cart: AS BUILT (implemented and verified, awaiting developer review)

Approved defaults: Q1 login required · Q2 `min(stock, 99)` per line · Q3 sellers cannot buy their own products · Q4 stock problems are flags in `GET /cart`, errors only on add/update · Q5 badge count from `GET /cart`.

Delivered (details and evidence: PROJECT_STATE §1b, §4, §6):

- Migration `000009_create_cart_tables` (`carts` one per user; `cart_items` CHECK 1..99, `UNIQUE(cart_id, variant_id)`).
- `db/queries/cart.sql` + `internal/modules/cart` (model / repository / service / dto / handler); routes in the `signedIn` group: `GET /cart`, `POST /cart/items`, `PATCH /cart/items/:id`, `DELETE /cart/items/:id`.
- Every mutation: one transaction that locks the cart row first; adding never touches `inventory`.
- Frontend: `types/cart.ts`, `services/cart.ts`, `features/cart/` (`CartProvider`, `useCart`), `/cart` page, header badge, working "Tambah ke keranjang".
- Tests: 3 unit (model), 12 integration, browser E2E 16 checks.

Deviations from the proposal:

- `DELETE /cart/items/:id` returns **200 with the updated cart**, not 204 (one round trip for the UI).
- Exceeding 99 per line is a 422 validation error; exceeding stock is a 409 `QUANTITY_EXCEEDS_STOCK` with `available` / `inCart` in `details`.
- No separate cart-count endpoint (Q5).

---

## Phase 5 — Checkout & orders: NOT PLANNED YET

Nothing below is approved or implemented. Write a full plan (migrations, endpoints, transaction steps, tests, UI) only after the developer answers these questions. Proposed defaults are marked.

| # | Question | Proposed default |
|---|---|---|
| Q1 | Addresses: saved address book with a default, or typed at each checkout? | Address book (`user_addresses`) with one default; checkout snapshots the chosen address onto each order |
| Q2 | Shipping simulation rule | Flat fee per **store order** by method: Regular / Express / Same Day (Same Day only when buyer and store are in the same city); fees in config |
| Q3 | What is checked out: the whole cart or selected lines? | All lines of the cart; any line with an `issue` blocks checkout until fixed |
| Q4 | Payment granularity | One simulated payment per **checkout** (covers all its store orders) |
| Q5 | Reservation expiry for unpaid checkouts | 30 minutes; expired checkouts are cancelled and their reservations released (lazily on read + a periodic sweep in the API process) |
| Q6 | Duplicate submissions | Client sends an `Idempotency-Key` header (UUID); `UNIQUE (user_id, idempotency_key)` on `checkouts`; a repeat returns the original checkout |
| Q7 | Order number format | `PK-YYYYMMDD-XXXXXX` (random base-32), unique |
| Q8 | Cancellation | Buyer may cancel while `pending_payment`; after `paid`, only before the seller marks `processing` (simulated refund) |

Fixed by CLAUDE.md §7 regardless of the answers: one PostgreSQL transaction for checkout; lock inventory rows, conditional `available_quantity >= qty` updates, `available → reserved` with ledger rows; per-store orders under one parent checkout; price/name/SKU snapshots; status history with validated transitions; payment events processed idempotently; mandatory overselling concurrency test.

Preconditions: CP-4 committed; decision on the rate-limiter/trusted-proxy issue (PROJECT_STATE §9) before any deploy.
