import { readFileSync } from 'node:fs'
import { expect, type Page, test } from '@playwright/test'

const assets = new Map([
  [
    '/hermes/',
    {
      contentType: 'text/html',
      body: readFileSync(new URL('../../../infra/hermes/panel/index.html', import.meta.url), 'utf8')
    }
  ],
  [
    '/hermes/panel.js',
    {
      contentType: 'application/javascript',
      body: readFileSync(new URL('../../../infra/hermes/panel/panel.js', import.meta.url), 'utf8')
    }
  ],
  [
    '/hermes/panel.css',
    {
      contentType: 'text/css',
      body: readFileSync(new URL('../../../infra/hermes/panel/panel.css', import.meta.url), 'utf8')
    }
  ]
])

type PublicInfo = Record<string, unknown>
type Reply = { status?: number; json: unknown }
type PanelFixture = {
  info: PublicInfo
  infoStatus: number
  calls: Record<string, unknown>[]
  paths: string[]
  unexpected: string[]
  errors: string[]
  settings: (body: Record<string, unknown>) => Reply | Promise<Reply>
}

declare global {
  interface Window {
    hermesPanelRequests: {
      path: string
      method: string
      credentials?: RequestCredentials
      cache?: RequestCache
      header: string | null
    }[]
    hermesPanelXSS?: boolean
  }
}

function ownerInfo(overrides: PublicInfo = {}): PublicInfo {
  return {
    claimed: true,
    telegram_linked: false,
    telegram_configured: false,
    gitlab_configured: false,
    chatgpt_connected: false,
    ready: false,
    dashboard_url: 'https://auth.webspace.cat/hermes/',
    recent_incidents: [],
    ...overrides
  }
}

async function panelFixture(
  page: Page,
  baseURL: string | undefined,
  overrides: PublicInfo = {}
): Promise<PanelFixture> {
  const origin = new URL(baseURL ?? '').origin
  const fixture: PanelFixture = {
    info: ownerInfo(overrides),
    infoStatus: 200,
    calls: [],
    paths: [],
    unexpected: [],
    errors: [],
    settings: () => ({ json: fixture.info })
  }
  page.on('pageerror', (error) => fixture.errors.push(error.message))
  await page.addInitScript(() => {
    window.hermesPanelRequests = []
    const nativeFetch = window.fetch
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      const path = new URL(url, window.location.origin).pathname
      if (path.startsWith('/hermes/api/')) {
        window.hermesPanelRequests.push({
          path,
          method: init?.method ?? 'GET',
          credentials: init?.credentials,
          cache: init?.cache,
          header: new Headers(init?.headers).get('X-Studyverse-Hermes')
        })
      }
      return nativeFetch(input, init)
    }
  })
  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    fixture.paths.push(url.pathname)
    if (url.origin !== origin) {
      fixture.unexpected.push(url.origin + url.pathname)
      return route.abort()
    }
    const asset = assets.get(url.pathname)
    if (asset) return route.fulfill({ ...asset, headers: { 'cache-control': 'no-store' } })
    if (url.pathname === '/hermes/api/info') {
      expect(request.method()).toBe('GET')
      return route.fulfill({ status: fixture.infoStatus, json: fixture.info, headers: { 'cache-control': 'no-store' } })
    }
    if (url.pathname === '/hermes/api/settings') {
      expect(request.method()).toBe('POST')
      expect(request.headers()['content-type']).toBe('application/json')
      const body: unknown = request.postDataJSON()
      if (typeof body !== 'object' || body === null || Array.isArray(body)) throw new Error('Expected settings object')
      const settings = body as Record<string, unknown>
      fixture.calls.push(settings)
      return route.fulfill({ ...(await fixture.settings(settings)), headers: { 'cache-control': 'no-store' } })
    }
    if (url.pathname === '/favicon.ico') return route.fulfill({ status: 204 })
    fixture.unexpected.push(url.pathname)
    return route.fulfill({ status: 404, body: '' })
  })
  await page.goto('/hermes/')
  await expect(page).toHaveTitle('Hermes · Studyverse')
  await expect(page.getByRole('heading', { name: 'Hermes', exact: true })).toBeVisible()
  await expect(page.locator('#settings')).toBeVisible()
  return fixture
}

