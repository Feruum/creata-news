import { context, metrics } from '@opentelemetry/api'
import { logs, SeverityNumber } from '@opentelemetry/api-logs'
import { ZoneContextManager } from '@opentelemetry/context-zone'
import { OTLPLogExporter } from '@opentelemetry/exporter-logs-otlp-http'
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-http'
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http'
import { registerInstrumentations } from '@opentelemetry/instrumentation'
import { DocumentLoadInstrumentation } from '@opentelemetry/instrumentation-document-load'
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch'
import { resourceFromAttributes } from '@opentelemetry/resources'
import { BatchLogRecordProcessor, LoggerProvider } from '@opentelemetry/sdk-logs'
import { AggregationType, MeterProvider, PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics'
import { AlwaysOnSampler, BatchSpanProcessor, WebTracerProvider } from '@opentelemetry/sdk-trace-web'
import { type Metric, onCLS, onINP, onLCP, onTTFB } from 'web-vitals'
import {
  type ProductEvent,
  type ProductEventAttributes,
  safeProductAttributes,
  sanitizeErrorStack
} from './telemetry-events'
import { SanitizingSpanExporter } from './telemetry-exporter'

export interface BrowserTelemetryConfig {
  tracesEndpoint: string
  apiUrls: string[]
  serviceName: string
  serviceVersion: string
  environment: string
  app: 'map' | 'events-admin'
}

export function startBrowserTelemetry(config: BrowserTelemetryConfig) {
  const endpoint = new URL(config.tracesEndpoint, window.location.origin)
  const signalEndpoint = (signal: string) => {
    const url = new URL(endpoint)
    url.pathname = url.pathname.replace(/\/traces$/, `/${signal}`)
    return url.href
  }
  const resource = resourceFromAttributes({
    'service.name': config.serviceName,
    'service.version': config.serviceVersion,
    'deployment.environment': config.environment,
    'deployment.environment.name': config.environment,
    app: config.app
  })
  const tracerProvider = new WebTracerProvider({
    resource,
    sampler: new AlwaysOnSampler(),
    spanProcessors: [
      new BatchSpanProcessor(
        new SanitizingSpanExporter(
          new OTLPTraceExporter({
            url: endpoint.href,
            timeoutMillis: 3000
          })
        ),
        { maxQueueSize: 256, maxExportBatchSize: 64, scheduledDelayMillis: 2000 }
      )
    ]
  })
  tracerProvider.register({ contextManager: new ZoneContextManager() })
  const meterProvider = new MeterProvider({
    resource,
    readers: [
      new PeriodicExportingMetricReader({
        exporter: new OTLPMetricExporter({ url: signalEndpoint('metrics'), timeoutMillis: 3000 }),
        exportIntervalMillis: 15000,
        exportTimeoutMillis: 4000
      })
    ],
    views: [
      {
        instrumentName: 'studyverse.web_vital',
        aggregation: {
          type: AggregationType.EXPLICIT_BUCKET_HISTOGRAM,
          options: {
            boundaries: [0, 0.1, 0.25, 0.5, 10, 100, 200, 500, 1000, 2500, 4000, 8000]
          }
        }
      }
    ]
  })
  metrics.setGlobalMeterProvider(meterProvider)
  const loggerProvider = new LoggerProvider({
    resource,
    processors: [
      new BatchLogRecordProcessor(
        new OTLPLogExporter({
          url: signalEndpoint('logs'),
          timeoutMillis: 3000
        }),
        { maxQueueSize: 128, maxExportBatchSize: 32, scheduledDelayMillis: 2000 }
      )
    ]
  })
  logs.setGlobalLoggerProvider(loggerProvider)
  const meter = meterProvider.getMeter('studyverse.browser', config.serviceVersion)
  const events = meter.createCounter('studyverse.ui.events', { description: 'Aggregate UI actions without user input' })
  const vital = meter.createHistogram('studyverse.web_vital', {
    description: 'Web vital values; CLS is a ratio, other values are ms'
  })
  const recordVital = (metric: Metric) => {
    if (Number.isFinite(metric.value))
      vital.record(metric.value, {
        vital: metric.name,
        'measurement.unit': metric.name === 'CLS' ? 'ratio' : 'ms'
      })
  }
  onCLS(recordVital)
  onINP(recordVital)
  onLCP(recordVital)
  onTTFB(recordVital)
  const logger = loggerProvider.getLogger('studyverse.browser', config.serviceVersion)
  const reportError = (value: unknown, kind: string) => {
    const error = value instanceof Error ? value : undefined
    logger.emit({
      context: context.active(),
      severityNumber: SeverityNumber.ERROR,
      severityText: 'ERROR',
      body: kind,
      attributes: {
        'exception.type': error && /^[A-Za-z]+Error$/.test(error.name) ? error.name : 'Error',
        'exception.stacktrace': sanitizeErrorStack(error?.stack, error?.message)
      }
    })
  }
  window.addEventListener('error', (event) => reportError(event.error, 'Unhandled browser error'))
  window.addEventListener('unhandledrejection', (event) => reportError(event.reason, 'Unhandled promise rejection'))
  const flush = () => {
    if (navigator.onLine)
      void Promise.allSettled([tracerProvider.forceFlush(), meterProvider.forceFlush(), loggerProvider.forceFlush()])
  }
  window.addEventListener('pagehide', flush)
  window.addEventListener('online', flush)
  const urls = config.apiUrls.map((url) => {
    const absolute = new URL(url, window.location.origin).href.replace(/\/$/, '')
    return new RegExp(`^${absolute.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?:/|$|\\?)`)
  })
  registerInstrumentations({
    instrumentations: [
      new FetchInstrumentation({
        propagateTraceHeaderCorsUrls: urls,
        ignoreUrls: [
          /\/telemetry\/v1\//,
          /\/api\/auth\//,
          /\/self-service\//,
          /\/admin\/hermes\/(?:info|access|settings)(?:$|\?)/
        ],
        clearTimingResources: true
      }),
      new DocumentLoadInstrumentation()
    ]
  })
  return (event: ProductEvent, attributes: ProductEventAttributes) => {
    events.add(1, { event, ...safeProductAttributes(attributes) })
  }
}
