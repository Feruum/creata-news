import { env } from '../lib/env'
import { generateMiniAppUrl } from '../lib/mini-app'

export function generateMapLink(roomId: string): string {
  return generateMiniAppUrl(env.MAP_APP_URL, { roomId })
}
