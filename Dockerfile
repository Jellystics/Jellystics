# Stage 1: Build frontend
FROM node:22-bookworm-slim AS frontend-builder

RUN npm install -g pnpm

WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./frontend/
RUN pnpm install --dir frontend --frozen-lockfile

COPY frontend/ ./frontend/
RUN pnpm --dir frontend build

# Stage 2: Build Go backend with embedded frontend
FROM golang:1.26-bookworm AS backend-builder

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
COPY --from=frontend-builder /app/frontend/dist ./internal/assets/web/

RUN CGO_ENABLED=0 GOOS=linux go build -o /jellystics ./cmd/server

# Stage 3: Minimal runtime image
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -yqq --no-install-recommends ca-certificates wget && \
    apt-get autoremove && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=backend-builder /jellystics .
COPY --chmod=755 entry.sh /entry.sh

HEALTHCHECK --interval=30s \
            --timeout=5s \
            --start-period=10s \
            --retries=3 \
            CMD [ "/usr/bin/wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/auth/isConfigured" ]

EXPOSE 3000

CMD ["/entry.sh"]
