import { expect, test } from 'bun:test'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { AdminContext } from './admin-context'
import { aiSettingsPatch } from './domain'
import { tr } from './i18n'
import { AISettingsPage } from './settings-pages'
import type { Locale, Settings } from './types'

const settings = {
  editorialInstructions: '', scoreThreshold: 80, digestHour: 18, timezone: 'Asia/Almaty',
  ai: {
    endpoint: 'https://example.org/v1', model: 'model', keyConfigured: true,
    dailyBudgetUsd: 1, spentTodayUsd: 0, inputPricePerMillion: 2, outputPricePerMillion: 3,
    exaKeyConfigured: true,
  },
} satisfies Settings

const serialize = (body: unknown) => JSON.parse(JSON.stringify(body))

test('AI settings save keeps an omitted or blank Exa key and excludes response-only fields', () => {
  const unchanged = {
    endpoint: 'https://example.org/v1', model: 'model', dailyBudgetUsd: 1,
    inputPricePerMillion: 2, outputPricePerMillion: 3,
  }
  expect(serialize(aiSettingsPatch(settings.ai, ''))).toEqual(unchanged)
  expect(serialize(aiSettingsPatch(settings.ai, ' ', { key: '  ', clear: false }))).toEqual(unchanged)
})

test('AI settings save can replace both keys or explicitly clear Exa', () => {
  expect(serialize(aiSettingsPatch(settings.ai, ' llm-fixture ', { key: ' exa-fixture ', clear: false })))
    .toMatchObject({ apiKey: 'llm-fixture', exaApiKey: 'exa-fixture' })
  expect(serialize(aiSettingsPatch(settings.ai, '', { key: 'exa-fixture', clear: true })))
    .toHaveProperty('exaApiKey', '')
})

function renderSettings(value: Settings, locale: Locale) {
  const client = new QueryClient()
  client.setQueryData(['settings'], value)
  return renderToStaticMarkup(createElement(QueryClientProvider, { client },
    createElement(AdminContext.Provider, { value: {
      mode: 'hermes', locale, t: key => tr(key, locale), setLocale: () => {},
      api: () => Promise.reject(new Error('Unexpected API request')),
      session: undefined, sessionLoading: false, sessionError: null, refreshSession: () => {}, toast: () => {},
    } }, createElement(AISettingsPage)),
  ))
}

test('Hermes exposes a localized, masked Exa input only outside demo mode', () => {
  for (const [locale, label, clearLabel] of [
    ['ru', 'API-ключ Exa', 'Удалить ключ Exa при сохранении'],
    ['kk', 'Exa API кілті', 'Сақтағанда Exa кілтін жою'],
    ['en', 'Exa API key', 'Clear the Exa key when saving'],
  ] as const) {
    const html = renderSettings(settings, locale)
    if (process.env.VITE_CREATA_DEMO_MODE === '1') {
      expect(html).not.toContain(label)
      expect(html.match(/type="password"/g)).toHaveLength(1)
    } else {
      expect(html).toContain(label)
      expect(html).toContain(clearLabel)
      expect(html.match(/<input[^>]+type="password"[^>]*\bvalue=""[^>]*>/g)).toHaveLength(2)
    }
  }
})

test('Hermes only offers clearing when an Exa key is configured', () => {
  const { exaKeyConfigured, ...legacyAI } = settings.ai
  expect(renderSettings({ ...settings, ai: legacyAI }, 'en')).not.toContain('Clear the Exa key when saving')
})
