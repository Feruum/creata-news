import { describe, expect, test } from 'bun:test'
import type { IncomingMessage, ServerResponse } from 'node:http'
import { Readable, Writable } from 'node:stream'
import { createTelemetryMiddleware } from './telemetry-proxy'

async function runRequest(method: string, url: string, body: Uint8Array, headers: Record<string, string> = {}) {
  const request = Object.assign(Readable.from([body]), { method, url, headers }) as IncomingMessage
  let responseBody = ''
  const responseHeaders = new Map<string, string>()
  const response = Object.assign(
    new Writable({
      write(chunk, _encoding, done) {
        responseBody += chunk.toString()
        done()
      }
    }),
    {
      statusCode: 200,
      setHeader(name: string, value: string) {
        responseHeaders.set(name, value)
      }
    }
  ) as unknown as ServerResponse
  let forwarded: Request | undefined
  const handler = createTelemetryMiddleware('http://collector:4318', async (request) => {
    forwarded = request
    return new Response('{}')
  })
  await handler(request, response, () => {})
  return { response, responseBody, responseHeaders, forwarded }
}

describe('Vite local OTLP gateway', () => {
  test('rejects methods, signals and oversized bodies before contacting collector', async () => {
    const get = await runRequest('GET', '/telemetry/v1/traces', new Uint8Array())
    expect(get.response.statusCode).toBe(405)
    const unknown = await runRequest('POST', '/telemetry/v1/events', new Uint8Array())
    expect(unknown.response.statusCode).toBe(404)
    const large = await runRequest('POST', '/telemetry/v1/logs', new Uint8Array(1048577), {
      'content-type': 'application/json'
    })
    expect(large.response.statusCode).toBe(413)
    expect(large.forwarded).toBeUndefined()
  })

  test('removes credentials and URL parameters from forwarded requests', async () => {
    const result = await runRequest('POST', '/telemetry/v1/metrics?private=secret', new TextEncoder().encode('{}'), {
      'content-type': 'application/json',
      cookie: 'session=secret',
      authorization: 'Bearer secret'
    })
    expect(result.forwarded?.url).toBe('http://collector:4318/v1/metrics')
    expect([...result.forwarded!.headers.keys()]).toEqual(['content-type'])
    expect(result.responseHeaders.get('Cache-Control')).toBe('no-store')
  })
})
