import type { Category } from '@studyverse/proto'

type FilterCategory = Pick<Category, 'id' | 'name' | 'color'>

export function buildCategoryOptions(catalog: readonly FilterCategory[], eventCategories: readonly FilterCategory[]) {
  const categories = new Map<number, FilterCategory>()
  for (const category of [...eventCategories, ...catalog]) {
    if (category.id > 0 && category.name.trim()) categories.set(category.id, category)
  }
  return [...categories.values()]
    .map(({ id, name, color }) => ({ value: id, label: name, color: color || '#6B7280' }))
    .sort((left, right) => left.label.localeCompare(right.label))
}
