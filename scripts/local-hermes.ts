import { spawn, spawnSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { homedir } from 'node:os'
import { normalize, resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const venv = resolve(root, '.runtime/hermes-venv')
const windows = process.platform === 'win32'
const interpreter = resolve(venv, windows ? 'Scripts/python.exe' : 'bin/python')
const requirements = resolve(root, 'hermes-worker/requirements.txt')
const pythonEnv = { ...process.env, CREATA_POSTIZ_API_KEY: '' }
const version = readFileSync(requirements, 'utf8').match(/^Telethon==([^\s#]+)/m)?.[1]
if (!version) throw Error('Missing Telethon version in hermes-worker/requirements.txt')

function hasDependencies(python: string) {
  const probe = spawnSync(python, ['-I', '-c', `import telethon,sys; sys.exit(telethon.__version__ != ${JSON.stringify(version)})`], {
    cwd: root, env: pythonEnv, stdio: 'ignore', windowsHide: true,
  })
  return probe.status === 0
}

function execute(python: string, args: string[]) {
  const result = spawnSync(python, args, { cwd: root, env: pythonEnv, stdio: 'inherit', windowsHide: true })
  if (result.error) throw result.error
  if (result.status !== 0) throw Error(`Hermes Python setup failed (${result.signal || result.status}).`)
}

export function resolveHermesPython() {
  const configured = process.env.CREATA_PYTHON?.trim()
  if (configured) {
    const python = /[\\/]/.test(configured) ? resolve(root, configured) : configured
    if (hasDependencies(python)) return python
    // Only repair the project-owned environment. An explicit external environment belongs to its operator.
    const comparable = (path: string) => windows ? normalize(path).toLowerCase() : normalize(path)
    if (comparable(python) !== comparable(interpreter)) {
      throw Error('CREATA_PYTHON must run Python with hermes-worker/requirements.txt installed. Clear it to use the managed environment.')
    }
  } else if (existsSync(interpreter) && hasDependencies(interpreter)) {
    return interpreter
  }

  const stableRuntime = resolve(homedir(), '.cache/codex-runtimes/codex-primary-runtime/dependencies/python', windows ? 'python.exe' : 'bin/python3')
  const candidates = windows ? ['python', 'python3', 'py', stableRuntime] : ['python3', 'python', stableRuntime]
  let base: string | undefined
  for (const candidate of candidates) {
    const args = candidate === 'py' ? ['-3'] : []
    const result = spawnSync(candidate, [...args, '-I', '-c', 'import sys,venv; print(sys.executable)'], {
      cwd: root, env: pythonEnv, encoding: 'utf8', windowsHide: true,
    })
    if (result.status !== 0) continue
    const path = result.stdout.trim()
    if (!configured && hasDependencies(path)) return path
    base = path
    break
  }
  if (!base) throw Error('Install Python 3 with venv, or set CREATA_PYTHON to an interpreter with the Hermes requirements.')
  // Rebuild the launchers when a Python update removed their base interpreter, retaining installed packages.
  if (existsSync(resolve(venv, 'pyvenv.cfg'))) execute(base, ['-m', 'venv', '--upgrade', '--without-pip', venv])
  else execute(base, ['-m', 'venv', venv])
  if (!hasDependencies(interpreter)) {
    execute(interpreter, ['-m', 'pip', 'install', '-r', requirements])
    if (!hasDependencies(interpreter)) throw Error('The Hermes environment could not import the pinned Telethon version.')
  }
  return interpreter
}

export async function runHermesPython(args: string[]) {
  const child = spawn(resolveHermesPython(), args, { cwd: root, env: pythonEnv, stdio: 'inherit', windowsHide: true })
  return new Promise<number>((done, reject) => {
    child.once('error', reject)
    child.once('exit', code => done(code ?? 1))
  })
}

if (import.meta.main) process.exitCode = await runHermesPython(process.argv.slice(2))