async function assertPrivateBrowser(page: Page, fixture: PanelFixture) {
  expect(fixture.unexpected).toEqual([])
  expect(fixture.errors).toEqual([])
  expect(
    fixture.paths.every((path) =>
      [
        '/hermes/',
        '/hermes/panel.js',
        '/hermes/panel.css',
        '/hermes/api/info',
        '/hermes/api/settings',
        '/favicon.ico'
      ].includes(path)
    )
  ).toBe(true)
  const state = await page.evaluate(async () => ({
    local: localStorage.length,
    session: sessionStorage.length,
    databases: (await indexedDB.databases()).length,
    caches: (await caches.keys()).length,
    scripts: Array.from(document.scripts, (script) => new URL(script.src).pathname),
    requests: window.hermesPanelRequests
  }))
  expect(state).toMatchObject({ local: 0, session: 0, databases: 0, caches: 0, scripts: ['/hermes/panel.js'] })
  for (const request of state.requests)
    expect(request).toMatchObject({ credentials: 'include', cache: 'no-store', header: '1' })
}

test('claimed owner sees real panel assets without an automatic claim or external telemetry', async ({
  page,
  baseURL
}) => {
  const fixture = await panelFixture(page, baseURL, {
    chatgpt_connected: true,
    model: 'gpt-5.5',
    telegram_configured: true,
    telegram_linked: true,
    telegram_id: '987654321',
    gitlab_configured: true,
    ready: true
  })
  await expect(page.locator('#chatgpt-status')).toHaveText('Подключён')
  await expect(page.locator('#model')).toHaveText('Модель Hermes: gpt-5.5')
  await expect(page.locator('#telegram-id')).toHaveText('Разрешён Telegram ID: 987654321')
  await expect(page.locator('#ready')).toHaveText('Можно проверять')
  await expect(page.locator('[data-action="claim"]')).toHaveCount(0)
  expect(fixture.calls).toEqual([])
  await assertPrivateBrowser(page, fixture)
})

test('web console polls the same owner question and clears its answer when access is lost', async ({ page, baseURL }) => {
  const fixture = await panelFixture(page, baseURL)
  fixture.settings = (body) => ({
    json: {
      ...fixture.info,
      console: body.action === 'console'
        ? { status: 'queued', output: '' }
        : { status: 'complete', output: '<script>diagnostic evidence</script>' }
    }
  })
  await page.getByLabel('Команда или вопрос').fill('ask Как прошёл деплой?')
  await page.getByRole('button', { name: 'Выполнить' }).click()
  await expect(page.getByRole('log')).toHaveText('<script>diagnostic evidence</script>')
  expect(fixture.calls).toEqual([
    { action: 'console', command: 'ask Как прошёл деплой?', request_id: expect.any(String) },
    { action: 'console_result', request_id: fixture.calls[0].request_id }
  ])
  fixture.infoStatus = 403
  await page.getByRole('button', { name: 'Обновить' }).click()
  await expect(page.locator('#console-output')).toBeEmpty()
  await assertPrivateBrowser(page, fixture)
})

test('an ambiguous Linear create can be retried without creating a new request', async ({ page, baseURL }) => {
  const fixture = await panelFixture(page, baseURL, { linear_configured: true, linear_team: 'OPS' })
  fixture.settings = () => ({
    json: { ...fixture.info, console: fixture.calls.length === 1
      ? { status: 'retry', output: 'Проверь ту же задачу.' }
      : { status: 'complete', output: 'OPS-42' } }
  })
  await page.getByLabel('Команда или вопрос').fill('linear create Проверить деплой')
  const run = page.getByRole('button', { name: 'Выполнить' })
  await run.click()
  await expect(page.getByRole('log')).toHaveText('Проверь ту же задачу.')
  await run.click()
  await expect(page.getByRole('log')).toHaveText('OPS-42')
  expect(fixture.calls[0]).toEqual(fixture.calls[1])
  await assertPrivateBrowser(page, fixture)
})

