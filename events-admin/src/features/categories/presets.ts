export const categoryPresets = [
  { name: 'Music', color: '#8B5CF6' },
  { name: 'Technology', color: '#3B82F6' },
  { name: 'Design', color: '#EC4899' },
  { name: 'Business', color: '#F59E0B' },
  { name: 'Science', color: '#06B6D4' },
  { name: 'Languages', color: '#14B8A6' },
  { name: 'Sports', color: '#22C55E' },
  { name: 'Games', color: '#6366F1' },
  { name: 'Arts & Culture', color: '#F97316' },
  { name: 'Books', color: '#84CC16' },
  { name: 'Volunteering', color: '#EF4444' },
  { name: 'Networking', color: '#64748B' }
] as const

export function existingCategoryNames(categories: ReadonlyArray<{ name: string }>) {
  return new Set(categories.map(({ name }) => name.trim().toLowerCase()))
}
