# Roadmap — Phase 0 to 7

Penjelasan lengkap semua fase, apa yang dikerjakan, dan acceptance criteria.

---

## Phase 0 — Foundation (Infrastructure & API Skeleton)

**Goal:** Buat struktur backend siap terima fitur, koneksi ke DB, response envelope, health checks, dan migrations framework.

**Tasks (10 item):**

| # | What | Why | Output |
|---|------|-----|--------|
| 0.1 | Restruktur Go: pindah spike ke `backend/cmd/api`, hapus spike lama, init modul `github.com/RavaGhiffari/ecommerceCopyGoReact` | Foundation yang benar | folder structure, `go.mod` |
| 0.2 | Config dari env var: `APP_PORT`, `DATABASE_URL`, pgxpool params, CORS, cookie, app env | Tidak ada hardcoded values | `internal/config/config.go` |
| 0.3 | pgxpool factory: min/max conn, lifetime, health check, ping on startup | DB reliability | `internal/database/postgres.go` |
| 0.4 | Response envelope: `{success, data, message}` dan `{success, error: {code, message, details}}` | Konsistensi API | `internal/platform/response/` |
| 0.5 | Logger (JSON `slog`), request ID middleware, panic recovery, CORS, timeout, graceful shutdown | Observability dan stability | `internal/platform/logger/`, `internal/middleware/`, `main.go` |
| 0.6 | `GET /health` (liveness) dan `GET /ready` (DB ping) | Know app status | `internal/router/routes.go` |
| 0.7 | golang-migrate + Makefile: `make migrate-up`, `migrate-down`, `migrate-version`, `migrate-create name=...` | Migration workflow yang terstruktur | `Makefile`, `db/migrations/000001_init.{up,down}.sql` |
| 0.8 | sqlc.yaml + dummy query (`SELECT 1`) | Prove sqlc generation works | `sqlc.yaml`, `db/queries/dummy.sql` |
| 0.9 | Frontend: `VITE_API_BASE_URL` env, status page calling `/health`/`/ready`, loading/error states | Real API integration | `frontend/.env.example`, `frontend/src/pages/StatusPage.tsx` |
| 0.10 | Update `docker-compose.yml` dan `backend/Dockerfile` untuk path `backend/` | Docker ready | `backend/Dockerfile`, `docker-compose.yml` |

**Acceptance Criteria:**
- ✓ `docker compose up -d --build` → Postgres + API start
- ✓ `curl http://localhost:8081/health` → `{success: true, data: {status: "ok"}}`
- ✓ `curl http://localhost:8081/ready` → checks DB connection
- ✓ `make migrate-up` → applies migrations
- ✓ `make migrate-down` → rolls back
- ✓ `sqlc generate` → succeeds (even with dummy query)
- ✓ Frontend shows API status live
- ✓ `go build ./cmd/api` dan `go test ./...` pass
- ✓ `npm run build && npm run lint` pass di frontend

**Time estimate:** 1–2 hours focused work

---

## Phase 1 — Identity & Security (Auth, Users, Sessions)

**Goal:** Users bisa register/login/logout dengan aman. Roles (customer, seller, admin) enforced server-side. CSRF protected. Sessions via cookies.

**Migrations:**
- `users` (UUID PK, email UNIQUE & lowercase, password_hash, status CHECK active/suspended, timestamps)
- `roles` (id, name — seeded: customer, seller, admin)
- `user_roles` (user_id FK, role_id FK, PK (user_id, role_id))
- `user_sessions` (UUID id PK, user_id FK, token_hash TEXT, expires_at TIMESTAMPTZ, created_at)

**Backend Work:**

**Modules:**
- `internal/modules/auth/` → handler, service, repository, dto, model
  - `POST /api/v1/auth/register` (email, password, name) → create user with customer role, return user
  - `POST /api/v1/auth/login` (email, password) → validate, create session, set HttpOnly cookie, return user
  - `POST /api/v1/auth/logout` → delete session, clear cookie
  - `GET /api/v1/auth/me` (authenticated) → return current user + roles

