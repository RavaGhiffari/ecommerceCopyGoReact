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

Progress: 5a–5d committed. **5e done and verified, awaiting review** (evidence: PROJECT_STATE §1i). After 5e Phase 5 is complete.

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
| **5e Frontend** ✅ | `/checkout`, `/checkout/:id` (payment page), `/orders`, `/orders/:id`, seller order pages, Sonner toast and AlertDialog, enable the cart and product-page buttons. As built: place sends the methods of the displayed quote; changing the address resets shipping to regular; one payment `eventId` per outcome per visit; buyer order list has no status filter (the API has none); default history notes translated in the UI; `sonner` without `next-themes` |

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

---

## Phase 6 — Reviews & dashboards: PLAN (approved 2026-11-15 with the proposed answers to Q1–Q15)

Phase 5 is committed (`33e046e`, `6f10765`). Progress: 6a committed (`491953f`, `f8ab642`), 6b committed (`6f01036`, `b5fe739`). 6c committed (`0677906`, `6e18967`, docs `9716ed9`). 6d committed (`4ed91de`, `e00c95f`, docs `07bdc43`). **6e done and verified, awaiting review** (evidence: PROJECT_STATE §1n).

### 6.0 Decisions

| # | Question | Proposed answer |
|---|---|---|
| Q1 | What makes a purchase "eligible" | One **order item** in an order whose status is `delivered`. `reviews.order_item_id UNIQUE` = one review per eligible purchase. Buying the same product in two orders gives two reviews. Refunded or cancelled orders are never eligible |
| Q2 | Editing and deleting | The buyer may **edit** rating and comment (no time limit); no delete by the buyer. Alternative: one-shot, no edits |
| Q3 | Review window | None: a delivered item can be reviewed at any time. Alternative: 30 days after delivery |
| Q4 | Rating aggregate | Denormalised `products.rating_count` + `products.rating_total` (integer sum), updated **in the same transaction** as the review write with an atomic `UPDATE … SET rating_total = rating_total + $delta`. Average = total / count, computed when reading. Store rating = `SUM(total) / SUM(count)` over its products, computed on read. Invariant tested like the stock ledger: `rating_total = SUM(rating)` of visible reviews. Alternative: `AVG()` on read from `reviews` (no denormalisation, but a join in every listing) |
| Q5 | Sort by rating | Bring back `sort=rating`: average DESC, then count DESC, then newest. Products without reviews sort last |
| Q6 | Reviewer name shown publicly | First name plus a masked rest, e.g. "Rava G***". The API never returns the email or user id of a reviewer |
| Q7 | Comment | Optional, at most 1000 characters, trimmed, plain text (React escapes it; no HTML) |
| Q8 | Review moderation | Admin can **hide** / unhide a review (`status visible|hidden`); hidden reviews leave the aggregate and the public list, the author still sees theirs with a "hidden" note. Every hide/unhide writes an audit log row. Seller replies → backlog |
| Q9 | Admin scope in this phase | Users (list, search by email, suspend / reactivate), stores (list, suspend / reactivate), categories UI (list, create, edit on the existing API), platform orders (read-only list + detail), basic stats, review moderation, audit log viewer. **Product moderation** is covered by suspending the store; a per-product admin block → backlog. Alternative: add a `blocked` product status now |
| Q10 | Suspension rules | An admin cannot suspend themselves or another admin. Suspending a user ends their sessions (already enforced in the session lookup). Suspending a user does **not** suspend their store (separate action). Existing orders of a suspended store keep their lifecycle (buyer refund, seller fulfilment) so no buyer is stuck. Reactivating restores visibility |
| Q11 | Audit log | New table `audit_logs (actor_id, action, target_type, target_id, details JSONB, created_at)`, append-only, written **in the same transaction** as the admin action. Actions in this phase: `user.suspend`, `user.reactivate`, `store.suspend`, `store.reactivate`, `category.create`, `category.update`, `review.hide`, `review.unhide` |
| Q12 | Seller dashboard | `GET /seller/stats`: order counts per status, revenue (sum of `items_subtotal` of orders in paid/processing/shipped/delivered) for the last 30 days and all time, units sold, top 5 products by units in the last 30 days, rating of the store. Stat tiles + a small table; **no chart library** (no new dependency) |
| Q13 | Admin stats | `GET /admin/stats`: users (active/suspended), stores (active/suspended), products (active), orders per status, GMV last 30 days (same revenue rule as Q12, over all stores) |
| Q14 | Buyer side | `/account` gets a "Menunggu ulasan" section (delivered items without a review); `/orders/:id` gets a "Beri ulasan" / "Ubah ulasan" button per item; product detail shows the rating summary (average, count, 1–5 distribution) and a paginated review list |
| Q15 | `updated_at` side effect | Counter-only updates (`sold_count`, rating columns) currently bump `products.updated_at` and reorder the seller's product list (PROJECT_STATE §9). Proposed: sort the seller product list by `created_at DESC` instead. Alternative: make the trigger skip counter-only changes |

