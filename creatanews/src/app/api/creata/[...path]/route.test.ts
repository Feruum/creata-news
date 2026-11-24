import { expect, spyOn, test } from 'bun:test'
import { NextRequest } from 'next/server'
import { GET, POST } from './route'

const url = 'http://localhost/api/creata/v1/public/events'
const publicEvents = { params: Promise.resolve({ path: ['v1', 'public', 'events'] }) }

test('the public proxy caps streamed request bytes before contacting the API', async () => {
  const upstream = spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{}'))
  try {
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new Uint8Array(12 * 1024 * 1024))
        controller.enqueue(new Uint8Array(1))
        controller.close()
      },
    })
    const options = { method: 'POST', body: stream, duplex: 'half' } as const
    const request = new NextRequest(url, options)
    expect((await POST(request, publicEvents)).status).toBe(413)
    expect(upstream).not.toHaveBeenCalled()
  } finally {
    upstream.mockRestore()
  }
})

test('the public proxy forwards ordinary bodies without the private SSR header', async () => {
  const upstream = spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{}'))
  try {
    const request = new NextRequest(url, {
      method: 'POST',
      body: '{"type":"view"}',
      headers: { 'Content-Type': 'application/json', 'X-Creata-SSR-Token': 'browser-supplied' },
    })
    expect((await POST(request, publicEvents)).status).toBe(200)
    const options = upstream.mock.calls[0]?.[1]
    expect(new Headers(options?.headers).has('X-Creata-SSR-Token')).toBe(false)
    const raw = options?.body as Uint8Array
    expect(new TextDecoder().decode(raw)).toBe('{"type":"view"}')
  } finally {
    upstream.mockRestore()
  }
})

test('the website does not proxy internal service routes', async () => {
  const upstream = spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{}'))
  try {
    const path = ['v1', 'internal', 'ai-config']
    const response = await GET(new NextRequest('http://localhost/api/creata/' + path.join('/')), {
      params: Promise.resolve({ path }),
    })
    expect(response.status).toBe(404)
    expect(upstream).not.toHaveBeenCalled()
  } finally {
    upstream.mockRestore()
  }
})