- `internal/modules/users/` → handler, service, repository
  - `GET /api/v1/me` (dupe of `/auth/me`, or one is removed)
  - `PATCH /api/v1/me` (name, phone, timezone, etc)
  - `internal/modules/admin/users` → admin can list/suspend users (later, Phase 7)

- `internal/middleware/` → auth, RBAC
  - `AuthMiddleware` — extract session from cookie, load user + roles into context
  - `RequireRole(role string)` — middleware that enforces role
  - `RequireAuth` — middleware that rejects unauthenticated

**Services/Helpers:**
- `internal/platform/password/` → Argon2id hashing (config: cost params from env)
- `internal/platform/token/` → session token generation (32 random bytes), hashing (SHA-256)
- `internal/platform/csrf/` → CSRF token generation, validation (double-submit + same-site cookie usually sufficient)

**Tests:**
- Unit: hashing, token generation, CSRF validation
- Integration: register flow (new user + customer role), login success/failure, session creation, logout, expired session rejection, suspended user rejection, role enforcement
- Handler: protected routes return 401 if no session, 403 if wrong role

**Frontend:**
- `src/features/auth/` → Register, Login pages (RHF + Zod)
- `src/services/auth.ts` → register, login, logout, fetchMe (via TanStack Query)
- `src/hooks/useAuth.ts` → context hook for current user
- `src/app/router.tsx` → route guards (UX only)
- Update `src/components/layout/Header.tsx` → show user name or login link

**Acceptance Criteria:**
- ✓ Register new user → customer role assigned, password hashed
- ✓ Login with correct email/password → session created, cookie set
- ✓ Login with wrong password → `{success: false, error: {...}}`
- ✓ Logout → session deleted, cookie cleared
- ✓ Unauthenticated request to `/api/v1/me` → 401
- ✓ User with wrong role accessing seller endpoint → 403
- ✓ Session expiry → 401 on next request
- ✓ Suspended user cannot login
- ✓ Frontend shows logged-in user in header

**Time estimate:** 4–6 hours

---

## Phase 2 — Catalog (Stores, Categories, Products, Inventory Basics)

