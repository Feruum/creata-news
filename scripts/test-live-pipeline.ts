import { runHermesPython } from './local-hermes'

const env = process.env
const target = env.CREATA_TEST_API_ORIGIN && new URL(env.CREATA_TEST_API_ORIGIN)
const database = env.CREATA_TEST_DATABASE_URL && new URL(env.CREATA_TEST_DATABASE_URL)
if (!target || !['127.0.0.1', 'localhost', '[::1]'].includes(target.hostname)
  || env.CREATA_ENV !== 'test' || env.CREATA_ALLOW_PRIVATE_SOURCES !== '1'
  || target.origin !== env.CREATA_API_ORIGIN || !database || !database.pathname.includes('test')
  || env.DATABASE_URL !== env.CREATA_TEST_DATABASE_URL) {
  throw Error('Run bun run test:pipeline to start the disposable test API and database for the live Hermes checks.')
}

process.exitCode = await runHermesPython(['-m', 'unittest', 'discover', '-s', 'hermes-worker/tests', '-p', 'test_live_api.py', '-v'])
