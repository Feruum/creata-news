# Calendar Feature

**Parent:** `../../AGENTS.md`

Complex drag-drop calendar with 5 view types.

## Structure

```
calendar/
├── contexts/
│   ├── calendar-context.tsx    # Main state
│   ├── dnd-context.tsx         # Drag-drop state
│   ├── use-calendar.ts         # Hook (separate for fast-refresh)
│   └── use-drag-drop.ts
├── views/
│   ├── month-view/
│   ├── week-and-day-view/
│   ├── year-view/
│   └── agenda-view/
├── dnd/                        # Drag-drop components
├── dialogs/                    # Modal dialogs
└── helpers.ts                  # Date utilities (451 lines)
```

## Provider Hierarchy

```tsx
<CalendarProvider>      {/* Events, filters, navigation */}
  <DndProvider>         {/* Drag state, drop handlers */}
    <CalendarHeader />
    <CalendarBody />
  </DndProvider>
</CalendarProvider>
```

## Drag-Drop (Native HTML5)

NOT using react-dnd or dnd-kit. Native HTML5 DnD API:

```tsx
// DraggableEvent
<div draggable onDragStart={...} onDragEnd={...}>

// DroppableArea
<div onDragOver={...} onDrop={...}>
```

Flow:
1. `onDragStart` → sets `draggedEvent` in DndContext
2. `onDrop` → calls `handleEventDrop(date, hour, minute)`
3. Duration preserved, optional confirmation dialog

## Resizing

Uses `re-resizable`. Scale: 96px = 1 hour.

## Key Helpers (`helpers.ts`)

- `groupEvents()` - Non-overlapping column assignment
- `groupEventsByDay()` - Date → events map
- `calculateMonthEventPositions()` - Stack positions (0-2)
- `isMultiDayEvent()` - Spans midnight check

## State (CalendarContext)

```typescript
{
  selectedDate: Date,
  view: 'day' | 'week' | 'month' | 'year' | 'agenda',
  filteredEvents: IEvent[],
  selectedColors: string[],
  use24HourFormat: boolean
}
```

Settings persisted to localStorage (`calendar-settings` key).
