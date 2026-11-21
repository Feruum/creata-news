FROM oven/bun:1.4.2 AS build
WORKDIR /workspace
COPY . .
RUN bun install --frozen-lockfile
WORKDIR /workspace/events-admin
ARG VITE_WEB_URL
ARG VITE_EDITORIAL_URL
ARG VITE_HERMES_ADMIN_URL
ENV VITE_WEB_URL=$VITE_WEB_URL
ENV VITE_EDITORIAL_URL=$VITE_EDITORIAL_URL
ENV VITE_HERMES_ADMIN_URL=$VITE_HERMES_ADMIN_URL
RUN bun run build

FROM nginx:alpine
COPY --from=build /workspace/events-admin/dist /usr/share/nginx/html
COPY events-admin/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q --spider http://localhost/health || exit 1
