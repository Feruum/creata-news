import { expect, spyOn, test } from 'bun:test'
import { serverApi } from './api'

test('the public catalog revalidates cached bytes and drops a withdrawn article', async () => {
  const path = '/public/articles/validation-regression'
  const etag = 'W/"published"'
  const originalToken = process.env.CREATA_SSR_TOKEN
  process.env.CREATA_SSR_TOKEN = 'server-runtime-test-token'
  const upstream = spyOn(globalThis, 'fetch')
    .mockResolvedValueOnce(Response.json({ title: 'Published' }, { headers: { ETag: etag } }))
    .mockResolvedValueOnce(new Response(null, { status: 304 }))
    .mockResolvedValueOnce(new Response(null, { status: 404 }))
    .mockResolvedValueOnce(Response.json({ title: 'Republished' }))
  try {
    expect(await serverApi(path)).toEqual({ title: 'Published' })
    expect(await serverApi(path)).toEqual({ title: 'Published' })
    expect(new Headers(upstream.mock.calls[1]?.[1]?.headers).get('If-None-Match')).toBe(etag)
    expect(new Headers(upstream.mock.calls[1]?.[1]?.headers).get('X-Creata-SSR-Token')).toBe(process.env.CREATA_SSR_TOKEN)
    expect(upstream.mock.calls[1]?.[1]?.redirect).toBe('error')
    await expect(serverApi(path)).rejects.toMatchObject({ status: 404 })
    expect(await serverApi(path)).toEqual({ title: 'Republished' })
    expect(new Headers(upstream.mock.calls[3]?.[1]?.headers).has('If-None-Match')).toBe(false)
  } finally {
    upstream.mockRestore()
    if (originalToken === undefined) delete process.env.CREATA_SSR_TOKEN
    else process.env.CREATA_SSR_TOKEN = originalToken
  }
})
