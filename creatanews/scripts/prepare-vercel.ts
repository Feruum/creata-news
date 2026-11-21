import { cp, mkdir, readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const app = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const root = resolve(app, '../..')
const output = resolve(root, '.artifacts/sundayopen-vercel')
await mkdir(output, { recursive: true })

for (const path of ['src', 'public', 'content', 'tsconfig.json', 'next.config.ts', 'next-env.d.ts']) {
  await cp(resolve(app, path), resolve(output, path), {
    recursive: true,
    filter: (source) => !source.endsWith('.test.ts')
  })
}

const manifest = JSON.parse(await readFile(resolve(app, 'package.json'), 'utf8'))
// Match the locally verified versions instead of resolving newer caret releases during deployment.
for (const section of ['dependencies', 'devDependencies']) {
  for (const name of Object.keys(manifest[section])) {
    if (name.startsWith('@studyverse/')) continue
    const installed = JSON.parse(await readFile(resolve(app, 'node_modules', name, 'package.json'), 'utf8'))
    manifest[section][name] = installed.version
  }
}
for (const name of ['proto', 'env', 'registration-forms']) {
  const source = resolve(root, 'packages', name)
  const target = resolve(output, 'vendor', name)
  await mkdir(target, { recursive: true })
  await cp(resolve(source, 'dist'), resolve(target, 'dist'), { recursive: true })
  if (name === 'registration-forms') {
    await mkdir(resolve(target, 'src'), { recursive: true })
    await cp(resolve(source, 'src/styles.css'), resolve(target, 'src/styles.css'))
  }
  const dependency = JSON.parse(await readFile(resolve(source, 'package.json'), 'utf8'))
  delete dependency.scripts
  delete dependency.devDependencies
  for (const [dependencyName, version] of Object.entries(dependency.dependencies || {})) {
    if (typeof version === 'string' && version.startsWith('workspace:')) {
      if (!dependencyName.startsWith('@studyverse/')) throw new Error(`Unsupported workspace dependency: ${dependencyName}`)
      dependency.dependencies[dependencyName] = `file:../${dependencyName.slice('@studyverse/'.length)}`
    }
  }
  await writeFile(resolve(target, 'package.json'), `${JSON.stringify(dependency, null, 2)}\n`)
  manifest.dependencies[`@studyverse/${name}`] = `file:./vendor/${name}`
}
manifest.scripts = { build: 'next build', start: 'next start', dev: 'next dev --port 3002' }
delete manifest.devDependencies['@playwright/test']
await writeFile(resolve(output, 'package.json'), `${JSON.stringify(manifest, null, 2)}\n`)
await writeFile(
  resolve(output, 'vercel.json'),
  `${JSON.stringify({ framework: 'nextjs', installCommand: 'bun install --frozen-lockfile', buildCommand: 'bun run build' }, null, 2)}\n`
)
await writeFile(resolve(output, '.vercelignore'), 'node_modules\n.next\n.env*\n*.tsbuildinfo\n')
console.log(`Prepared Sunday Open only: ${output}`)
