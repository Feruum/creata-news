import { randomBytes } from 'node:crypto'
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { parseEnv } from 'node:util'
import { ensureLocalTestDatabase } from './local-database'

const root = resolve(import.meta.dirname, '..')
const path = resolve(root, '.env')
if (!existsSync(path)) {
  let content = readFileSync(resolve(root, '.env.example'), 'utf8')
  for (const key of ['CREATA_SESSION_SECRET', 'CREATA_WORKER_TOKEN', 'CREATA_BOT_SERVICE_TOKEN', 'CREATA_SSR_TOKEN', 'TELEGRAM_WEBHOOK_SECRET']) {
    content = content.replace(new RegExp(`^${key}=(?=\\r?$)`, 'm'), key + '=' + randomBytes(32).toString('hex'))
  }
  writeFileSync(path, content, { flag: 'wx', mode: 0o600 })
  console.log('Created private local .env with independent service secrets.')
}

if (process.argv.includes('--test-db')) {
  const content = readFileSync(path, 'utf8')
  const local = parseEnv(content)
  const testUrl = await ensureLocalTestDatabase(process.env.DATABASE_URL || local.DATABASE_URL, process.env.CREATA_TEST_DATABASE_URL || local.CREATA_TEST_DATABASE_URL)
  if (!local.CREATA_TEST_DATABASE_URL) {
    const expression = /^CREATA_TEST_DATABASE_URL=[^\r\n]*/m
    const line = 'CREATA_TEST_DATABASE_URL=' + testUrl
    const updated = expression.test(content) ? content.replace(expression, () => line) : content.trimEnd() + '\n' + line + '\n'
    writeFileSync(path, updated, { mode: 0o600 })
  }
  console.log('Isolated creata_test database is ready. bun run test:pipeline reads its URL from .env.')
} else {
  console.log('Local configuration is ready.')
}
