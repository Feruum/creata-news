import { spawn } from 'node:child_process'
import { copyFile, cp, mkdir, mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const apps = [
  { workspace: 'events-admin', base: 'editorial' },
  { workspace: 'agent-admin', base: 'hermes' },
]

function build(workspace: string, base: string, output: string) {
  const child = spawn(process.execPath, [
    'x', 'vite', 'build', '--configLoader', 'runner', '--base', `/${base}/`, '--outDir', output,
  ], {
    cwd: resolve(root, workspace),
    env: {
      ...process.env,
      VITE_CREATA_DEMO_MODE: '1',
      VITE_ADMIN_BASE_PATH: `/${base}`,
      VITE_WEB_URL: '',
      VITE_EDITORIAL_URL: '/editorial',
      VITE_HERMES_ADMIN_URL: '/hermes',
    },
    stdio: 'inherit',
    windowsHide: true,
  })
  return new Promise<void>((resolve, reject) => {
    child.on('error', reject)
    child.on('exit', (code, signal) => {
      if (code === 0) resolve()
      else reject(new Error(`${workspace} demo build failed (${signal ?? code})`))
    })
  })
}

export async function buildDemoAdmins(publicDir: string) {
  const temp = await mkdtemp(join(tmpdir(), 'creata-demo-admin-'))
  try {
    for (const { workspace, base } of apps) {
      await build(workspace, base, join(temp, base))
    }
    const rewrites = join(publicDir, 'demo-apps')
    await mkdir(rewrites, { recursive: true })
    for (const { base } of apps) {
      const output = join(temp, base)
      const destination = join(publicDir, base)
      await mkdir(destination, { recursive: true })
      await cp(join(output, 'assets'), join(destination, 'assets'), { recursive: true })
      await copyFile(join(output, 'creata-mark.svg'), join(destination, 'creata-mark.svg'))
      await copyFile(join(output, 'index.html'), join(destination, 'index.html'))
      await copyFile(join(output, 'index.html'), join(rewrites, `${base}.html`))
    }
  } finally {
    await rm(temp, { recursive: true, force: true })
  }
}

if (import.meta.main) {
  await buildDemoAdmins(resolve(root, 'creatanews/public'))
}
