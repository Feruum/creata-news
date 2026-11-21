# Backend - Go Connect-RPC Server

**Parent:** `../../AGENTS.md`

## Architecture

```
cmd/server/main.go    # Entry point
internal/
├── services/         # Connect-RPC handlers (split into packages)
│   ├── events/       # Event CRUD
│   ├── organizations/# Org management
│   ├── orgmembers/   # Org member management
│   ├── roletemplates/# Role template CRUD
│   ├── rooms/        # Room catalog
│   ├── statistics/   # Analytics (13 files)
│   ├── users/        # User management
│   ├── activity/     # Activity feed
│   ├── eventattendance/ # Attendance tracking
│   ├── categories.go
│   ├── event_registrations.go
│   ├── organization_types.go
│   └── search.go
├── db/               # sqlc queries + generated
│   └── queries/      # .sql files
├── auth/             # Kratos session middleware
├── authz/            # Permission enforcement (Enforcer)
├── config/           # Server configuration
├── models/           # Shared domain models
├── perms/            # SpiceDB client wrapper
├── search/           # Meilisearch integration
├── storage/          # S3/MinIO uploads
├── telegram/         # Telegram notification helpers
├── telemetry/        # OpenTelemetry tracing
├── worker/           # Background workers
gen/                  # DO NOT EDIT - buf generated
```

## Service Pattern

```go
type EventsService struct {
    queries *db.Queries
    perms   *perms.Client
    authz   *authz.Enforcer
}

func (s *EventsService) CreateEvent(ctx context.Context, req *connect.Request[pb.CreateEventRequest]) (*connect.Response[pb.CreateEventResponse], error) {
    // 1. Permission check
    _, err := s.authz.RequireOrgPermission(ctx, req.Msg.OrganizationId, "create_event")
    if err != nil {
        return nil, err
    }
    // 2. DB query
    result, err := s.queries.CreateEvent(ctx, db.CreateEventParams{...})
    // 3. Return
    return connect.NewResponse(&pb.CreateEventResponse{...}), nil
}
```

## Authorization (authz.Enforcer)

```go
// Get local user ID
localUserID, err := s.authz.GetLocalUserID(ctx)

// Require permissions (returns error if denied)
_, err := s.authz.RequirePlatformAdmin(ctx)
_, err := s.authz.RequirePlatformStaff(ctx)
_, err := s.authz.RequireOrgPermission(ctx, orgID, "create_event")
_, err := s.authz.RequireResourcePermission(ctx, "event", eventID, "edit")

// Check permissions (returns bool)
isAdmin := s.authz.CheckPlatformAdmin(ctx, localUserID)
canEdit := s.authz.CheckOrgPermission(ctx, localUserID, orgID, "manage")
```

## sqlc Queries

Location: `internal/db/queries/*.sql`

```sql
-- name: GetEvent :one
SELECT * FROM events WHERE id = $1;

-- name: ListEventsByOrg :many
SELECT * FROM events WHERE organization_id = $1 ORDER BY start_time;
```

Regenerate: `sqlc generate`

## Commands

```bash
go run ./cmd/server/...              # Dev
go build -o ems-backend ./cmd/server # Build
go test -v -race ./internal/...      # Test
golangci-lint run                    # Lint
```

## DO NOT EDIT

- `gen/**/*.go` - buf generated
- `internal/db/*.sql.go` - sqlc generated

## Active TODOs

| File | TODO |
|------|------|
| `internal/services/statistics/low_registration.go` | Capacity placeholder |
