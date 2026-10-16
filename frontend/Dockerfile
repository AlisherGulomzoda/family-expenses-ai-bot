# syntax=docker/dockerfile:1

# Build the SPA with Node, serve it with nginx. nginx also proxies /api to the
# backend, so the browser talks to one origin (no CORS) and only this service
# is published.

FROM node:24-alpine AS build
WORKDIR /app

COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund

COPY . .
# Type check + build; the MSW worker is only for the mock mode in development.
RUN npm run build && rm -f dist/mockServiceWorker.js

FROM nginx:1.30-alpine
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
COPY nginx/security-headers.conf /etc/nginx/snippets/security-headers.conf
COPY --from=build /app/dist /usr/share/nginx/html

HEALTHCHECK --interval=10s --timeout=3s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1/nginx-health || exit 1
