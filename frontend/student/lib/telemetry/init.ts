import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { DocumentLoadInstrumentation } from '@opentelemetry/instrumentation-document-load';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { Resource } from '@opentelemetry/resources';
import {
  SEMRESATTRS_SERVICE_NAME,
  SEMRESATTRS_SERVICE_VERSION,
} from '@opentelemetry/semantic-conventions';

let isInitialized = false;

export function initTelemetry() {
  if (isInitialized || typeof window === 'undefined') {
    return;
  }

  const resource = new Resource({
    [SEMRESATTRS_SERVICE_NAME]: process.env.NEXT_PUBLIC_OTEL_SERVICE_NAME || 'student-portal-web',
    [SEMRESATTRS_SERVICE_VERSION]: '1.0.0',
  });

  const provider = new WebTracerProvider({
    resource,
  });

  // Configure OTLP exporter
  const exporter = new OTLPTraceExporter({
    url: process.env.NEXT_PUBLIC_OTEL_ENDPOINT || 'http://localhost:4318/v1/traces',
  });

  provider.addSpanProcessor(new BatchSpanProcessor(exporter));

  provider.register();

  // Register instrumentations
  registerInstrumentations({
    instrumentations: [
      new FetchInstrumentation({
        propagateTraceHeaderCorsUrls: [
          new RegExp(process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'),
        ],
        clearTimingResources: true,
      }),
      new DocumentLoadInstrumentation(),
    ],
  });

  isInitialized = true;
}

export function getTracer() {
  if (typeof window === 'undefined') {
    return null;
  }

  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { trace } = require('@opentelemetry/api');
  return trace.getTracer(
    process.env.NEXT_PUBLIC_OTEL_SERVICE_NAME || 'student-portal-web',
    '1.0.0'
  );
}
