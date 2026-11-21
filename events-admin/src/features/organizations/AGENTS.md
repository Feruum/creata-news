# Organizations Feature

**Parent:** `../../AGENTS.md`

Organization management: structure, members, roles, invites.

## Structure

```
organizations/
├── api/
│   ├── keys.ts              # Query key factories
│   ├── queries.ts           # useOrganizations, useOrganizationMembers, useOrganizationTypes, etc.
│   └── mutations.ts         # useCreateOrganization, useAddOrganizationMember, role template CRUD
├── components/
│   ├── create-organization-form.tsx
│   ├── organization-info-modal.tsx
│   ├── organization-structure-chart.tsx   # Org hierarchy visualization
│   ├── role-template-editor.tsx
│   ├── add-member-dialog.tsx
│   ├── org-membership-history.tsx
│   ├── user-membership-timeline.tsx
│   ├── social-icon.tsx
│   └── index.ts             # Component barrel
├── columns.tsx              # TanStack Table column definitions (images, badges, socials)
└── index.ts                 # Feature barrel — keys, queries, mutations, columns, components
```

## Data Fetching

Hybrid pagination: server-side by default, switches to client-side (fetch up to 1000) when filters are active. Table state persisted in URL via `nuqs`.

Route: `src/routes/_authenticated/organizations.tsx`

## Key Patterns

- Organization types + role templates are editable (CRUD)
- Members have roles within orgs, tracked with membership history
- Structure chart visualizes org hierarchy
- Column definitions in `columns.tsx` — custom renderers for images, status badges, social links

## DO NOT

- Confuse with `apps/map/src/features/organizations/` — separate feature, different app
- Look for calendar logic here — calendar is a separate vendored feature
