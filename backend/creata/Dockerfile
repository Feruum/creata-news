FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend/creata/go.mod backend/creata/go.sum ./
RUN go mod download
COPY backend/creata/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /creata-api ./cmd/server
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && addgroup -S creata && adduser -S -G creata creata
WORKDIR /app
RUN mkdir -p /app/uploads && chown -R creata:creata /app/uploads
COPY --from=build /creata-api /app/creata-api
COPY --chown=creata:creata backend/creata/content /app/content
ENV CREATA_CONTENT_IMPORT_DIR=/app/content
USER creata
EXPOSE 5555
ENTRYPOINT ["/app/creata-api"]
