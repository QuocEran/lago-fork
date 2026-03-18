# `lago-fork` Repository Summary (English Learning Version)

## 1) What this repository is

This repository is a fork of **Lago**, an open-source platform for **usage-based billing** with subscription support.

At a high level, Lago helps you:

- ingest usage events,
- enrich them with business context (plans, metrics, filters, subscriptions),
- transform that data into billable outputs (invoices, charges, webhooks, integrations).

If your product can emit events, Lago is designed to turn those events into pricing and billing logic.

---

## 2) What the product can do

Based on the project documentation, major capabilities include:

- Usage metering.
- Subscription and hybrid pricing plans.
- Coupons and add-ons.
- Automated invoicing.
- Prepaid credits.

Practical interpretation:

- Lago is a programmable billing layer for SaaS products that need flexible pricing models.

---

## 3) High-level architecture

The overall Lago architecture (from docs) includes:

- Main API service (Rails) for request handling.
- Sidekiq + Redis for asynchronous jobs.
- Clock process for scheduled recurring tasks.
- Dedicated workers/queues for billing, events, webhooks, PDFs, analytics, etc.

In this fork, the most visible backend code is the Go-based high-throughput event pipeline:

- `events-processor/` consumes raw events from Kafka/Redpanda.
- Enriches events using PostgreSQL data.
- Publishes processed outputs to downstream Kafka topics.

---

## 4) Important folder map in this local fork

### Main folders with immediately useful code/docs

- `events-processor/`: Go service for high-volume event post-processing.
- `connectors/`: ingest connector configs (HTTP/SQS -> Kafka).
- `docs/`: architecture, development environment, monitoring.
- `deploy/`: Docker Compose deployment templates (local/light/production).
- `docker/`: all-in-one image docs for test/staging usage.

### Current workspace caveat

- `api/` is empty in this workspace.
- `front/` is empty in this workspace.

Likely explanation:

- these may be submodules not checked out (or intentionally excluded in your local copy).

---

## 5) Go deep dive: `events-processor`

## Purpose

`events-processor` is responsible for post-processing high-throughput usage events:

- consume raw events,
- enrich them with billing domain data,
- publish enriched events,
- route failed events to dead-letter topics.

## End-to-end processing flow

1. Entry point: `events-processor/main.go`

- Initializes logger, tracing, and Sentry.
- Handles graceful shutdown signals (`SIGINT`, `SIGTERM`).
- Starts the processor via `processors.StartProcessingEvents(...)`.

2. Wiring and infra setup: `events-processor/processors/main_processor.go`

- Reads env vars for Kafka/Postgres/Redis config.
- Creates producers for:
  - enriched events,
  - enriched expanded events,
  - charged-in-advance events,
  - dead-letter events.
- Creates DB and Redis-backed stores.
- Starts a consumer group over `LAGO_KAFKA_RAW_EVENTS_TOPIC`.

3. Record processing: `events-processor/processors/events_processor/processor.go`

- Processes fetched Kafka records concurrently (`errgroup`).
- Unmarshals JSON event payloads.
- Runs enrichment + publishing flow.
- Commit/retry behavior:
  - For retryable errors and events newer than 12 hours, records may be left uncommitted so they can be reprocessed.
  - Otherwise, failed records are pushed to dead-letter and then committed.

4. Domain enrichment: `events-processor/processors/events_processor/enrichment_service.go`

- Builds enriched event model from raw event.
- Fetches billable metric and evaluates expressions when configured.
- Fetches subscription by external subscription id + event timestamp.
- Fetches flat filters and expands events by charge/filter matching.

5. Publishing logic: `events-processor/processors/events_processor/event_producer_service.go`

- Publishes to topic-specific producers.
- Falls back to dead-letter if publishing fails.

6. Follow-up side effects

- `subscription_refresh_service.go`: flags subscriptions for refresh.
- `cache_service.go`: expires cache entries related to charge/filter updates.

## Core Go technologies used

- Kafka client: `franz-go`.
- Database: `pgx`, `gorm`.
- Redis: `go-redis`.
- Observability: OpenTelemetry, Sentry, Datadog tracing libs.
- Concurrency patterns: goroutines, `errgroup`, partition-based consumption.

---

## 6) What `connectors/` is for

`connectors/` provides ingestion paths into the Kafka pipeline:

- `http.yml`: HTTP ingestion service -> Kafka topic.
- `sqs.yml`: SQS ingestion -> Kafka topic.

