export function matchesCommand(query: string, ...keywords: string[]): boolean {
  const search = query.trim().toLocaleLowerCase()
  return keywords.some((keyword) => keyword.toLocaleLowerCase().includes(search))
}
