import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'

/** Loopback stand-in for the documented Postiz Public API. */
export function startPostizFixture(publicSite: string) {
  const key = randomBytes(32).toString('hex')
  const channel = { id: 'qa-threads', name: 'Creata QA', identifier: 'threads', disabled: false }
  const posts: Array<{id: string; content: string; state: string; publishDate: string; integration: {id: string}}> = []
  const http = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    async fetch(request) {
      if (request.headers.get('Authorization') !== key) return new Response(null, {status: 401})
      const path = new URL(request.url).pathname
      if (path === '/public/v1/integrations' && request.method === 'GET') return Response.json([channel])
      if (path !== '/public/v1/posts') return new Response(null, {status: 404})
      if (request.method === 'GET') return Response.json({posts})
      if (request.method !== 'POST') return new Response(null, {status: 405})
      const body = await request.json()
      const [post] = body.posts || []
      const content = post?.value?.[0]?.content
      if (body.type !== 'draft' || body.posts?.length !== 1 || post?.integration?.id !== channel.id ||
          post?.settings?.__type !== channel.identifier || typeof content !== 'string' || !content.includes(publicSite + '/')) {
        return Response.json({error:'Invalid draft'}, {status: 400})
      }
      const id = 'fixture-' + (posts.length + 1)
      posts.push({id, content, state:'DRAFT', publishDate:new Date().toISOString(), integration:{id:channel.id}})
      return Response.json([{postId:id, integration:channel.id}])
    },
  })
  return {
    env: {
      CREATA_POSTIZ_API_URL: `http://127.0.0.1:${http.port}/public/v1`,
      CREATA_POSTIZ_API_KEY: key,
      CREATA_POSTIZ_MODE: 'draft',
      CREATA_TEST_POSTIZ_CHANNEL: channel.id,
    },
    verify() {
      assert.equal(posts.length, 2, 'Expected exactly one Hermes draft and one newsroom draft')
      assert.ok(posts.some(p => p.content.includes('Дереккөздерге негізделген мақала') && p.content.includes(publicSite + '/kk/articles/')))
      assert.ok(posts.some(p => /UI editorial \d+ EN\b/.test(p.content) && p.content.includes('Lead EN') && p.content.includes(publicSite + '/en/articles/')))
      console.log('Postiz HTTP fixture verified both localized drafts and their published article links.')
    },
    stop: () => http.stop(true),
  }
}