for (const fields of [
  {
    form: '#telegram-form',
    input: '#bot-token',
    action: 'configure_telegram',
    field: 'bot_token',
    token: '123456789:synthetic_telegram_token_1234567890'
  },
  {
    form: '#gitlab-form',
    input: '#gitlab-token',
    action: 'configure_gitlab',
    field: 'gitlab_token',
    token: 'glpat-synthetic_private_read_only_token' // gitleaks:allow -- synthetic fixture
  }
]) {
  test(`${fields.action} sends a write-only password value and clears it before the response`, async ({
    page,
    baseURL
  }) => {
    const fixture = await panelFixture(page, baseURL)
    let release = (_reply: Reply) => {
      throw new Error('Response not waiting')
    }
    const pending = new Promise<Reply>((resolve) => {
      release = resolve
    })
    fixture.settings = () => pending
    const input = page.locator(fields.input)
    await expect(input).toHaveAttribute('type', 'password')
    await expect(input).toHaveAttribute('autocomplete', 'new-password')
    await input.fill(fields.token)
    await page.locator(fields.form).getByRole('button', { name: 'Сохранить токен', exact: true }).click()
    await expect.poll(() => fixture.calls.length).toBe(1)
    await expect(input).toHaveValue('')
    expect(fixture.calls).toEqual([{ action: fields.action, [fields.field]: fields.token }])
    await expect(page.locator(fields.form).getByRole('button')).toBeDisabled()
    release({ json: { ...fixture.info, [fields.field]: fields.token } })
    await expect(page.locator('#notice')).toHaveText('Сохранено.')
    await expect(input).toHaveValue('')
    expect(await page.content()).not.toContain(fields.token)
    await assertPrivateBrowser(page, fixture)
  })
}

test('official OpenAI device code flow posts only the flow id when checking approval', async ({ page, baseURL }) => {
  const fixture = await panelFixture(page, baseURL)
  const oauth = {
    flow_id: 'synthetic-flow-123',
    verification_url: 'https://auth.openai.com/codex/device',
    user_code: 'ABCD-EFGH',
    status: 'pending',
    expires_at: 4102444800
  }
  fixture.settings = (body) =>
    body.action === 'connect_chatgpt'
      ? { json: { ...fixture.info, oauth } }
      : {
          json: { ...fixture.info, chatgpt_connected: true, model: 'gpt-5.5', oauth: { ...oauth, status: 'approved' } }
        }
  await page.getByRole('button', { name: 'Подключить ChatGPT', exact: true }).click()
  await expect(page.locator('#user-code')).toHaveText('ABCD-EFGH')
  const login = page.getByRole('link', { name: 'Войти в OpenAI ↗', exact: true })
  await expect(login).toHaveAttribute('href', oauth.verification_url)
  await expect(login).toHaveAttribute('target', '_blank')
  await expect(login).toHaveAttribute('rel', 'noopener noreferrer')
  await page.getByRole('button', { name: 'Проверить подключение', exact: true }).click()
  await expect(page.locator('#chatgpt-status')).toHaveText('Подключён')
  await expect(page.locator('#oauth')).toBeHidden()
  await expect(page.locator('#user-code')).toBeEmpty()
  expect(fixture.calls).toEqual([{ action: 'connect_chatgpt' }, { action: 'poll_chatgpt', flow_id: oauth.flow_id }])
  await assertPrivateBrowser(page, fixture)
})

