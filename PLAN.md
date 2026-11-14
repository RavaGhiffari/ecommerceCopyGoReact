# PLAN.md

Rules live in [CLAUDE.md](CLAUDE.md); verified state in [docs/PROJECT_STATE.md](docs/PROJECT_STATE.md).

---

## Phase 4 — Cart: AS BUILT (committed `7e57829`, `f312b0b`)

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

## Phase 5 — Checkout & orders: PLAN (approved 2026-11-14 with the proposed answers to Q1–Q12)

Progress: 5a, 5b and 5c committed. **5d done and verified, awaiting review** (evidence: PROJECT_STATE §1h). Next: 5e.

### 5.0 Decisions

Q1–Q8 use the proposed defaults until the developer changes them. Q9–Q12 came up while writing the plan.

| # | Question | Proposed answer |
|---|---|---|
| Q1 | Addresses | Address book (`user_addresses`), at most one default per user and at most 10 per user. The chosen address is **copied** onto every order, so editing or deleting an address never changes an order |
| Q2 | Shipping | Flat fee per **store order** by method, from config: Regular 15 000, Express 30 000, Same Day 45 000. Same Day only if the address city equals the store `location` (case- and space-insensitive; see Q11) |
| Q3 | What is checked out | The **whole cart**. Any line with an `issue` blocks checkout (409) until the buyer fixes it |
| Q4 | Payment granularity | One simulated payment per **checkout**, covering all its store orders |
| Q5 | Expiry | 30 minutes (`CHECKOUT_TTL`). An expired checkout is cancelled and its reservations released. A sweep goroutine in the API does this every minute, and the expiry is also checked under lock whenever someone touches the checkout |
| Q6 | Duplicate submissions | Required `Idempotency-Key` header (UUID), with `UNIQUE (user_id, idempotency_key)`. Same key and same body → the original checkout (200). Same key and a different body → 409 `IDEMPOTENCY_KEY_REUSED` (see Q12) |
| Q7 | Order number | `PK-YYYYMMDD-XXXXXX` (6 random Crockford base-32 characters), `UNIQUE`. On a collision, retry with a new number |
| Q8 | Cancellation | While `pending_payment`, the buyer cancels the **whole checkout** (one payment covers all its orders). After `paid`, the buyer may cancel **one order** until the seller marks it `processing`; the order becomes `refunded` (simulated) and its stock goes back to available |
| Q9 | Simulated payment **failure** | The payment is marked `failed`, the checkout is cancelled and the stock released (simple and predictable). Alternative: keep the checkout open and allow a retry until it expires |
| Q10 | "Beli sekarang" on the product page | Add to cart, then go to `/checkout`. Because of Q3 the whole cart is checked out, so the buyer sees everything that is in it. Alternative: keep the button disabled |
| Q11 | Same Day "same city" | `stores.location` and the address city are both free text, so the match is a normalised string compare ("Jakarta Selatan" ≠ "Jakarta"). Good enough for a simulation; a city table is out of scope |
| Q12 | Idempotency fingerprint | Store a SHA-256 of the canonical request body with the checkout, to detect a reused key with a different body. Alternative: ignore the body and always return the original |

### 5.1 Slices

Each slice ends with its tests green and a short review; the developer commits.

| Slice | Content |
|---|---|
| **5a Addresses** ✅ | Migration `000010_create_user_addresses`, module `addresses`, routes under `/me/addresses`, address management on `/account`. As built: create 201 / update 200 / set default 200 return the address, delete 204; PATCH has no `isDefault`; deleting the default promotes the newest; every write takes `SELECT … FROM users … FOR NO KEY UPDATE` |
| **5b Checkout core** ✅ | Migration `000011_create_checkout_tables` (checkouts, orders, order_items, order_status_histories, payments, payment_events, new ledger type); module `shipping` (pure functions); module `checkout` (quote and place); inventory `Reserve`; buyer `GET /orders`, `GET /orders/:id`, `GET /checkout/:id`. As built: `CHECKOUT_STOCK_CHANGED` was folded into `CART_HAS_ISSUES` (place re-validates under lock and reports every bad line the same way; the client reacts identically); a quote defaults a store without a choice to `regular`, placing requires a choice per store; payment rows are created by the checkout package (the `payments` package arrives with simulate in 5c); `CHECKOUT_SWEEP_INTERVAL` is deferred to 5c with the sweeper; nullable `timestamptz` maps to `*time.Time` (sqlc override) |
| **5c Payment & lifecycle** ✅ | `POST /payments/:id/simulate`; inventory `Consume` / `Release` / `Restock`; `sold_count`; expiry sweeper; `POST /checkout/:id/cancel`; `POST /orders/:id/cancel`; status-transition table. As built: no `payments` package; simulate, cancel, expiry and the sweeper live in `checkout/lifecycle.go` (one payment per checkout, so every payment outcome is a checkout change); the transition table also names the allowed actor (system / buyer / seller); new 409 codes `CHECKOUT_EXPIRED` (the request itself closed the expired checkout, committed) and `PAYMENT_EVENT_CONFLICT`; cancelling one unpaid order → 409 `INVALID_STATUS_TRANSITION` with `checkoutId`; a refund also lowers `sold_count`; no migration was needed |
| **5d Seller orders** ✅ | `GET /seller/orders`, `GET /seller/orders/:id`, `PATCH /seller/orders/:id/status` (processing → shipped → delivered). As built: the list also takes `pageSize` and shows every status unless `?status=` is given; an empty note gets a default history note; any known status passes validation and the transition table answers (409); no migration |
| **5e Frontend** | `/checkout`, `/checkout/:id` (payment page), `/orders`, `/orders/:id`, seller order pages, Sonner toast and AlertDialog, enable the cart and product-page buttons |

