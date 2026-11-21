import assert from 'node:assert/strict'
import { test } from 'node:test'
import { BasicTracerProvider, InMemorySpanExporter, SimpleSpanProcessor } from '@opentelemetry/sdk-trace-web'
import { SanitizingSpanExporter } from './telemetry-exporter'

test('exports an actual SDK span with intact trace identity and sanitized payload', async () => {
  const exporter = new InMemorySpanExporter()
  const original = new InMemorySpanExporter()
  const provider = new BasicTracerProvider({
    spanProcessors: [new SimpleSpanProcessor(original), new SimpleSpanProcessor(new SanitizingSpanExporter(exporter))]
  })
  const span = provider.getTracer('test').startSpan('GET https://map.webspace.cat/api?token=secret')
  span.setAttribute('http.url', 'https://map.webspace.cat/api?password=secret')
  span.setAttribute('http.status_code', 200)
  span.recordException(new Error('private form value'))
  const traceId = span.spanContext().traceId
  span.end()
  await provider.forceFlush()
  const [record] = exporter.getFinishedSpans()
  assert.equal(record.spanContext().traceId, traceId)
  assert.deepEqual(record.duration, original.getFinishedSpans()[0].duration)
  assert.equal(record.ended, true)
  assert.deepEqual(record.resource.attributes, original.getFinishedSpans()[0].resource.attributes)
  assert.equal(record.name, 'GET https://map.webspace.cat/api')
  assert.equal(record.attributes['http.url'], 'https://map.webspace.cat/api')
  assert.equal(record.attributes['http.status_code'], 200)
  assert.equal(JSON.stringify(record.events).includes('private form value'), false)
  await provider.shutdown()
})