test('untrusted OAuth URLs are never presented as an OpenAI sign-in link', async ({ page, baseURL }) => {
  const fixture = await panelFixture(page, baseURL)
  for (const verification_url of [
    'https://auth.openai.com.evil.example/device',
    'http://auth.openai.com/device',
    'javascript:alert(1)'
  ]) {
    fixture.settings = () => ({
      json: {
        ...fixture.info,
        oauth: {
          status: 'pending',
          verification_url,
          user_code: 'ABCD-EFGH',
          flow_id: 'synthetic-flow-123'
        }
      }
    })
    await page.getByRole('button', { name: 'Подключить ChatGPT', exact: true }).click()
    await expect(page.locator('#notice')).toHaveText('Сохранено.')
    await expect(page.locator('#oauth')).toBeHidden()
    await expect(page.locator('#user-code')).toBeEmpty()
    await expect(page.locator('#verification-url')).toHaveAttribute('href', '#')
  }
  await assertPrivateBrowser(page, fixture)
})

test('Telegram binding and revocation accept only the server-resolved identity', async ({ page, baseURL }) => {
  const fixture = await panelFixture(page, baseURL, { telegram_configured: true })
  fixture.settings = (body) => ({
    json: {
      ...fixture.info,
      telegram_linked: body.action === 'bind_telegram',
      telegram_id: '987654321'
    }
  })
  await expect(page.locator('section[aria-labelledby="telegram-title"] input:not([type="password"])')).toHaveCount(0)
  await page.getByRole('button', { name: 'Использовать Telegram из профиля', exact: true }).click()
  await expect(page.locator('#telegram-id')).toHaveText('Разрешён Telegram ID: 987654321')
  await page.getByRole('button', { name: 'Отозвать доступ', exact: true }).click()
  await expect(page.locator('#telegram-id')).toHaveText('Telegram ещё не разрешён')
  expect(fixture.calls).toEqual([{ action: 'bind_telegram' }, { action: 'disconnect_telegram' }])
  await assertPrivateBrowser(page, fixture)
})

test('incident summary and diagnosis are rendered as text rather than executable markup', async ({ page, baseURL }) => {
  const summary = '<img src="/xss-probe" onerror="window.hermesPanelXSS=true">'
  const result = '<script>window.hermesPanelXSS=true</script><svg onload="window.hermesPanelXSS=true">'
  const fixture = await panelFixture(page, baseURL, {
    recent_incidents: [
      {
        id: 17,
        environment: 'production',
        service: 'backend',
        summary,
        result,
        raw_config: 'private-config'
      }
    ]
  })
  const history = page.locator('#history')
  await expect(history.locator('summary')).toHaveText(`#17 · production · backend — ${summary}`)
  await history.locator('summary').click()
  await expect(history.locator('pre')).toHaveText(result)
  await expect(history.locator('img, script, svg')).toHaveCount(0)
  expect(await page.evaluate(() => window.hermesPanelXSS)).toBeUndefined()
  expect(await page.content()).not.toContain('private-config')
  await assertPrivateBrowser(page, fixture)
})

test('403 revokes the visible panel and clears staged credentials, OAuth code and incident text', async ({
  page,
  baseURL
}) => {
  const fixture = await panelFixture(page, baseURL, {
    telegram_linked: true,
    telegram_id: '987654321',
    oauth: {
      status: 'pending',
      verification_url: 'https://auth.openai.com/codex/device',
      user_code: 'ABCD-EFGH',
      flow_id: 'synthetic-flow-123'
    },
    recent_incidents: [
      {
        id: 17,
        environment: 'production',
        service: 'backend',
        summary: 'Owner-only incident',
        result: 'Owner-only diagnosis'
      }
    ]
  })
  await page.locator('#bot-token').fill('123456789:synthetic_unsubmitted_token_12345')
  await page.locator('#gitlab-token').fill('glpat-synthetic_unsubmitted_token') // gitleaks:allow -- fixture
  fixture.infoStatus = 403
  fixture.info = { error: 'private-server-diagnostic' }
  await page.getByRole('button', { name: 'Обновить', exact: true }).click()
  await expect(page.locator('#notice')).toContainText('Нет доступа')
  await expect(page.locator('#settings')).toBeHidden()
  await expect(page.locator('#bot-token')).toHaveValue('')
  await expect(page.locator('#gitlab-token')).toHaveValue('')
  await expect(page.locator('#user-code')).toBeEmpty()
  await expect(page.locator('#history')).toBeEmpty()
  expect(await page.content()).not.toContain('private-server-diagnostic')
  await assertPrivateBrowser(page, fixture)
})