### 5.2 Data model

All money is `NUMERIC(14,2)`, because totals can exceed one variant price. All timestamps are `TIMESTAMPTZ`. Every FK from a history table is `ON DELETE RESTRICT`.

**`user_addresses`** (000010) has these columns:

- `id`.
- `user_id` → users, `ON DELETE CASCADE`. An address is not history; orders keep their own copy.
- `label` (e.g. "Rumah"), `recipient_name`.
- `phone`, with a CHECK for digits and `+`, 8–20 characters.
- `address_line`, `city`, `province`.
- `postal_code`, with a CHECK for 5 digits.
- `is_default`, plus timestamps.

A partial unique index `ON (user_id) WHERE is_default` lets the database guarantee at most one default per user.

**000011:**

| Table | Key columns and constraints |
|---|---|
| `checkouts` | `user_id`, `idempotency_key UUID`, `request_hash BYTEA`, `status` (`pending_payment`, `paid`, `cancelled`, `expired`), `items_total`, `shipping_total`, `grand_total` (CHECK `grand_total = items_total + shipping_total`), `expires_at`, `paid_at`, `cancelled_at`. `UNIQUE (user_id, idempotency_key)`. Partial index `(expires_at) WHERE status = 'pending_payment'` for the sweeper |
| `orders` | `checkout_id`, `store_id`, `buyer_id`, `order_number UNIQUE`, `status` (the 7 statuses from CLAUDE.md §7), `shipping_method` (`regular`, `express`, `same_day`), `shipping_fee`, `items_subtotal`, `total` (CHECK sum), **address snapshot** (`ship_recipient_name`, `ship_phone`, `ship_address_line`, `ship_city`, `ship_province`, `ship_postal_code`), `store_name` snapshot. Indexes `(buyer_id, created_at DESC)`, `(store_id, status, created_at DESC)`, `(checkout_id)` |
| `order_items` | `order_id`, `variant_id`, `product_id` (for links and Phase 6 reviews), **snapshots** `product_name`, `variant_name`, `sku`, `unit_price`, `quantity` (CHECK 1..99), `line_total` (CHECK `= unit_price * quantity`) |
| `order_status_histories` | `order_id`, `from_status` (NULL at creation), `to_status`, `actor_id` (NULL = system, i.e. expiry or payment), `note`, `created_at`. Append-only |
| `payments` | `checkout_id UNIQUE`, `reference` (`SIM-…`, unique), `provider` (`simulated`), `amount`, `status` (`pending`, `succeeded`, `failed`, `expired`, `cancelled`), `expires_at`, `paid_at` |
| `payment_events` | `payment_id`, `event_id` **UNIQUE** (makes duplicate deliveries a no-op), `event_type`, `payload JSONB`, `created_at` |

`inventory_movements.movement_type` gains `restock`, for stock that was sold and comes back after a refund. The ledger invariant becomes, per variant: `SUM(available_delta) = available_quantity` **and** `SUM(reserved_delta) = reserved_quantity`. Tests check both.

| Event | `available` | `reserved` | Ledger type | Reference |
|---|---|---|---|---|
| Place checkout | −q | +q | `reservation` | (`checkout`, id) |
| Payment succeeded | 0 | −q | `consumption` | (`checkout`, id) |
| Cancel, failure or expiry before payment | +q | −q | `release` | (`checkout`, id) |
| Refund of a paid order | +q | 0 | `restock` | (`order`, id) |

### 5.3 Endpoints

