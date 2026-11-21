# Events Admin - React Frontend

**Parent:** `../../AGENTS.md`

React 19 + Vite + TanStack Router admin panel for event management.

## Architecture

```
src/
├── routes/           # TanStack Router file-based routing
│   ├── __root.tsx    # App shell, providers
│   ├── _authenticated.tsx  # Auth guard layout
│   └── _authenticated/     # Protected pages
├── features/         # Domain modules (9 features)
│   ├── bookings/     # Booking requests, conflicts, approve/reject
│   ├── calendar/     # VENDORED — event calendar with DnD, views
│   ├── categories/   # Event categories CRUD
│   ├── events/       # Events CRUD, forms, details, import
│   ├── organizations/# Org structure, members, roles, invites
│   ├── search/       # Command palette search
│   ├── statistics/   # Charts, leaderboards, analytics
│   ├── theme/        # Theme toggle hook
│   └── users/        # User management, roles, notifications
├── shared/           # Cross-cutting code
│   ├── ui/           # shadcn primitives (53 components)
│   ├── components/   # App shell (sidebar, data-table, analytics)
│   ├── hooks/        # Generic hooks (use-debounce, use-mobile, etc.)
│   └── lib/          # Utilities (auth, kratos, api, excel, form-utils)
├── App.tsx           # Root component
├── main.tsx          # Entry point
└── index.css         # Global styles
```

## Feature Conventions

- Each feature has `index.ts` barrel + `components/` minimum
- Cross-feature imports go through barrel: `import { X } from '@/features/events'`
- Within-feature imports are direct: `import { Y } from './event-card'`
- All files kebab-case, hooks are `.ts` not `.tsx`
- Component suffixes: `-modal`, `-dialog`, `-form`, `-skeleton`, `-section`, `-cell`
- VENDORED: `calendar/` internals (views/, skeletons/, dialogs/, contexts/, dnd/) — do NOT restructure

## Routing (TanStack Router)

File-based in `src/routes/`:
- `__root.tsx` - App shell, providers
- `_authenticated.tsx` - Auth guard layout
- `_authenticated/events.tsx` → `/events`
- `auth/login.tsx` → `/auth/login`

```tsx
const { eventId } = Route.useParams()
```

## Data Fetching

TanStack Query + Connect-RPC via `@connectrpc/connect-query`:

```tsx
import { useSuspenseQuery, useMutation } from '@connectrpc/connect-query'
import { listEvents, createEvent } from '@studyverse/proto/...'

const { data } = useSuspenseQuery(listEvents, { page: 1, limit: 100 })

const mutation = useMutation(createEvent, {
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ... })
})
```

## Auth

```tsx
const { user, isAuthenticated, logout } = useAuth()
```

Files: `src/shared/lib/auth.ts`, `src/shared/lib/kratos.ts`

## Fast-Refresh Rule

`.tsx` files must export ONLY components (no hooks alongside).
Hook files are `.ts` not `.tsx`.

## Alias

`@/` → `src/`

## Testing

- **Playwright e2e** — `playwright.config.ts`, specs in `e2e/`
- Minimal unit test coverage

## Large Files (Refactor Candidates)

| File | Lines |
|------|-------|
| `shared/components/admin-data-table/index.tsx` | 682 |
| `features/calendar/helpers.ts` | 451 |
| `features/events/components/create-event-form.tsx` | 382 |