test('page departure clears password fields and the temporary device code', async ({ page, baseURL }) => {
  const fixture = await panelFixture(page, baseURL, {
    oauth: {
      status: 'pending',
      verification_url: 'https://auth.openai.com/codex/device',
      user_code: 'ABCD-EFGH',
      flow_id: 'synthetic-flow-123'
    },
    recent_incidents: [
      {
        id: 1,
        environment: 'production',
        service: 'backend',
        summary: 'Synthetic incident',
        result: 'Synthetic diagnosis'
      }
    ]
  })
  await page.locator('#bot-token').fill('123456789:synthetic_unsubmitted_token_12345')
  await page.locator('#gitlab-token').fill('glpat-synthetic_unsubmitted_token') // gitleaks:allow -- fixture
  await page.evaluate(() => window.dispatchEvent(new Event('pagehide')))
  await expect(page.locator('#bot-token')).toHaveValue('')
  await expect(page.locator('#gitlab-token')).toHaveValue('')
  await expect(page.locator('#user-code')).toBeEmpty()
  await expect(page.locator('#history')).toBeEmpty()
  await assertPrivateBrowser(page, fixture)
})

test('a late response cannot restore owner data after page departure', async ({ page, baseURL }) => {
  const fixture = await panelFixture(page, baseURL)
  let release = (_reply: Reply) => {
    throw new Error('Response not waiting')
  }
  fixture.settings = () =>
    new Promise<Reply>((resolve) => {
      release = resolve
    })
  await page.getByRole('button', { name: 'Подключить ChatGPT', exact: true }).click()
  await expect.poll(() => fixture.calls.length).toBe(1)
  await page.evaluate(() => window.dispatchEvent(new Event('pagehide')))
  release({
    json: {
      ...fixture.info,
      oauth: {
        status: 'pending',
        verification_url: 'https://auth.openai.com/codex/device',
        user_code: 'ABCD-EFGH',
        flow_id: 'synthetic-flow-123'
      },
      recent_incidents: [
        {
          id: 1,
          environment: 'production',
          service: 'backend',
          summary: 'Private incident',
          result: 'Private diagnosis'
        }
      ]
    }
  })
  await expect(page.locator('[data-action="connect_chatgpt"]')).toBeEnabled()
  await expect(page.locator('#settings')).toBeHidden()
  await expect(page.locator('#user-code')).toBeEmpty()
  await expect(page.locator('#history')).toBeEmpty()
  await assertPrivateBrowser(page, fixture)
})

test('mobile panel contains long incident output and keeps setup controls reachable', async ({ page, baseURL }) => {
  await page.setViewportSize({ width: 360, height: 800 })
  const fixture = await panelFixture(page, baseURL, {
    oauth: {
      status: 'pending',
      verification_url: 'https://auth.openai.com/codex/device',
      user_code: 'ABCD-EFGH',
      flow_id: 'synthetic-flow-123'
    },
    recent_incidents: [
      { id: 1, environment: 'production', service: 'backend', summary: 'X'.repeat(512), result: 'Y'.repeat(4096) }
    ]
  })
  await page.locator('#history summary').click()
  await expect(page.locator('#history pre')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.getByRole('button', { name: 'Подключить ChatGPT', exact: true }).scrollIntoViewIfNeeded()
  await expect(page.getByRole('button', { name: 'Подключить ChatGPT', exact: true })).toBeInViewport()
  await assertPrivateBrowser(page, fixture)
})
