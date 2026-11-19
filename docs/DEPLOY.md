# DEPLOY.md: running the production build

This describes the production-shaped stack (`docker-compose.prod.yml`, Phase 7e).
It was verified on one machine. It has **not** been deployed to real hosting
yet; §7 lists what is still missing before that.

## 1. What you get

```
browser ──HTTPS──▶ [your TLS terminator] ──HTTP──▶ web (nginx :8080) ──▶ api (Go :8081) ──▶ postgres
                                                     │
                                                     └── serves the built SPA (frontend/dist)
```

| Service | Image | Published? |
|---|---|---|
| `web` | `frontend/Dockerfile`: Node 24 builds the SPA, then `nginx:1.30.7-alpine3.24` serves it and forwards `/api`, `/health` and `/ready` | yes, `127.0.0.1:8080` by default |
| `api` | `backend/Dockerfile`: `golang:1.27.4-alpine3.24` builds the binaries, which run on `alpine:3.24.4` as a non-root user | no, internal network only |
| `postgres` | `postgres:18.7-trixie`, volume `pgdata-prod` | no |
| `migrate` | `migrate/migrate:v4.18.3` (profile `tools`, run on demand) | — |

Why one origin: the browser loads the page and calls `/api` on the same
`scheme://host:port`. The session cookie is then first-party, no CORS is
involved, and the API's CSRF check (the `Origin` header must match `Host`)
holds. nginx keeps the browser's `Host` header for that reason.

Production behaviour comes from `APP_ENV=production`:

- the session cookie is `Secure`, so it is only sent over HTTPS;
- Gin runs in release mode;
- `seed-dev` refuses to run.

## 2. Configure

```bash
cp .env.prod.example .env.prod     # git-ignored
# edit .env.prod:
#   POSTGRES_PASSWORD   e.g. openssl rand -hex 24
#   PUBLIC_ORIGIN       what people type, e.g. https://shop.example.com
#   WEB_BIND / WEB_HOST_PORT  where nginx listens on the host
```

Always pass `--env-file .env.prod`. Without it, Compose reads the **dev**
`.env`. To shorten the commands:

```bash
alias prod='docker compose -f docker-compose.prod.yml --env-file .env.prod'
```

All optional API settings are listed in `.env.example`. Add any you need to
the `api` service's `environment` in `docker-compose.prod.yml`.

## 3. First start

```bash
prod up -d postgres
prod run --rm migrate up                 # applies every migration (000001 … 000016)
prod up -d --build                       # api waits for postgres, web waits for a healthy api
prod ps                                  # all three healthy / up

# The first admin. The password comes from the environment and is never echoed.
ADMIN_PASSWORD='…' prod run --rm -e ADMIN_PASSWORD --entrypoint /app/seed-admin api \
  -email admin@example.com -name "Admin"
```

On Windows Git Bash, prefix the `run` commands with `MSYS_NO_PATHCONV=1`.
Otherwise `/app/seed-admin` is rewritten into a Windows path and the container
fails with `exec: "C:/Program"…`.

Check it:

```bash
curl -s http://127.0.0.1:8080/ready                  # {"success":true,"data":{"database":"up",…}}
curl -sI http://127.0.0.1:8080/ | grep -i content-security-policy
```

## 4. HTTPS in front (required for real use)

nginx here speaks plain HTTP on purpose. TLS certificates and their renewal
depend on where the app is hosted. Put an HTTPS terminator in front of `web`,
for example Caddy, Traefik, a cloud load balancer, or a second nginx with
certificates.

- **The login only works over HTTPS** (or on `http://localhost`, which browsers
  treat as secure). The cookie is `Secure`, so over plain HTTP from another
  host the browser drops it and every request after login is 401.
- **HSTS** (`Strict-Transport-Security`) belongs on the HTTPS listener. It is
  not set here, because on plain HTTP it means nothing.
- **Client IPs.** nginx replaces `X-Forwarded-For` with the address it saw
  (`$remote_addr`), and the API trusts that header only from nginx
  (`TRUSTED_PROXIES: 172.31.240.10`). With a terminator in front,
  `$remote_addr` becomes the terminator. Configure nginx's real-IP module so it
  is the client again. Without this, every user shares one login/sign-up rate
  limit.

  ```nginx
  # frontend/nginx/default.conf, inside server { }
  set_real_ip_from 10.0.0.5;          # the terminator's address, and only that
  real_ip_header   X-Forwarded-For;
  ```

- Keep `WEB_BIND=127.0.0.1` when the terminator runs on the same host, so the
  plain-HTTP port is not reachable from outside.

## 5. Updates

```bash
git pull
prod build
prod run --rm migrate up          # migrations first, while the old api still runs
prod up -d                        # recreates api and web with the new images
```

Running migrations while the old API still serves is safe only if the old
code works with the new schema (additive changes: new tables, columns with
defaults, indexes). Read a migration before applying it. If it renames or
drops something, stop the api first (`prod stop api`). Check `prod run --rm
migrate version` before and after. A failed migration
leaves the database "dirty". Read the error, fix the cause, then `migrate force
<version>` (see `make migrate-force` for the dev equivalent). Do not repeat
`up` blindly.

Image tags are exact. Bump them on purpose, in all places at once:
`backend/Dockerfile`, `frontend/Dockerfile`, both compose files, the
`Makefile` (`GO_IMAGE`) and `.github/workflows/ci.yml`. For Postgres, stay on
the same OS line (`-trixie`), because a different libc can change text sort
order under existing indexes. A major-version upgrade (18 → 19) needs
`pg_upgrade` or a dump and restore, not a tag change.

## 6. Stop and remove

```bash
prod down          # stops; keeps the pgdata-prod volume
prod down -v       # ALSO DELETES THE DATABASE — only for a throw-away test stack
```

## 7. Not covered yet (before real hosting)

| Topic | State |
|---|---|
| **Backups** | none. At minimum, a scheduled `pg_dump` (`prod exec postgres pg_dump -U … -Fc …`) copied off the host, plus one tested restore |
| TLS / certificates | §4: by the host's terminator |
| Monitoring and alerting | `/ready` exists for a health probe; no metrics, no alerts |
| Logs | JSON on stdout (`prod logs api`); no shipping or retention |
| Several API instances | not supported: the rate limiter is in memory, and the checkout sweeper and session cleanup run in every instance (safe with `SKIP LOCKED`, but untested) |
| Secrets | `.env.prod` on the host; no secret manager |
| Email (verification, password reset) | not built (backlog) |
| Image uploads | product images are external URLs; the SPA's CSP allows any `http(s)` image host for that reason |
| API docs | `backend/api/openapi.yaml` (OpenAPI 3.1) is a file in the repo, not served by the app. Open it in any OpenAPI viewer. `make api-lint` validates it |