### 6.1 Slices

Each slice ends with its tests green and a review; the developer commits.

| Slice | Content |
|---|---|
| **6a Reviews API** ✅ | Migration `000012_create_reviews` (reviews table + `products.rating_count/rating_total`); module `reviews`; `POST /reviews`, `PATCH /reviews/:id`, `GET /products/:id/reviews?page=` (public, with summary + distribution), `GET /me/reviews/pending` (eligible items without a review), `GET /me/reviews`; real ratings in product list/detail and store page; `sort=rating`; Q15. As built: buyer order items carry `review: {id, rating} \| null` (one query, for the 6b buttons); the average is rounded to one decimal in `platform/rating`; `REVIEW_NOT_ALLOWED` details carry `orderStatus`; no lock on the order (delivered is final); the store rating includes archived products |
| **6b Reviews UI** ✅ | Star rating component, review dialog (RHF + zod) from order detail and the account page, reviews section on product detail, stars on product cards, rating sort option. As built: the order item's `review` also carries `comment` (backend, additive) so "Ubah ulasan" starts from the saved text instead of overwriting it with an empty one; `/account` shows "Menunggu ulasan" **and** "Ulasan saya" (with the "hidden by admin" note, Q8); stars on cards were already there (`RatingBadge`) and now show real data; the rating next to the product title links to `#ulasan`; the star picker is native radios (keyboard works) |
| **6c Seller dashboard** ✅ | `GET /seller/stats` + dashboard tiles on `/seller`. As built: lives in module `orders` (`stats.go`; it reuses `storeOf`, so the store comes from the session and no wiring changed); no migration and no stats table, four aggregate queries on read; revenue = `items_subtotal` (no shipping) of `RevenueStatuses` = paid/processing/shipped/delivered, passed from Go as `text[]`; the 30-day window is on `orders.created_at` (a checkout expires in 30 minutes, so created ≈ paid); units sold are shown for 30 days **and** all time; top products use the product's current name and status (archived still count); the four reads are not one transaction (a payment between them could skew one response; acceptable for a dashboard). UI: 4 tiles (revenue, units, "Perlu diproses" = paid count linking to `/seller/orders?status=paid`, rating), orders per status linking to the filtered list, best-sellers table |
| **6d Admin API** ✅ | Migration `000014_create_audit_logs` (`000013` is taken by wishlist); module `admin`: `GET /admin/users?q=&status=&page=`, `PATCH /admin/users/:id/status`, `GET /admin/stores?q=&status=&page=`, `PATCH /admin/stores/:id/status`, `GET /admin/orders?status=&page=`, `GET /admin/orders/:id`, `GET /admin/stats`, `PATCH /admin/reviews/:id/status`, `GET /admin/audit-logs?page=`; category create/update start writing audit rows. As built: module `admin` holds users, stores, stats and the audit-log list; platform orders live in `orders/admin.go` and review moderation in `reviews/admin.go`, next to the code that owns that data; module `audit` = `Write(ctx, tx, Entry)` taking the caller's transaction (like `inventory`). Added `GET /admin/reviews?status=` (6e needs a moderation list) and migration **000015** (the reviews `updated_at` trigger now fires only on rating/comment changes; without it hide → unhide marked a review as edited, proved by a failing test). Suspending a user also **deletes their sessions**, so a later reactivation does not revive them. A change to the current status is a 200 no-op without an audit row. Rules are checked under the row lock; refused requests write nothing. `seed-dev` gained demo orders (one buyer, 8 checkouts in every status, reviews, spread over 45 days) |
| **6e Admin UI** ✅ | `RequireAdmin` guard (UX only), `/admin` (stats), `/admin/users`, `/admin/stores`, `/admin/categories`, `/admin/orders` (+ detail), `/admin/reviews`, `/admin/audit-logs`; header link for admins; AlertDialog confirmations for suspend. As built: the admin routes are children of one `admin` route (`RequireAdmin` + `AdminLayout` with a section bar); search / status / page live in the URL (`useAdminListParams`); a PATCH answer replaces the row in place (`usePatchedRows`), no refetch; only destructive actions (suspend, hide) ask first, reactivate / show run directly; the own row and other admins get no action (the API refuses them anyway); the order detail reuses `OrderDetailLayout` with a read-only note; the category dialog reuses the public `GET /categories`, offers only top-level parents and locks the parent of a category that has children; the audit log renders a one-line summary from each action's `details`. E2E `admin-flow.mjs` 16/16 |

