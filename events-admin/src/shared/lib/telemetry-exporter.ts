import type { ReadableSpan, SpanExporter } from '@opentelemetry/sdk-trace-web'
import { sanitizeSpanAttributes, sanitizeTelemetryUrl } from './telemetry-events'

export class SanitizingSpanExporter implements SpanExporter {
  private readonly exporter: SpanExporter

  constructor(exporter: SpanExporter) {
    this.exporter = exporter
  }

  export(spans: ReadableSpan[], callback: Parameters<SpanExporter['export']>[1]): void {
    const safe = spans.map(
      (span): ReadableSpan => ({
        ...span,
        spanContext: () => span.spanContext(),
        duration: span.duration,
        ended: span.ended,
        resource: span.resource,
        instrumentationScope: span.instrumentationScope,
        droppedAttributesCount: span.droppedAttributesCount,
        droppedEventsCount: span.droppedEventsCount,
        droppedLinksCount: span.droppedLinksCount,
        name: span.name.replace(/https?:\/\/[^\s)]+/g, sanitizeTelemetryUrl).split('?')[0],
        attributes: sanitizeSpanAttributes(span.attributes),
        status: { code: span.status.code },
        events: span.events.map((event) => ({
          ...event,
          attributes: sanitizeSpanAttributes(event.attributes ?? {})
        })),
        links: span.links.map((link) => ({ ...link, attributes: sanitizeSpanAttributes(link.attributes ?? {}) }))
      })
    )
    this.exporter.export(safe, callback)
  }

  shutdown(): Promise<void> {
    return this.exporter.shutdown()
  }

  forceFlush(): Promise<void> {
    return this.exporter.forceFlush?.() ?? Promise.resolve()
  }
}
