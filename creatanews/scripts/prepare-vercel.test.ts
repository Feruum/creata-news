import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { test } from 'node:test'

test('standalone Sunday artifact includes forms runtime, styles and local proto dependency', async () => {
  const root = resolve(import.meta.dir, '../../..')
  execFileSync(process.execPath, [resolve(import.meta.dir, 'prepare-vercel.ts')], { cwd: root, stdio: 'pipe' })
  const output = resolve(root, '.artifacts/sundayopen-vercel')
  const manifest = JSON.parse(await readFile(resolve(output, 'package.json'), 'utf8'))
  assert.equal(manifest.dependencies['@studyverse/registration-forms'], 'file:./vendor/registration-forms')
  const forms = JSON.parse(await readFile(resolve(output, 'vendor/registration-forms/package.json'), 'utf8'))
  assert.equal(forms.dependencies['@studyverse/proto'], 'file:../proto')
  assert.ok(!JSON.stringify(forms).includes('workspace:'), 'standalone dependencies cannot require a workspace')
  assert.match(await readFile(resolve(output, 'vendor/registration-forms/src/styles.css'), 'utf8'), /\.registration-forms|\.rf-/)
  assert.match(await readFile(resolve(output, 'vendor/registration-forms/dist/index.js'), 'utf8'), /registration-workspace/)
})