### 6.2 Data model

**`reviews`** (000012):

- `id`, `order_item_id UNIQUE` → order_items `ON DELETE RESTRICT`, `product_id` → products, `store_id` → stores, `user_id` → users (all RESTRICT, history).
- `rating SMALLINT` CHECK 1..5, `comment TEXT` (CHECK `char_length <= 1000`, may be empty).
- `status` CHECK `visible|hidden`, `created_at`, `updated_at` (trigger `set_updated_at`).
- Indexes: `(product_id, created_at DESC) WHERE status = 'visible'` (public list), `(user_id, created_at DESC)` (my reviews).
- `products` gains `rating_count INTEGER NOT NULL DEFAULT 0 CHECK (>= 0)` and `rating_total INTEGER NOT NULL DEFAULT 0 CHECK (rating_total BETWEEN rating_count AND rating_count * 5)`.

`product_id`, `store_id` and `user_id` are copied from the order (not trusted from the request) so the review rows can be filtered without joining order_items → orders.

**`audit_logs`** (000014): `id`, `actor_id` → users (RESTRICT), `action VARCHAR(64)`, `target_type VARCHAR(32)`, `target_id UUID`, `details JSONB NOT NULL DEFAULT '{}'`, `created_at`. Index `(created_at DESC)` and `(target_type, target_id, created_at DESC)`. No update or delete query is generated for it.

### 6.3 `POST /reviews` step by step (one transaction)

1. Handler: strict JSON `{orderItemId, rating, comment?}`; validation → 422.
2. Service: load the order item **joined to its order filtered by `buyer_id` = caller** (no lock needed: `delivered` is final in the transition table, so the status cannot change after the check). Not found or another buyer's → 404 `ORDER_ITEM_NOT_FOUND`.
3. Order status must be `delivered`, else 409 `REVIEW_NOT_ALLOWED`.
4. `INSERT … ON CONFLICT (order_item_id) DO NOTHING RETURNING id`; no row → 409 `REVIEW_EXISTS` (a plain unique violation would abort the transaction).
5. `UPDATE products SET rating_count = rating_count + 1, rating_total = rating_total + $rating WHERE id = $product`. The row lock of this `UPDATE` serialises concurrent reviews of the same product, so the counters never lose an update.
6. Commit; 201 with the review.

Edit: lock the review row (`WHERE id = $1 AND user_id = $caller FOR UPDATE`, foreign → 404), apply `rating_total + (new − old)` only while the review is visible. Hide / unhide: lock the review, move it out of / into the aggregate, write the audit row — same transaction.

### 6.4 Tests

- **Unit**: review validation (rating range, comment length/trim), name masking, stats date window, admin request validation.
- **Integration (reviews)**: delivered item → 201; not delivered / refunded → 409; another buyer's item → 404; second review → 409; edit changes the aggregate by the delta; hidden review leaves the aggregate and the public list; public list never exposes email/user id; `sort=rating` order; **20 buyers reviewing one product concurrently → `rating_count = 20`, `rating_total = SUM(rating)`** (and proof that it fails if the counter update is replaced by read-modify-write); the same item reviewed twice concurrently → one 201 + one 409.
- **Integration (admin)**: every admin route → 403 for customer and seller, 401 anonymous; suspend user → their next request is 401; cannot suspend self / another admin → 409; suspend store → its products leave the catalog and the cart flags them, existing orders still progress; every admin write leaves exactly one audit row in the same tx (rollback test: a failing action writes no audit row); stats match hand-computed fixtures.
- **Integration (seller stats)**: numbers match fixtures; another store's orders never counted.
- **E2E** (hand-run, git-ignored like the others): `review-flow.mjs` (deliver an order with the demo seller, review it, see the rating on the product page, edit it) and `admin-flow.mjs` (needs an admin from `make seed-admin`: suspend a store, see the products disappear, reactivate, create a category, read the audit log). Register-heavy scripts are not looped (rate limit).

### 6.4b Wishlist (practice feature, outside the phase plan; committed `3d722a3`, `12bbd7f`)

Built by the developer as the DEV_GUIDE §13 level-3 exercise, between 6b and 6c.

