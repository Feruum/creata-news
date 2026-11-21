FROM oven/bun:1.4.2 AS build
WORKDIR /app
COPY package.json bun.lock ./
COPY packages ./packages
COPY creatanews ./creatanews
COPY events-admin/package.json ./events-admin/package.json
COPY agent-admin/package.json ./agent-admin/package.json
COPY telegram-bot/package.json ./telegram-bot/package.json
RUN bun install --frozen-lockfile
ENV NEXT_TELEMETRY_DISABLED=1
RUN cd creatanews && bun run build
FROM oven/bun:1.4.2 AS runtime
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 HOSTNAME=0.0.0.0 PORT=3002
COPY --from=build /app/creatanews/.next/standalone ./
COPY --from=build /app/creatanews/.next/static ./creatanews/.next/static
COPY --from=build /app/creatanews/public ./creatanews/public
USER bun
EXPOSE 3002
CMD ["bun", "creatanews/server.js"]