| Method + path | Who | Notes |
|---|---|---|
| `GET/POST /me/addresses`, `PATCH/DELETE /me/addresses/:id`, `POST /me/addresses/:id/default` | signed in | Filtered by `user_id`; a foreign id → 404 |
| `POST /checkout/quote` | signed in | Body `{addressId, shipping: {<storeId>: "regular"…}}`. Read-only. Returns one group per store with its lines, allowed methods and fees, the totals, and any `issues`. The UI uses it to show prices before placing the order |
| `POST /checkout` | signed in | Same body plus the `Idempotency-Key` header. 201 with the checkout, its orders and its payment; 200 for a repeat |
| `GET /checkout/:id`, `POST /checkout/:id/cancel` | owner | Cancel only while `pending_payment` |
| `POST /payments/:id/simulate` | owner of the checkout | Body `{outcome: "success"\|"failure", eventId: UUID}`. Acts like a gateway webhook; a repeated `eventId` returns the current state |
| `GET /orders?page=`, `GET /orders/:id`, `POST /orders/:id/cancel` | buyer | Filtered by `buyer_id` |
| `GET /seller/orders?status=&page=`, `GET /seller/orders/:id`, `PATCH /seller/orders/:id/status` | seller | `store_id` comes from the session; body `{status, note?}` |

New error codes:

- 400: `IDEMPOTENCY_KEY_REQUIRED`.
- 409: `CART_HAS_ISSUES` and `CHECKOUT_STOCK_CHANGED` (both with per-line details), `IDEMPOTENCY_KEY_REUSED`, `CHECKOUT_NOT_PENDING`, `INVALID_STATUS_TRANSITION`, `CHECKOUT_EXPIRED`.
- 422: `CART_EMPTY`, `SHIPPING_METHOD_NOT_ALLOWED`.

### 5.4 `POST /checkout` step by step (one transaction, `database.WithTx`)