- Migration `000013_create_wishlist_items`: `user_id` (CASCADE), `product_id` (RESTRICT), `UNIQUE (user_id, product_id)`, index `(user_id, created_at DESC)`. Rows are inserted or deleted, never updated.
- Module `wishlist`; routes in the `signedIn` group, always the session user's own: `GET /me/wishlist` (paginated), `GET /me/wishlist/ids`, `PUT /me/wishlist/:productId` (idempotent: `ON CONFLICT DO NOTHING`), `DELETE /me/wishlist/:productId`. Only public products can be added.
- Frontend: `types/wishlist.ts`, `services/wishlist.ts`, `features/wishlist/WishlistButton` (heart on `ProductCard` and the product page), `/wishlist` page, header link.
- Tests (`router/wishlist_integration_test.go`): login required, add/list/remove, adding twice keeps one row, only public products, per-user isolation, concurrent adds leave one row, pagination.

### 6.5 Risks and left out

- Store rating is computed on read over all of its products; fine at this size, `EXPLAIN` in Phase 7.
- No seller replies, review photos, helpful votes or report-abuse flow.
- Audit log covers admin actions only, not seller or buyer actions.
- Admin cannot edit users' data or roles; no admin creation from the UI (still `make seed-admin`).
- No charts; the dashboards are numbers and tables.

---

## Phase 7 — Hardening: PLAN (approved 2026-11-17 with the proposed defaults for Q1–Q8)

**7a as built (uncommitted, PROJECT_STATE §1o):** as planned, with these details: env names are `RATE_LIMIT_LOGIN_FAILURES`, `RATE_LIMIT_LOGIN_WINDOW`, `RATE_LIMIT_REGISTRATIONS`, `RATE_LIMIT_REGISTER_WINDOW` (the hard-coded `auth.DefaultLimits` is gone; config is the single source); Vite for the suite runs on `127.0.0.1:5191` with `API_TARGET=http://127.0.0.1:8082`; every spec signs up its own customer instead of reusing `penjual3`, so specs are rerunnable without a reset; the admin password lives in the git-ignored `frontend/e2e/.auth/admin-password` (or `E2E_ADMIN_PASSWORD`); a 9th spec checks 390 px on the buyer pages (Q8).

Goal: make what exists trustworthy and repeatable: automated tests that anyone can run, CI on every push, proven query plans, a security pass over the known gaps (PROJECT_STATE §9), API docs and a production-shaped build. **No new product features.** One slice at a time, stop for review after each (CLAUDE.md §16).

### 7.0 Open questions (proposed default in bold)

