# syntax=docker/dockerfile:1

# Production image of the web app (Phase 7e): build the SPA with Node, then
# serve the static files with nginx, which also forwards /api to the Go API
# (see nginx/default.conf). Development keeps using `npm run dev`.
#
# Exact tags on purpose (see backend/Dockerfile).

# ---- Build stage ----
FROM node:24.22.3-alpine3.24 AS build

WORKDIR /app

# Install from the lockfile first so this layer is cached until it changes.
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# tsc -b (type check, including e2e/) + vite build -> /app/dist
RUN npm run build

# ---- Runtime stage ----
FROM nginx:1.30.7-alpine3.24

# Replace the default site; the security-header snippet is included per location.
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
COPY nginx/security-headers.conf /etc/nginx/snippets/security-headers.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
