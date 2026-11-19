# SECURITY_REVIEW.md — Phase 7d security pass

Written 2026-11-19 against `main` at `0cb0fc8` plus the 7d changes. Scope: the
known gaps in [PROJECT_STATE.md](PROJECT_STATE.md) §9 and the
[OWASP API Security Top 10 (2023)](https://owasp.org/API-Security/editions/2023/en/0x11-t10/)
checked against this codebase. Rules: [CLAUDE.md](../CLAUDE.md) §6, §7, §10.

**Status:** **fix** = changed in 7d, with a regression test · **ok** = already
handled, with the test that proves it · **accepted** = known, reasoned, not
changed · **7e / backlog** = belongs to a later slice.

Test paths are relative to `backend/internal/`. Integration tests need
`make test-integration`.

---

## 1. Fixes made in 7d

| # | Finding | Risk before | Fix | Regression test |
|---|---|---|---|---|
| F1 | **Page number had no upper bound** in buyer orders, seller orders and admin users / stores / orders / audit logs. The offset is `int32((page-1)*pageSize)`, so `?page=50000000&pageSize=48` overflowed into a negative `OFFSET`. PostgreSQL then refuses the query. *Found during this review; not in the plan.* | A crafted URL from any signed-in user got a **500 plus an ERROR log line** (6 endpoints, verified). No data leaked, but the log can be flooded | `MaxPage = 10_000` in `orders` and `admin`, checked as a 422. The `page` messages in `products` and `reviews` said 10000 but checked 6000; they now say 6000 | `router/paging_integration_test.go` `TestEveryListEndpoint_RejectsOutOfRangePaging` covers all 12 list endpoints. It **failed** before the fix |
| F2 | **`AddVariant` 20-variant cap was check-then-insert**, counted outside any lock | **Verified**: 10 concurrent requests on a product with 19 variants gave 7 × 201, so the product ended with **26 variants** (3 of 3 runs) | Inside the transaction: `LockStoreProduct` (`FOR NO KEY UPDATE`), then `CountProductVariants` as a **separate statement** (under READ COMMITTED a later statement sees rows committed while we waited; a count inside the locking SELECT would use the older snapshot), then insert | `router/catalog_integration_test.go` `TestAddVariant_CapHoldsUnderConcurrency`: exactly 1 × 201 and 9 × 422, 20 variants. It failed before the fix, and passed 3 of 3 runs after |
| F3 | **`ClientIP()` could never be the real client behind a proxy**: `SetTrustedProxies(nil)` was hard-coded | Behind nginx every user would share nginx's IP, so 10 sign-ups per hour for the whole site | `TRUSTED_PROXIES` (IPs/CIDRs). Empty, the default, keeps the old behaviour. Hostnames and bad CIDRs fail startup. `0.0.0.0/0` and `::/0` are refused, because trusting everyone lets a client pick its own IP | `router/router_test.go` `TestClientIP_TrustedProxies` (forged header ignored; trusted peer believed; right-most untrusted entry wins). `config/config_test.go` `TestLoad_TrustedProxies`. `router/auth_integration_test.go` `TestRegister_RateLimitKeysOnTheRealClientBehindATrustedProxy` |
| F4 | **Expired sessions were deleted only when the same user logged in again** | `user_sessions` grew without bound for users who never return. Housekeeping, not an access risk: `GetActiveSession` already ignores expired rows | Query `DeleteExpiredSessions` (`DELETE … WHERE id IN (SELECT … LIMIT n)`), looped until a partial batch, by `auth.RunSessionCleanup`. It runs as its own goroutine in `main.go` on `CHECKOUT_SWEEP_INTERVAL`, and `main.go` waits for it before closing the pool. `SESSION_CLEANUP_BATCH` defaults to 500 | `router/auth_integration_test.go` `TestCleanupExpiredSessions` (batch 2 with 3 expired runs twice; the active session survives). Wiring check: 26 expired sessions in `marketplace_e2e` dropped to 0 when `api-e2e` started |
| F5 | **`Secure` cookie path had no test** | A regression would send the session cookie over plain HTTP in production | `config.SecureCookies()` (= production, with no separate switch). `main.go` and the test app both build the auth handler through it | `config/config_test.go` `TestSecureCookies_FollowAppEnv`. `router/auth_integration_test.go` `TestSessionCookie_SecureOnlyInProduction` checks Secure + HttpOnly + SameSite=Lax on register, login and logout, and no Secure in development |
| F6 | **No security headers** on API responses | A JSON body that echoes user text (e.g. a review comment) could be sniffed or framed by an old or odd client. Responses with personal data could be cached | `middleware.SecurityHeaders` sits right after RequestID, so every answer gets it, including 404, 405, a recovered panic and a refused preflight. It sets `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, `Referrer-Policy: strict-origin-when-cross-origin` and `Cache-Control: no-store` | `middleware/security_headers_test.go`. `router/router_test.go` `TestSecurityHeaders_OnEveryKindOfResponse` covers 6 response kinds. Also checked with `curl -i` on the running dev API |
| F7 | **`npm audit --omit=dev`: 4 moderate advisories** (`shadcn` → `ts-morph` → `minimatch` ReDoS). Also the unused `cn` package | Build-time only, not reachable from the browser bundle, but reported as a runtime dependency | `cn` removed (nothing imports it). `shadcn` moved to `devDependencies`; it is still needed at build time for `@import "shadcn/tailwind.css"`. The lockfile changed only flags and the `cn` entry; no other version changed | `npm audit --omit=dev` → **0 vulnerabilities**. Build, lint, Vitest 107/107 and E2E 10/10 (twice) all pass |

---

## 2. OWASP API Security Top 10 (2023) against this codebase

| OWASP | How this codebase handles it | Status | Evidence |
|---|---|---|---|
| **API1 Broken Object Level Authorization** | Ownership by construction. Seller queries filter by the `store_id` taken from the session. Buyer queries filter by `user_id` / `buyer_id`. Cart lines are filtered by the caller's cart. A foreign id gives 404, never 403, so object existence is not revealed | ok | `TestSellerCannotTouchAnotherSellersProduct`, `TestCart_OtherUsersCannotTouchMyItems`, `TestCheckout_OtherBuyersSeeNothing`, `TestPayment_RequestRulesAndOwnership`, `TestSellerOrders_EachStoreSeesOnlyItsOwn`, `TestWishlist_EachUserSeesOnlyTheirOwn`, `TestReviews_OnlyTheBuyersDeliveredItems` |
| **API2 Broken Authentication** | Argon2id. Opaque 32-byte token, of which only SHA-256 is stored. Rotated on login, deleted on logout. Suspended users are filtered on every request. Same 401 and timing for an unknown email and a wrong password. Login limited per IP+email, register per IP. Secure cookie in production (F5). Expired sessions cleaned up (F4) | ok + fix | `TestLogin_UnknownEmailAndWrongPasswordLookIdentical`, `TestLogin_RotatesTheSession`, `TestLogout_InvalidatesTheSessionServerSide`, `TestExpiredSessionIsRejected`, `TestSuspendedUser_IsLoggedOutAndCannotLogIn`, `TestLogin_RateLimitedAfterConsecutiveFailures`, `TestRegister_RateLimitedPerIP`, F3, F4, F5 |
| **API3 Broken Object Property Level Authorization** (mass assignment / over-exposure) | Strict JSON: unknown fields are rejected (`role`, `price`, `total` cannot be smuggled in). Prices, totals, `store_id`, `buyer_id` and review `product_id`/`store_id` all come from the database, never from the request. Response DTOs are explicit: no password hash, and no reviewer email or id in the public review list | ok | `platform/request/json_test.go` (`unknown field such as role`), `TestRegister_ClientCannotChooseRole`, `TestCart_PriceIsAlwaysCurrent`, `TestCheckout_HappyPathTwoStores` (totals from server data), `TestReviews_PublicListSummaryAndEdits`, `reviews/model_test.go` `TestMaskName` |
| **API4 Unrestricted Resource Consumption** | Body cap of 1 MiB → 413. `pageSize` capped everywhere; `page` now capped everywhere (F1). Search capped to a few words (`TestBuildSearchQuery_LimitsWords`) and admin `q` to 100 characters. Per-request context timeout plus server `ReadHeaderTimeout`/`WriteTimeout`. Pool `MaxConns`. Per-user limits: 10 addresses, 20 variants (F2), cart quantity 1..99 | ok + fix | `platform/request/json_test.go` (`too large`), `TestEveryListEndpoint_RejectsOutOfRangePaging`, `TestAddresses_LimitPerUser`, `TestAddresses_ConcurrentCreatesRespectTheLimit`, `TestAddVariant_CapHoldsUnderConcurrency` |
| **API5 Broken Function Level Authorization** | `RequireAuth` / `RequireRole` on route groups. Admin routes need the admin role, which can only be granted with `make seed-admin`. An admin cannot change their own status or another admin's. Frontend guards are UX only | ok | `middleware/auth_test.go` `TestRequireRole`, `TestAdmin_EveryRouteNeedsTheAdminRole` (401 anonymous, 403 customer and seller), `TestAdmin_UserStatusRules` |
| **API6 Unrestricted Access to Sensitive Business Flows** | Checkout: idempotency key, the cart row locked, inventory locked in `variant_id` order, conditional updates. Payment events are idempotent by `eventId`. Sellers cannot buy their own products. Sign-up limited per IP | ok | `TestCheckout_IdempotencyKey`, `TestCheckout_ConcurrentSubmitsWithTheSameKey`, `TestCheckout_ConcurrentBuyersNeverOversell`, `TestCheckout_ConcurrentLifecycleRequestsHaveOneWinner`, `TestCheckout_CannotBuyFromOwnStore`, `TestCart_SellerCannotBuyOwnProduct` |
| **API7 Server Side Request Forgery** | The server never fetches a URL. Image URLs are validated as http(s) and only rendered by the browser | ok (n/a) | Code read: no outgoing HTTP client in `backend/` (`grep -r "http.Get\|http.Client"` finds none outside tests) |
| **API8 Security Misconfiguration** | Config fails fast on bad values. CORS is an exact allow-list with no `*`. CSRF uses SameSite=Lax, an Origin-vs-Host check and JSON-only bodies. Errors use one envelope and never return SQL errors or stacks (F1 was a 500 with a generic body, not a leak). Security headers (F6). Gin release mode in production | ok + fix | `config/config_test.go`, `middleware/same_origin_test.go` `TestSameOrigin`, `TestCrossSiteLoginIsRejected`, `TestCORS_DisallowedOriginPreflight`, `TestSecurityHeaders_OnEveryKindOfResponse` |
| **API9 Improper Inventory Management** | All routes live in one file (`router/router.go`). There are no old versions and no debug endpoints. `/health` and `/ready` reveal nothing about the infrastructure. The OpenAPI spec and its route-coverage test are 7e | 7e | PLAN.md §7.1 7e |
| **API10 Unsafe Consumption of APIs** | No third-party APIs: payment and shipping are simulated. Webhook signature checks become required when a real gateway arrives (CLAUDE.md §7) | ok (n/a) / backlog | — |

---

## 3. PROJECT_STATE §9 security items

| §9 item | Decision |
|---|---|
| Rate limiter keyed on `ClientIP()` = Docker gateway | **Production fixed by F3** once `TRUSTED_PROXIES` names nginx (7e compose). **Dev unchanged by design**: the Vite proxy sends no `X-Forwarded-For` and every dev request comes from one machine anyway |
| Expired sessions only cleaned on that user's next login | **Fixed (F4)** |
| `Secure` cookie path untested | **Fixed (F5)** |
| `AddVariant` 20-variant race | **Fixed (F2)**; the race was proven, not assumed |
| `npm audit` advisories through `shadcn` | **Fixed (F7)** |
| Rate limiter in memory (single instance) | **accepted**: there is one API instance. Redis / a shared store is out of scope (PLAN.md §7.3) |
| No CSRF token (SameSite + Origin + JSON only) | **accepted**: the frontend stays same-origin (nginx proxies `/api` in 7e). Revisit only if the SPA is ever hosted on another site |
| Images are arbitrary http(s) URLs | **accepted / backlog**: a seller can hotlink any host, and a viewer's browser then contacts it. No server fetch, so no SSRF. Decide together with real image upload/storage |
| Payment event after expiry not stored | **backlog**: matters when a real gateway (and its signatures) arrives |
| No email verification / password reset | **backlog** (not hardening) |

---

## 4. Decided during 7d (not code changes)

- **No index on `user_sessions.expires_at`.** Measured on `marketplace_perf` with 208 000 sessions (in a rolled-back transaction): the first batch with expired rows took 4.8 ms, and the steady state with nothing expired took 9.9 ms (a seq scan once a minute, in the background). An index would add a write to every login to save ~10 ms per minute.
- **API CSP is `default-src 'none'`**, because the API only serves JSON. The SPA's CSP (scripts, styles, images, `connect-src`) belongs in nginx in **7e**, as does **HSTS** (it only means something over HTTPS, and TLS ends at the proxy).
- **Not changed (Q3 of the 7d plan):** CORS rules, rate-limit algorithm and `Retry-After`, Argon2 parameters. They were reviewed and are fine as they are.
- **Session cleanup is its own goroutine**, not a task inside the checkout sweeper. The plan said "in the existing sweeper", but `checkout` should not know about sessions, so each module owns its own job. They share the interval and the shutdown wait in `main.go`.

## 5. Not verified

- `TRUSTED_PROXIES` through a real nginx: needs the 7e production compose.
- `Secure` cookie over real HTTPS: only the `Set-Cookie` header is tested.
- Several API instances running the cleanup at the same time: reasoned harmless (two DELETEs of the same rows), but not tested.
