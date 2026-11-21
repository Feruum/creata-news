import { SQL } from 'bun'
import { spawnSync } from 'node:child_process'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')

function isProjectPostgres(url: URL) {
  return ['postgres:', 'postgresql:'].includes(url.protocol)
    && ['127.0.0.1', 'localhost'].includes(url.hostname)
    && url.port === (process.env.CREATA_DATABASE_PORT || '54328')
    && url.pathname === '/creata'
}

async function ping(url: URL) {
  const db = new SQL(url.href, { max: 1, connectionTimeout: 3 })
  try {
    await db`SELECT 1`
  } finally {
    await db.close()
  }
}

export async function ensureLocalDatabase(dsn: string | undefined) {
  if (!dsn) throw Error('DATABASE_URL is required. Run bun run setup:local.')
  const url = new URL(dsn)
  try {
    await ping(url)
  } catch (error) {
    const code = (error as { code?: string }).code
    if (code !== 'ERR_POSTGRES_CONNECTION_REFUSED' || !isProjectPostgres(url)) {
      throw Error(`PostgreSQL connection failed (${code || 'connection error'}). Check DATABASE_URL and database availability.`)
    }
    console.log('Starting the existing project PostgreSQL container...')
    const result = spawnSync('docker', ['compose', 'start', '--wait', 'postgres'], {
      cwd: root, stdio: 'inherit', windowsHide: true,
    })
    if (result.status !== 0) throw Error('Start Docker and check docker compose ps -a. On a fresh installation, initialize PostgreSQL with docker compose up -d --wait postgres.')
    await ping(url)
  }
  return url
}

export async function ensureLocalTestDatabase(dsn: string | undefined, configuredTestDsn?: string) {
  if (!dsn || !isProjectPostgres(new URL(dsn))) {
    throw Error('Automatic creata_test setup requires the project Compose database. For other databases, separately provision a test database and set CREATA_TEST_DATABASE_URL.')
  }
  const test = new URL(dsn)
  test.pathname = '/creata_test'
  if (configuredTestDsn && new URL(configuredTestDsn).href !== test.href) {
    throw Error('CREATA_TEST_DATABASE_URL already points elsewhere. Existing configuration was preserved.')
  }
  await ensureLocalDatabase(dsn)
  const db = new SQL(dsn, { max: 1, connectionTimeout: 3 })
  try {
    const found = await db`SELECT 1 FROM pg_database WHERE datname = 'creata_test'`
    if (!found.length) {
      try {
        await db`CREATE DATABASE creata_test`
      } catch (error) {
        // A simultaneous setup may have completed creation after the catalog check.
        if (!['42P04', '23505'].includes((error as { code?: string }).code || '')) throw error
      }
    }
  } finally {
    await db.close()
  }
  await ping(test)
  return test.href
}