Event payload format follows Lago usage event schema (e.g., `external_subscription_id`, `transaction_id`, `code`, `timestamp`, `properties`).

---

## 7) How components fit together

Conceptual flow:

1. External systems emit usage events.
2. Connectors publish raw events to Kafka.
3. `events-processor` enriches and republishes events.
4. Core Lago backend (API/workers) consumes downstream data to drive billing operations.
5. Billing-related outputs trigger invoices, webhooks, and integrations.

This separation is useful because event throughput and billing orchestration can scale independently.

---

## 8) Quick local run and test notes

## Dev stack (from docs)

- Create alias:
  - `lago="docker compose -f $LAGO_PATH/docker-compose.dev.yml"`
- Start dependencies:
  - `lago up -d --wait db redis traefik mailhog clickhouse webhook`
- Start main services:
  - `lago up front api api-worker api-clock`

## Go service run/test

- Start service:
  - `lago up -d events-processor`
- Run tests:
  - `lago exec events-processor go test ./...`

Important note from repo guidance:

- Direct host `go build` / `go test` may fail due to CGO dependencies.
- Prefer containerized commands via `lago exec`.

---

## 9) Key environment variables to understand first

For `events-processor`, prioritize learning these:

- `DATABASE_URL`
- `LAGO_KAFKA_BOOTSTRAP_SERVERS`
- `LAGO_KAFKA_RAW_EVENTS_TOPIC`
- `LAGO_KAFKA_ENRICHED_EVENTS_TOPIC`
- `LAGO_KAFKA_ENRICHED_EVENTS_EXPANDED_TOPIC`
- `LAGO_KAFKA_EVENTS_CHARGED_IN_ADVANCE_TOPIC`
- `LAGO_KAFKA_EVENTS_DEAD_LETTER_TOPIC`
- `LAGO_KAFKA_CONSUMER_GROUP`
- `LAGO_REDIS_STORE_URL`
- `LAGO_REDIS_CACHE_URL`

Why this matters:

- Most startup failures in event-driven services come from topic naming, broker auth/TLS mismatch, or missing DB/Redis connectivity.

---

## 10) Suggested learning path (Go-focused)

## Stage 1: Understand startup and control flow

1. Read `events-processor/main.go`.
2. Read `events-processor/processors/main_processor.go`.
3. Read `events-processor/processors/events_processor/processor.go`.

Goal:

- Understand dependency wiring, consumer lifecycle, and commit/retry/dead-letter behavior.

## Stage 2: Understand billing domain enrichment

1. Read `events-processor/processors/events_processor/enrichment_service.go`.
2. Read models in `events-processor/models/`:
   - `event.go`
   - `billable_metrics.go`
   - `subscriptions.go`
   - `flat_filters.go`

Goal:

- Understand how one incoming event can become multiple enriched outputs.

## Stage 3: Understand reliability and performance design

1. Read `events-processor/config/kafka/consumer.go`.
2. Read tests in:
   - `events-processor/processors/events_processor/*_test.go`
   - `events-processor/config/kafka/*_test.go`

Goal:

- Understand partition handling, rebalance safety, and delivery guarantees tradeoffs.

## Stage 4: Hands-on exercises

1. Add one custom metric or trace attribute in processing flow.
2. Add one test for a retryable failure path.
3. Experiment with producer key strategy and observe partition impact.

---

## 11) Common pitfalls (for learners)

- Confusing at-least-once processing with exactly-once behavior.
- Forgetting that commit strategy determines whether failed records are re-read.
- Not validating event schema before enrichment.
- Ignoring dead-letter topic monitoring.
- Running Go tests outside container when CGO requirements exist.

---

## 12) Fast cheat sheet

- Product: open-source usage-based billing API.
- Main architecture: API + async workers + scheduler + event pipeline.
- Go service in this fork: `events-processor`.
- Input: raw Kafka events.
- Outputs:
  - enriched events,
  - enriched expanded events,
  - charged-in-advance events,
  - dead-letter events.
- Data dependencies:
  - PostgreSQL for billing domain lookups,
  - Redis for flags/cache support.

---

## 13) Bottom line

If your goal is to learn Go from real production-style code, this fork is strong for:

- event-driven architecture,
- Kafka consumer group patterns,
- retry/commit/dead-letter design,
- domain enrichment pipelines,
- observability integration.

The best entry point is still `events-processor/`: focused code, explicit flow, and practical testing surface.
