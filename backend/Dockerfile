# ==============================================================================
# EMS Backend Dockerfile - Multi-Stage Build for Go + Connect-RPC
# ==============================================================================

# Stage 1: Builder
FROM golang:1.26.10-alpine@sha256:9ce231b28998b32e116927c61aa5c406183061d4f3f2d61b292c88e7f6bdf7ca AS builder
WORKDIR /app

ARG SERVICE_VERSION=1.0.0
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Keep the compiler fixed to the verified builder image
ENV GOTOOLCHAIN=local

# Copy go mod files first for better caching
COPY apps/backend/go.mod apps/backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY apps/backend/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -X github.com/studyverse/ems-backend/internal/buildinfo.Version=${SERVICE_VERSION} -X github.com/studyverse/ems-backend/internal/buildinfo.Commit=${GIT_COMMIT} -X github.com/studyverse/ems-backend/internal/buildinfo.BuiltAt=${BUILD_TIME}" \
    -o /app/server \
    ./cmd/server

# Bundle the existing Drizzle migrator into the application image.
FROM oven/bun:1.4.2-alpine@sha256:d888c0ae6c86d7866ff10c5aafdd9077b36aee6455b33dd270fb93c0dd5cef6f AS migrations
WORKDIR /workspace
COPY package.json bun.lock nx.json tsconfig.base.json ./
COPY apps/agent-admin/package.json ./apps/agent-admin/
COPY apps/events-admin/package.json ./apps/events-admin/
COPY apps/map/package.json ./apps/map/
COPY apps/sundayopen/package.json ./apps/sundayopen/
COPY apps/telegram-bot/package.json ./apps/telegram-bot/
COPY packages/db/package.json ./packages/db/
COPY packages/env/package.json ./packages/env/
COPY packages/proto/package.json ./packages/proto/
COPY packages/registration-forms/package.json ./packages/registration-forms/
COPY scripts/release/install-dependencies.sh ./install-dependencies.sh
RUN --mount=type=cache,target=/root/.bun/install/cache,sharing=locked sh ./install-dependencies.sh
COPY packages/db/scripts/migrate-runtime.ts ./packages/db/scripts/migrate-runtime.ts
COPY packages/db/scripts/automatic-migrations.ts ./packages/db/scripts/automatic-migrations.ts
COPY scripts/release/verify-runtime-migrations.ts ./scripts/release/verify-runtime-migrations.ts
RUN bun build packages/db/scripts/migrate-runtime.ts --target=node --format=cjs --outfile /migrate.cjs

# Stage 2: Runtime
FROM oven/bun:1.4.2-alpine@sha256:d888c0ae6c86d7866ff10c5aafdd9077b36aee6455b33dd270fb93c0dd5cef6f AS runtime
WORKDIR /app

ARG SERVICE_VERSION=1.0.0
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

LABEL org.opencontainers.image.source="https://github.com/StudyVerseApp/ems-monorepo" \
      org.opencontainers.image.version="${SERVICE_VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Copy binary from builder
COPY --from=builder /app/server /app/server
COPY --from=migrations /migrate.cjs /app/migrations/migrate.cjs
COPY packages/db/migrations /app/migrations/sql
COPY --chmod=755 scripts/release/runtime-entrypoint.sh /app/docker-entrypoint.sh

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (Azure Container Apps expects 8080)
EXPOSE 8080

# Set PORT env var for the app
ENV PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
    CMD wget -q --spider http://localhost:${PORT:-8080}/health/live || exit 1

# Run the binary
ENTRYPOINT ["/app/docker-entrypoint.sh", "/app/server"]
