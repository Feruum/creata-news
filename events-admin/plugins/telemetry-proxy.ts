import type { IncomingMessage, ServerResponse } from 'node:http'
import type { Plugin } from 'vite'

const MAX_REQUEST_BYTES = 1024 * 1024

export function createTelemetryMiddleware(
  collectorUrl: string,
  exportRequest: (request: Request) => Promise<Response> = fetch
) {
  return async (request: IncomingMessage, response: ServerResponse, next: () => void): Promise<void> => {
    const pathname = new URL(request.url || '/', 'http://localhost').pathname
    if (!pathname.startsWith('/telemetry/')) return next()
    response.setHeader('Cache-Control', 'no-store')
    const reply = (status: number, message: string) => {
      response.statusCode = status
      response.end(message)
    }
    if (!/^\/telemetry\/v1\/(traces|metrics|logs)$/.test(pathname)) return reply(404, 'Unknown telemetry signal')
    if (request.method !== 'POST') return reply(405, 'Method not allowed')
    if (Number(request.headers['content-length']) > MAX_REQUEST_BYTES) return reply(413, 'Request too large')
    const contentType = request.headers['content-type']?.split(';')[0].trim().toLowerCase()
    if (contentType !== 'application/json' && contentType !== 'application/x-protobuf') {
      return reply(415, 'Unsupported telemetry format')
    }
    const chunks: Buffer[] = []
    let length = 0
    try {
      for await (const chunk of request) {
        const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk)
        length += bytes.byteLength
        if (length > MAX_REQUEST_BYTES) return reply(413, 'Request too large')
        chunks.push(bytes)
      }
    } catch {
      return reply(400, 'Invalid request')
    }
    const body = new ArrayBuffer(length)
    new Uint8Array(body).set(Buffer.concat(chunks))
    try {
      const upstream = await exportRequest(
        new Request(new URL(pathname.replace(/^\/telemetry/, ''), collectorUrl), {
          method: 'POST',
          headers: { 'content-type': contentType },
          body,
          signal: AbortSignal.timeout(3000)
        })
      )
      response.statusCode = upstream.status
      response.setHeader('Content-Type', upstream.headers.get('content-type') || 'application/json')
      response.end(Buffer.from(await upstream.arrayBuffer()))
    } catch {
      reply(502, 'Telemetry unavailable')
    }
  }
}

export function telemetryProxy(): Plugin {
  const handler = createTelemetryMiddleware(process.env.OTEL_COLLECTOR_HTTP_URL || 'http://localhost:4318')
  return {
    name: 'studyverse-telemetry-proxy',
    configureServer(server) {
      server.middlewares.use(handler)
    },
    configurePreviewServer(server) {
      server.middlewares.use(handler)
    }
  }
}
