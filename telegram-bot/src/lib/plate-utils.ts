interface PlateNode {
  type?: string
  text?: string
  fallback?: string
  children?: PlateNode[]
}

/**
 * Extract plain text from a Plate JSON string.
 * Used by telegram bot to format notification messages.
 */
export function plateJsonToText(jsonString: string | null | undefined): string {
  if (!jsonString) return ''
  try {
    const nodes: PlateNode[] = JSON.parse(jsonString)
    if (!Array.isArray(nodes)) return ''
    const parts: string[] = []
    walkNodes(nodes, parts)
    return parts.join('').trim()
  } catch {
    return ''
  }
}

function walkNodes(nodes: PlateNode[], parts: string[]) {
  for (const node of nodes) {
    if (typeof node.text === 'string') {
      parts.push(node.text)
    } else if (node.type === 'tg_emoji' && node.fallback) {
      parts.push(node.fallback)
    } else if (node.children) {
      walkNodes(node.children, parts)
      if (node.type && node.type !== 'li' && node.type !== 'lic' && node.type !== 'ul' && node.type !== 'ol') {
        parts.push('\n')
      }
    }
  }
}
