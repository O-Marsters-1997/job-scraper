# ADR 0014 — OpenTelemetry tracing for local dev

## Context

ADR 0013 rejected W3C `traceparent` over AMQP and HTTP and deferred traces (ADR 0010). Logs
show what happened, not where time went, and a request that crosses api → RabbitMQ → worker →
outbound fetches can't be seen as one timed flow.

## Decision

Both binaries emit OTLP HTTP spans through `telemetry.InitTracing`. It is configured only by
the standard `OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_SERVICE_NAME`. With no endpoint every span
is a no-op, so prod and CI are unchanged.

- **Spans:** `otelhttp.NewMiddleware` roots each api request; `otelpgx` traces every pgx query;
  `logger.OutboundSpans` traces source and harvester fetches; `logger.Transport` traces and
  propagates on the worker's calls to our own api.
- **Propagation:** `queue.Publish` opens a producer span and injects W3C trace context into the
  AMQP headers; the consume loop extracts it and opens a consumer span around the handler.
  Third-party fetches record spans but never send trace headers.
- **Correlation:** the span context wins. `ctxHandler` writes `trace_id` and `span_id` from the
  span when one is active. `Middleware`'s UUID `trace_id` stays as the fallback when tracing
  is off, so ADR 0013 still holds with no collector.
- **Viewing:** `just tracing-up` runs `grafana/otel-lgtm`. Local dev stays fully local; prod
  keeps its Grafana Cloud logs and metrics path (ADR 0010).

## Consequences

- A trace ID from a log line pastes straight into Tempo.
- `run_id` still follows a Scrape Run across traces, since cron-started publishes root their own trace.
- Amends ADR 0013's rejection of `traceparent`.
