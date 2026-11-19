# frontend

The web app of the marketplace: React 19, TypeScript, Vite, Tailwind 4 and
shadcn/ui (Radix). It talks only to the Go API under `/api/v1`.

Project rules and architecture: [../CLAUDE.md](../CLAUDE.md). How things fit
together: [../docs/DEV_GUIDE.md](../docs/DEV_GUIDE.md). API contract:
[../backend/api/openapi.yaml](../backend/api/openapi.yaml).

## Commands

Run them in this folder. The API must be running (`make up` in the repo root).

| Command | What it does |
|---|---|
| `npm run dev` | Vite dev server. It proxies `/api`, `/health` and `/ready` to `127.0.0.1:8081`, so the browser sees one origin |
| `npm run build` | `tsc -b` (type check, `e2e/` included) and `vite build` into `dist/` |
| `npm run lint` | oxlint |
| `npm run test` | Vitest + React Testing Library (`src/**/*.test.ts(x)`). On Windows, start from an upper-case drive path (`cd /D/...`) |
| `npm run e2e` | Playwright. Use `make e2e` from the repo root: it prepares the separate `marketplace_e2e` database and API first |

## Layout

- `src/pages/`: one component per route. Routes are in `src/app/`; most are lazy-loaded.
- `src/features/<area>/`: the parts of a feature (auth, cart, checkout, orders, seller, admin, …).
- `src/services/`: the only code that calls the API (`lib/api.ts` handles the envelope and `ApiError`).
- `src/types/`: TypeScript mirrors of the Go DTOs. Keep them in sync with `openapi.yaml`.
- `src/components/ui/`: shadcn components (owned by us after generation). `components/shared/`: our own building blocks.

## Production

`Dockerfile` builds `dist/` and serves it with nginx (`nginx/default.conf`).
nginx also forwards `/api` to the API, so production is same-origin like
development, and it adds the SPA's security headers and CSP. See
[../docs/DEPLOY.md](../docs/DEPLOY.md).
