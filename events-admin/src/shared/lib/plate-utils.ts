import type { Value } from 'platejs'

interface PlateNode {
  type?: string
  text?: string
  fallback?: string
  children?: PlateNode[]
}

export function plateJsonToText(value: Value): string {
  const parts: string[] = []

  function walk(nodes: PlateNode[]) {
    for (const node of nodes) {
      if (typeof node.text === 'string') {
        parts.push(node.text)
      } else if (node.type === 'tg_emoji' && node.fallback) {
        parts.push(node.fallback)
      } else if (node.children) {
        walk(node.children)
        if (node.type && node.type !== 'li' && node.type !== 'lic') {
          parts.push('\n')
        }
      }
    }
  }

  walk(value as unknown as PlateNode[])
  return parts.join('').trim()
}

export function getPlateTextLength(value: Value): number {
  return plateJsonToText(value).length
}

export function isPlateValueEmpty(value: Value): boolean {
  return getPlateTextLength(value) === 0
}

export const EMPTY_PLATE_VALUE: Value = [{ type: 'p', children: [{ text: '' }] }]

export function parsePlateJson(jsonString: string | null | undefined): Value {
  if (!jsonString) return EMPTY_PLATE_VALUE

  try {
    const parsed = JSON.parse(jsonString)
    if (typeof parsed === 'string') return [{ type: 'p', children: [{ text: parsed }] }]
    if (Array.isArray(parsed) && parsed.length > 0) {
      return parsed as Value
    }
    return EMPTY_PLATE_VALUE
  } catch {
    return [{ type: 'p', children: [{ text: jsonString }] }]
  }
}