1. Handler: strict JSON, `Idempotency-Key` must be a UUID, field validation (422).
2. **Lock the cart row** (`LockCartByUser … FOR UPDATE`). This serialises the user's checkouts with each other and with cart edits; a line added during checkout waits.
3. **Idempotency**: look up `(user_id, key)`. If found with the same `request_hash`, return it with no new writes. If the hash differs, return 409.
4. Load the address **filtered by `user_id`** (a foreign address → 404).
5. Read the cart lines. An empty cart → 422.
6. **Lock every inventory row** for those variants with a single `SELECT … FOR UPDATE … ORDER BY variant_id`. Because every checkout locks in the same order, two checkouts that share variants can never wait on each other in a circle (no deadlock).
7. Re-validate each line under the lock: product active, store active, `available_quantity >= quantity`, and not the buyer's own store. Any problem → 409 with the affected lines and a **rollback**: nothing is written and the cart is untouched.
8. Take prices **from the database** (the variant's current price) and shipping fees per store from config, check each shipping method, and compute totals with `decimal`. Nothing is priced from the request.
9. Insert, in order:
   - one `checkouts` row;
   - one `orders` row per store. The order number uses `INSERT … ON CONFLICT (order_number) DO NOTHING RETURNING id` and retries when no row comes back; a plain unique-violation error would abort the whole transaction;
   - the `order_items` snapshots;
   - the `order_status_histories` rows (NULL → `pending_payment`);
   - one `payments` row (`pending`, same `expires_at`).
10. **Reserve** stock: `UPDATE inventory SET available_quantity = available_quantity - q, reserved_quantity = reserved_quantity + q WHERE variant_id = $1 AND available_quantity >= q`, plus a ledger row. The `WHERE` clause and the CHECK constraint are a second guard after the lock.
11. Delete the purchased cart lines (all of them, per Q3).
12. Commit and return 201. An error in any step rolls back everything.

**Payment success (5c).** Lock the checkout row. If it is still `pending_payment` and not past `expires_at`:

- the payment becomes `succeeded` and the checkout `paid`;
- every order becomes `paid`, with a history row;
- `Consume` lowers reserved stock by q;
- `products.sold_count` goes up by q;
- the event is stored.

Expiry, cancellation and payment all **lock the checkout row first**, so "pay" and "expire" at the same moment cannot both win; the second one sees the new status and stops. The sweeper uses `FOR UPDATE SKIP LOCKED LIMIT 50`, so it never waits on a checkout that someone is paying.

### 5.5 Order status transitions

One table in `orders/transitions.go`, used by every code path.

| From | To | By |
|---|---|---|
| `pending_payment` | `paid` | payment success |
| `pending_payment` | `cancelled` | buyer cancel (whole checkout), payment failure, expiry |
| `paid` | `processing` | seller |
| `paid` | `refunded` | buyer cancel before processing (simulated refund, `restock`) |
| `processing` | `shipped` | seller |
| `shipped` | `delivered` | seller (simulation; there is no courier) |

`delivered`, `cancelled` and `refunded` are final. Any other change → 409 `INVALID_STATUS_TRANSITION`. Each change locks the order row, checks the table, updates the order and writes a history row, all in one transaction.

### 5.6 Code layout (backend)

- `modules/addresses`: handler, service, repository, dto, model.
- `modules/shipping`: pure functions only, `Methods(sameCity) []Method` and `Fee(method)`. No database, so it is easy to unit-test.
- `modules/checkout`: quote, place and cancel. It orchestrates cart, inventory, orders and payments inside one transaction.
- `modules/orders`: buyer and seller reads, `Transition(ctx, tx, orderID, to, actor)`, and the order number generator.
- `modules/payments`: simulate, event recording, and the expiry sweeper (`Run(ctx)`, started from `cmd/api/main.go` and stopped on shutdown).
- `modules/inventory`: add `Reserve`, `Release`, `Consume` and `Restock`. They follow the same lock → update → ledger pattern and all take the caller's transaction.
- Config: `CHECKOUT_TTL` (30m), `CHECKOUT_SWEEP_INTERVAL` (1m), `SHIPPING_FEE_REGULAR`, `SHIPPING_FEE_EXPRESS`, `SHIPPING_FEE_SAME_DAY`.

### 5.7 Tests

Integration tests run against `marketplace_test` unless noted.

**Unit:** shipping fees and the Same Day rule; the transition table (every allowed and forbidden pair); order number format; totals with decimals.

**Addresses:** CRUD; one default per user (also proved by the partial unique index); the limit of 10; a foreign id → 404.

**Checkout:**
- Happy path with two stores: 2 orders, snapshots, reservation ledger rows, cart emptied, totals = sum of lines + fees.
- Repeat with the same key → the same checkout and **no second reservation**.
- Same key with a different body → 409.
- Stock lowered by the seller between quote and place → 409 and **nothing written**. Rollback proof: counts of checkouts, orders and ledger rows unchanged, and the cart intact.
- A line with an archived product or a suspended store → 409.
- Buying from one's own store → 409.
- Another user's address → 404.

**Concurrency (mandatory):**
- 10 buyers race for 3 units of one variant → exactly 3 succeed, `available` ends at 0 and is never negative, and the ledger invariant holds.
- Two checkouts that share two variants, in opposite cart order → no deadlock.

**Payment:**
- Success consumes reservations and raises `sold_count`.
- The same `eventId` twice → one effect.
- Failure releases stock.
- Paying after `expires_at` → 409 and the stock is released.
- Payment racing the sweeper → exactly one outcome.

**Lifecycle and ownership:**
- A buyer cancelling a pending checkout releases stock; a refund after payment restocks.
- Seller transitions work.
- Seller A cannot read or change seller B's order (404).
- A buyer cannot call seller routes (403).
- Buyer B cannot read buyer A's order (404).

**Frontend E2E:** a hand-run script `checkout-flow.mjs`, like the others. It covers adding an address, checking out a two-store cart, a successful payment, the order list, the seller marking an order shipped and the buyer seeing it, plus a cancel flow.

### 5.8 Frontend (5e)

- New shadcn components: `sonner` (toast), `alert-dialog` (cancel confirmation), `radio-group` (shipping method, address choice), `dialog` (address form).
- `/checkout`:
  - an address picker, with "add new";
  - one card per store with its lines and a shipping-method radio group;
  - a summary from `POST /checkout/quote`, so prices shown come only from the server;
  - "Buat pesanan", which sends an `Idempotency-Key` generated **once per visit** (`crypto.randomUUID()` in a ref, so a double click or a retry reuses it).
- `/checkout/:id`: payment page with a countdown to `expiresAt`, buttons "Simulasikan bayar berhasil" and "Simulasikan bayar gagal", and cancel (with an AlertDialog).
- `/orders` and `/orders/:id`: list and detail with a status timeline, plus a cancel or refund button where allowed.
- `/seller/orders` and `/seller/orders/:id`: filter by status, next-status buttons.
- Cart page: "Lanjut ke checkout" enabled, but disabled while the cart has issues. "Beli sekarang" follows Q10.
- Types in `src/types/checkout.ts` and `src/types/orders.ts` mirror the Go DTOs.

### 5.9 Risks and things deliberately left out

- The sweeper runs inside the API process, which is fine for one instance. With several instances, `SKIP LOCKED` already prevents double-processing, but this is untested.
- No real refunds, invoices, courier tracking or partial shipments; one shipment per order.
- Prices can change between quote and place. Place always uses the current price, and the UI shows the totals it gets back. An optional "price changed" warning can compare the quote and placed totals on the client.
- No rate limit on checkout is planned, since the session and the cart lock already limit abuse. Revisit in Phase 7 together with the trusted-proxy setting.

Preconditions: this plan is approved, and the housekeeping commit (PROJECT_STATE §1d) is done.