| # | Question | Default |
|---|---|---|
| Q1 | Which DB do E2E tests use? | **A dedicated `marketplace_e2e` DB and a second API container (Compose profile `e2e`, port 8082), reset + migrated + `seed-dev` before each run.** The dev DB stops collecting throw-away accounts |
| Q2 | Where does the Playwright suite live? | **`frontend/e2e/` with `@playwright/test` as a frontend devDependency**; local runs use the installed Edge (`channel: 'msedge'`), CI uses Playwright's Chromium |
| Q3 | Register rate limit (10/hour per IP) blocks E2E | **Make limits configurable (`RATE_LIMIT_REGISTER_PER_HOUR`, `RATE_LIMIT_LOGIN_FAILURES`); only the e2e container raises them.** Defaults stay as they are; a test proves the default still bites |
| Q4 | CI | **GitHub Actions on the existing `origin` repo**: backend unit + integration (Postgres service container), frontend build + lint + Vitest; E2E as a separate job on `main` and on manual trigger |
| Q5 | OpenAPI | **Hand-written `backend/api/openapi.yaml` (3.1) + a Go test that fails when a route in `router.go` is missing from the spec.** No annotation generator (swaggo) — keeps handlers clean |
| Q6 | Deploy prep: how far? | **Production build only, no real deploy**: `frontend/Dockerfile` (build → nginx serving `dist` and proxying `/api`, so the app stays same-origin and the CSRF model holds), `docker-compose.prod.yml`, pinned image digests/tags, `.env.example` complete. Hosting choice later |
| Q7 | Trusted proxies | **`TRUSTED_PROXIES` setting (empty = today's behaviour); the prod compose sets it to the nginx container network**, so `ClientIP()` is the real client and the rate limiter works per user. Dev stays as is (documented) |
| Q8 | Phase 3 leftovers (390 px polish, banner/recommended) | **Not in Phase 7.** Only the mobile check that the E2E suite gets for free (one 390 px project for a few key pages) |

### 7.1 Slices

| Slice | Content | Done when |
|---|---|---|
| **7a — Test suites** | Vitest + React Testing Library (jsdom) in `frontend`: zod schemas (address, product, review, category), `features/orders/status.ts` transition copy, money/date formatting, `lib/api.ts` envelope + `ApiError`, `RequireAuth`/`RequireSeller`/`RequireAdmin` redirects, `ProductCard` render, cart line issue display, loading/empty/error states of one list page. Playwright suite in `frontend/e2e/` ported from the 8 git-ignored scripts (auth, catalog, cart, address, checkout, review, wishlist, admin) with shared fixtures (login helpers, API-side setup), no `Date.now()` emails on the dev DB, no manual cleanup. Compose profile `e2e` + `make e2e` (reset DB → migrate → seed-dev → seed-admin with a generated password → run). Configurable rate limits (Q3) | `npm run test` and `make e2e` green 3 runs in a row; the dev DB untouched by E2E; `docs/e2e-reference/` can be deleted |
| **7b — CI** | `.github/workflows/ci.yml`: `backend` job (go vet, gofmt check, unit, integration on a `postgres:18.x` service, `sqlc diff` so generated code matches the SQL, migrations up → down → up), `frontend` job (npm ci, build, lint, Vitest), `e2e` job (compose `e2e` profile, Playwright report as artifact on failure). `govulncheck` and `npm audit --omit=dev` as non-blocking steps first | a pushed branch shows all jobs green; a deliberately broken test turns it red (shown once) |
| **7c — Performance & indexes** | A `make seed-load` command (dev-only) that bulk-creates ~50k products / 200k variants / 100k orders through SQL for measurement only, on a separate `marketplace_perf` DB. `EXPLAIN (ANALYZE, BUFFERS)` for: `ListPublicProducts` (each sort, search, category), `sort=rating`, `ListCartLines`, `ListCheckoutLines`, `ListOrderItems`, seller/admin order lists with and without `?status=`, seller/admin stats, store rating, public reviews. Indexes only where a plan proves it (migration `000016`, with before/after plans recorded in PROJECT_STATE). Frontend: route-level `React.lazy` (admin, seller, checkout/orders chunks first) so the main chunk drops below 500 kB | every listed query has its plan recorded; no Seq Scan on a large table for a selective query; Vite warning gone |
| **7d — Security pass** | Written review of PROJECT_STATE §9 items + OWASP API Top 10 against this codebase, findings in a table. Expected fixes: `TRUSTED_PROXIES` (Q7); expired-session cleanup in the existing sweeper (new query `DeleteExpiredSessions`, batched); test for the `Secure` cookie under `APP_ENV=production`; security headers middleware (`X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options`/CSP `frame-ancestors`; CSP for the SPA in nginx); body-size and pagination limits re-checked; `AddVariant` 20-variant cap made race-free (lock the product row); unused `cn` npm package removed (still in `package.json`). Each fix gets a regression test | review table in `docs/SECURITY_REVIEW.md`; every "fix" row links a test |
| **7e — API docs & deploy prep** | `backend/api/openapi.yaml` for every route + error codes (Q5) and the route-coverage test; served read-only at `/api/docs` only in development (or just the file — decide in review). `frontend/Dockerfile`, `docker-compose.prod.yml`, pinned `golang`/`postgres`/`alpine`/`nginx` versions, `.env.example` with all 17+ variables, `frontend/README.md` replaced, a short `docs/DEPLOY.md` (build, migrate, seed-admin, env, backups not covered) | `docker compose -f docker-compose.prod.yml up` serves the built app on one origin with `APP_ENV=production`; login works over the proxy; the Secure-cookie note documented (needs HTTPS in front) |

Order matters: 7a first (everything later is verified by it), 7b right after (so 7c–7e run in CI), 7c/7d can swap.

### 7.2 What changes in the backend

- `config`: `RATE_LIMIT_*`, `TRUSTED_PROXIES`, `SESSION_CLEANUP_BATCH` (validated like the others).
- `router.go`: `SetTrustedProxies(cfg.TrustedProxies)`; security-headers middleware after CORS.
- Sweeper: a second task deletes expired sessions (`DELETE … WHERE id IN (SELECT id … WHERE expires_at < now() LIMIT n)`), logged with a count.
- Possibly migration `000016` (indexes only, decided by 7c measurements).
- No change to any business rule, transaction or lock order.

### 7.3 Risks and left out

- E2E against a real browser stays the slowest and flakiest layer; the rule from §11 applies (repeat timing-sensitive tests, never trust one green run). Retries in CI are limited to 1 and reported.
- `seed-load` writes data with raw SQL (unlike `seed-dev`); it must never target the dev or test DB — it refuses unless the DB name ends in `_perf`.
- Rate limiter stays in memory (single instance); multi-instance and Redis are out of scope.
- No real hosting, TLS, backups, monitoring/alerting or log shipping; `DEPLOY.md` lists them as next steps.
- No email verification / password reset (backlog, not hardening).
