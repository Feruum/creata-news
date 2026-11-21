# =============================================================================
# Stage 1: Dependencies
# =============================================================================
FROM oven/bun:1.4.2-alpine@sha256:d888c0ae6c86d7866ff10c5aafdd9077b36aee6455b33dd270fb93c0dd5cef6f AS deps
WORKDIR /app

# Install workspace dependencies from lockfile manifests first for cache reuse.
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

# =============================================================================
# Stage 2: Builder
# =============================================================================
FROM deps AS builder
WORKDIR /app

# Copy only source that telegram-bot build depends on.
COPY apps/telegram-bot ./apps/telegram-bot
COPY packages/proto ./packages/proto
COPY packages/db/scripts/migrate-runtime.ts ./packages/db/scripts/migrate-runtime.ts
COPY packages/db/scripts/automatic-migrations.ts ./packages/db/scripts/automatic-migrations.ts
COPY scripts/release/verify-runtime-migrations.ts ./scripts/release/verify-runtime-migrations.ts
RUN bun build packages/db/scripts/migrate-runtime.ts --target=node --format=cjs --outfile /app/migrate.cjs

# Build workspace libraries consumed by telegram-bot.
RUN cd packages/proto && bun run tsc

# Build telegram-bot directly to avoid Nx project graph side effects.
RUN cd apps/telegram-bot && bun run build

# =============================================================================
# Stage 3: Runner
# =============================================================================
FROM oven/bun:1.4.2-alpine@sha256:d888c0ae6c86d7866ff10c5aafdd9077b36aee6455b33dd270fb93c0dd5cef6f AS runner
WORKDIR /app

ARG SERVICE_VERSION=1.0.0
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

LABEL org.opencontainers.image.source="https://github.com/StudyVerseApp/ems-monorepo" \
      org.opencontainers.image.version="${SERVICE_VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}"

ENV NODE_ENV=production
ENV SERVICE_VERSION=${SERVICE_VERSION}
ENV GIT_COMMIT=${GIT_COMMIT}
ENV BUILD_TIME=${BUILD_TIME}

# Create non-root user
RUN addgroup --system --gid 1001 botjs && \
    adduser --system --uid 1001 bot

# node-fetch checks signal.constructor.name. Keep the AbortSignal polyfill
# outside the bundle so Bun cannot rename its class during module linking.
COPY --from=builder /app/apps/telegram-bot/dist ./apps/telegram-bot/dist
COPY --from=builder /app/node_modules/.bun/abort-controller@3.0.0/node_modules/abort-controller ./node_modules/abort-controller
COPY --from=builder /app/node_modules/.bun/event-target-shim@5.0.1/node_modules/event-target-shim ./node_modules/event-target-shim
COPY --from=builder /app/migrate.cjs /app/migrations/migrate.cjs
COPY apps/telegram-bot/drizzle /app/migrations/sql
COPY --chmod=755 scripts/release/runtime-entrypoint.sh /app/docker-entrypoint.sh

# Set ownership
RUN chown -R bot:botjs /app

USER bot

# Grammy bot needs graceful shutdown handling
STOPSIGNAL SIGTERM

# Health check - verify the bot HTTP liveness endpoint.
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:${HTTP_PORT:-3001}/health/live || exit 1

# Start the bot
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["bun", "apps/telegram-bot/dist/index.js"]
