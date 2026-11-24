import { cpSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { buildDemoAdmins } from './build-demo-admins'

const mediaSource = resolve(import.meta.dirname, '../backend/creata/content/assets')

export function copyDemoMedia(publicDir: string) {
  cpSync(mediaSource, join(publicDir, 'demo-media'), { recursive: true })
}

export async function prepareDemoAssets(publicDir: string) {
  copyDemoMedia(publicDir)
  await buildDemoAdmins(publicDir)
}
