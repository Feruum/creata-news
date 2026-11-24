import { resolve } from 'node:path'

if (process.env.NEXT_PUBLIC_CREATA_DEMO_MODE === '1') {
  const { prepareDemoAssets } = await import('../../scripts/prepare-demo-assets')
  await prepareDemoAssets(resolve(import.meta.dirname, '../public'))
}