**Goal:** Sellers can create stores and products. Buyers can browse. Inventory tracked (but NOT reserved yet, that's Phase 5).

**Migrations:**
- `stores` (UUID id, user_id FK → seller, name, slug UNIQUE, description, logo_url, banner_url, location, status CHECK active/suspended, timestamps)
- `categories` (UUID id, parent_id FK self, name, slug UNIQUE, timestamps)
- `products` (UUID id, store_id FK, category_id FK, name, slug UNIQUE (store_id, slug), description, status CHECK active/archived, timestamps, search_vector tsvector + GIN)
- `product_images` (UUID id, product_id FK, image_url, sort_order)
- `product_variants` (UUID id, product_id FK, sku UNIQUE, name, price NUMERIC(12,2) CHECK >= 0, timestamps)
- `inventory` (UUID id, product_variant_id FK, available_quantity INT CHECK >= 0, reserved_quantity INT CHECK >= 0, timestamps)
- `inventory_movements` (UUID id, product_variant_id FK, movement_type VARCHAR, quantity INT, reference_type VARCHAR, reference_id UUID, actor_id UUID, created_at TIMESTAMPTZ)

**Backend:**

**Modules:**
- `internal/modules/stores/` → handler, service, repository
  - `POST /api/v1/seller/stores` (name, description, ...) → create store, promote user to seller role, return store
  - `GET /api/v1/seller/store` (authenticated seller) → return user's store
  - `PATCH /api/v1/seller/store` → update store info
  - `GET /api/v1/stores` → paginated list, filters, sorting
  - `GET /api/v1/stores/:slug` → public store detail + products

- `internal/modules/categories/` → handler, service, repository
  - `GET /api/v1/categories` → tree or flat list
  - `GET /api/v1/categories/:id` → category + related products
  - `POST /api/v1/admin/categories` (admin only) → create
  - `PATCH /api/v1/admin/categories/:id` (admin only) → update

- `internal/modules/products/` → handler, service, repository
  - `POST /api/v1/seller/products` (authenticated seller) → create product + default variant, store_id from session
  - `GET /api/v1/seller/products` → seller's products
  - `PATCH /api/v1/seller/products/:id` → seller can edit only own
  - `DELETE /api/v1/seller/products/:id` → archive (soft delete)
  - `GET /api/v1/products` → paginated, category filter, price sort, search (PostgreSQL full-text), pagination
  - `GET /api/v1/products/:id` → detail (product + images + variants + lowest price + store info)
  - `GET /api/v1/products/:id/related` → similar products

- `internal/modules/inventory/` → handler, service, repository
  - `PATCH /api/v1/seller/products/:id/inventory` (authenticated seller) → adjust available quantity, create movement row
  - Movement tracking (immutable ledger)

**Services:**
- Full-text search on products (name + description)
- Price calculation (min of active variants)
- Slug validation & uniqueness per store

**Tests:**
- Seller A cannot read/edit/delete seller B's product (403)
- Slug unique per store (same slug in different stores OK)
- Inventory adjustment creates movement row
- Search + pagination + filter + sort correctness
- Negative stock rejected by DB constraint
- Product soft-delete doesn't lose history

**Frontend:**
- `src/features/products/ProductList.tsx` → paginated list with filter/sort/search
- `src/features/products/ProductDetail.tsx` → detail + add to cart (later)
- `src/features/stores/StoreDetail.tsx` → store info + products
- `src/features/seller/SellerDashboard.tsx` (stub) → seller's products
- `src/features/admin/CategoryManager.tsx` (stub) → admin categories

**Acceptance Criteria:**
- ✓ Seller creates store → promoted to seller role, store created
- ✓ Seller creates product → variant created with price
- ✓ Product listing shows min variant price
- ✓ Seller A cannot edit seller B's product
- ✓ Search + pagination + filter + sort work
- ✓ Inventory movements audited
- ✓ Frontend shows real products, no hardcoding

**Time estimate:** 6–8 hours

---

## Phase 3 — Marketplace UI

**Goal:** Homepage, navigation, product discovery flows feel like a real marketplace.

**Frontend:**
- `src/pages/HomePage.tsx` → banner, category shortcuts, popular/recommended sections, grid
- `src/components/layout/MainNav.tsx` → logo, search, categories, cart, user menu, seller signup
- `src/components/product/ProductCard.tsx` → image, name, price, store, rating (stub), sold count
- `src/features/search/SearchPage.tsx` → search results with filters
- `src/features/categories/CategoryPage.tsx` → category page + products in category
- `src/pages/NotFound.tsx`, `src/pages/Error.tsx` → error handling

**Styling:** Responsive desktop-first, Tailwind + shadcn/ui for common components

**Backend:** No new backend features, consume Phase 2 APIs

**Tests:**
- Frontend component rendering
- Navigation works
- API integration (real calls to backend)

**Acceptance Criteria:**
- ✓ Homepage loads, shows real products from API
- ✓ Search works
- ✓ Category navigation works
- ✓ Responsive layout (desktop, tablet, mobile)
- ✓ No hardcoded product data

**Time estimate:** 4–6 hours

---

## Phase 4 — Cart & Inventory Validation

**Goal:** Buyers can add/remove/update cart items. Stock validated when adding. Cart persists per user.

**Migrations:**
- `carts` (UUID id, user_id FK UNIQUE, created_at, updated_at)
- `cart_items` (UUID id, cart_id FK, product_variant_id FK, quantity INT CHECK > 0, UNIQUE (cart_id, product_variant_id), created_at, updated_at)

**Backend:**

**Modules:**
- `internal/modules/cart/` → handler, service, repository
  - `GET /api/v1/cart` (authenticated) → return user's cart with items + current prices
  - `POST /api/v1/cart/items` (authenticated) → add variant to cart (check stock available)
  - `PATCH /api/v1/cart/items/:id` (authenticated) → update quantity
  - `DELETE /api/v1/cart/items/:id` (authenticated) → remove item

**Services:**
- Validate stock when adding (query current available quantity)
- Prevent adding same variant twice (increment quantity instead)
- Return cart with items grouped by store + current product info

**Tests:**
- Add item to cart
- Update quantity
- Remove item
- Stock validation (cannot add more than available)
- Cart persistence across sessions
- Concurrent updates safety

**Frontend:**
- `src/features/cart/CartPage.tsx` → items grouped by store, quantities, remove button, summary
- `src/hooks/useCart.ts` → TanStack Query for cart state
- Add-to-cart button on product detail/card

**Acceptance Criteria:**
- ✓ Add item to cart → stored in DB
- ✓ Cannot add more than available stock
- ✓ Update quantity
- ✓ Remove item
- ✓ Cart persists across sessions
- ✓ Frontend shows cart items grouped by store

**Time estimate:** 3–4 hours

---

## Phase 5 — Checkout & Orders (The Critical Transaction)

**Goal:** Atomic multi-store checkout. One transaction validates inventory, creates orders, reserves stock, clears cart.

**Migrations:**
- `checkouts` (UUID id, user_id FK, subtotal NUMERIC, shipping_cost NUMERIC, discount NUMERIC, total_amount NUMERIC, status VARCHAR, created_at, updated_at)
- `orders` (UUID id, checkout_id FK, seller_id FK, order_number VARCHAR UNIQUE, status VARCHAR, subtotal NUMERIC, shipping_cost NUMERIC, total NUMERIC, created_at, updated_at)
- `order_items` (UUID id, order_id FK, product_variant_id FK, product_name_snapshot VARCHAR, sku_snapshot VARCHAR, unit_price_snapshot NUMERIC, quantity INT, subtotal NUMERIC, created_at)
- `order_status_histories` (UUID id, order_id FK, old_status VARCHAR, new_status VARCHAR, changed_by_id FK, reason VARCHAR, created_at)
- `payments` (UUID id, checkout_id FK, reference VARCHAR, status VARCHAR, amount NUMERIC, created_at, updated_at)

**User Addresses:**
- `user_addresses` (UUID id, user_id FK, type VARCHAR, address_line1, address_line2, city, province, postal_code, PRIMARY (user_id, type))

**Shipping:**
- `shipping_methods` (seeded: Regular, Express, Same Day with configurable cost)

**Backend:**

**Modules:**
- `internal/modules/checkout/` → handler, service, repository
  - `POST /api/v1/checkout` (authenticated buyer) → trigger full transaction:
    1. Validate user, load cart items
    2. Lock inventory rows (FOR UPDATE)
    3. Validate stock availability
    4. Compute subtotal, shipping, total (server-side)
    5. Create checkout record
    6. Create one order per store
    7. Create order items with snapshots
    8. Update inventory: `available_quantity -= qty`, `reserved_quantity += qty`
    9. Write inventory_movements for each
    10. Delete purchased items from cart
    11. Commit transaction
  - `GET /api/v1/checkout/:id` (authenticated) → return checkout + orders + items
  - Rollback on **any** error

- `internal/modules/orders/` → handler, service, repository
  - `GET /api/v1/orders` (buyer) → buyer's orders
  - `GET /api/v1/orders/:id` (buyer, with ownership check) → order detail
  - `PATCH /api/v1/orders/:id/cancel` (buyer) → cancel if allowed, release reserved stock
  - `GET /api/v1/seller/orders` (seller) → seller's incoming orders
  - `PATCH /api/v1/seller/orders/:id/status` (seller) → update status, validate transitions, log history

- `internal/modules/payments/` (stub)
  - `POST /api/v1/payments/:id/simulate` → simulate success/failure, update checkout status

**Services:**
- Checkout transaction (with idempotency key)
- Status transition validator (pending_payment → paid → processing → shipped → delivered)
- Concurrency test: 100 concurrent checkouts on 10 items stock, verify no oversell

**Tests:**
- Checkout transaction (success path)
- Transaction rollback on stock shortage
- Concurrent checkouts don't oversell
- Order items snapshot prices
- Status transitions validated
- Seller can only see/update own orders
- Buyer can cancel order (early phase)

**Frontend:**
- `src/features/checkout/CheckoutPage.tsx` → address selection, shipping method, order review, payment simulation
- `src/features/orders/OrderHistory.tsx` → list of buyer's orders
- `src/features/orders/OrderDetail.tsx` → order detail with items + status

**Acceptance Criteria:**
- ✓ Checkout is atomic (all-or-nothing)
- ✓ Inventory reserved during checkout
- ✓ Concurrent checkouts don't oversell
- ✓ Order items snapshot prices (can't change later)
- ✓ Seller can only manage own orders
- ✓ Buyer can view orders, cancel if allowed
- ✓ Frontend shows real order data

**Time estimate:** 8–10 hours (most complex phase)

---

## Phase 6 — Reviews & Frontend Completion

**Goal:** Buyers can review products after purchase. Seller + admin dashboards filled out.

**Migrations:**
- `reviews` (UUID id, product_id FK, buyer_id FK, order_id FK, rating INT CHECK 1..5, comment TEXT, UNIQUE (product_id, buyer_id), created_at, updated_at)

**Backend:**

**Modules:**
- `internal/modules/reviews/` → handler, service, repository
  - `POST /api/v1/products/:id/reviews` (authenticated buyer) → create review (only if buyer purchased the product and hasn't reviewed)
  - `GET /api/v1/products/:id/reviews` (public) → reviews + average rating
  - `PATCH /api/v1/reviews/:id` (buyer who owns it) → update review
  - `DELETE /api/v1/reviews/:id` (buyer who owns it) → delete review

**Services:**
- Average rating calculation per product
- Eligibility check (buyer purchased product)

**Tests:**
- Buyer can only review product they purchased
- One review per eligible purchase
- Rating 1–5 only

**Frontend:**
- `src/features/reviews/ReviewForm.tsx` → on order detail, show review form if eligible
- `src/features/reviews/ReviewList.tsx` → show reviews on product detail
- `src/features/seller/SellerDashboard.tsx` → filled out with store stats, product list, incoming orders, sales graph
- `src/features/admin/AdminDashboard.tsx` → user management, store management, order overview, audit logs (stubs)

**Acceptance Criteria:**
- ✓ Buyer can review eligible product
- ✓ One review per product per buyer
- ✓ Average rating displayed
- ✓ Seller dashboard shows orders + stats
- ✓ Admin can manage users/stores/categories

**Time estimate:** 4–5 hours

---

## Phase 7 — Hardening

**Goal:** Tests, security review, performance, documentation, deployment prep.

**Backend:**
- Unit + integration + E2E tests for all critical paths
- Security review: SQLi, XSS, auth bypass, rate limiting
- Query optimization: N+1 detection, index review
- API docs (OpenAPI/Swagger)
- Graceful error handling, logging

**Frontend:**
- Component tests (Vitest + RTL)
- E2E tests (Playwright): register → product search → add to cart → checkout → order
- Performance: Lighthouse, bundle size

**Deployment:**
- Docker multi-stage production builds
- Environment separation (dev/staging/prod)
- Secrets management (.env not committed)
- Database backup strategy

**Acceptance Criteria:**
- ✓ 80%+ test coverage on critical paths
- ✓ No SQLi/XSS/auth bypass found
- ✓ API documented
- ✓ E2E tests pass
- ✓ Ready for deployment

**Time estimate:** 6–10 hours

---

## Summary Table

| Phase | Focus | Features | Est. Hours |
|-------|-------|----------|-----------|
| 0 | Infrastructure | Config, DB, migrations, health checks | 2 |
| 1 | Identity | Register, login, auth middleware, roles | 5 |
| 2 | Catalog | Stores, products, inventory basics | 7 |
| 3 | UI | Homepage, search, category pages | 5 |
| 4 | Cart | Add/remove items, stock validation | 3.5 |
| 5 | Checkout | Multi-store transaction, orders | 9 |
| 6 | Reviews | Product reviews, dashboards | 4.5 |
| 7 | Hardening | Tests, security, docs, deploy | 8 |
| **Total** | — | — | **44** |

---

## Next Step

Approval has been given for **Phase 0 only**. Once Phase 0 acceptance criteria pass, wait for developer review before starting Phase 1.

**Currently:** Awaiting start of Phase 0 item 0.1 (restructure Go, init `backend/`).
