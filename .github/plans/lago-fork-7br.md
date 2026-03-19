# lago-fork-7br: Core Job Handlers — Invoices, Events, Payments, Webhooks

## Overview
Port Rails' core background job handlers into the Go Asynq runtime. Each handler must be idempotent, observable (structured logging), and expose dead-letter behavior via `asynq.SkipRetry`.

## Architecture

```mermaid
graph TD
    Scheduler -->|cron| FinalizeInvoicesBatch
    Scheduler -->|cron| MarkOverdueBatch
    Scheduler -->|cron| ValidateEventsBatch
    API -->|on event write| EnqueueValidateEvent
    InvoiceService -->|on finalize| EnqueueSendWebhook
    FinalizeInvoicesBatch -->|fan-out| FinalizeInvoiceHandler
    MarkOverdueBatch -->|fan-out| MarkOverdueHandler
    FinalizeInvoiceHandler -->|calls| InvoiceService.Finalize
    WebhookHandler -->|HTTP POST| ExternalURL
    WebhookHandler -->|record| webhooks table
```

## Task Types

| Asynq Task Type                | Queue    | Max Retry | Unique | Description |
|-------------------------------|----------|-----------|--------|-------------|
| `invoice:finalize`            | critical | 5         | Yes    | Finalize one draft invoice by ID |
| `invoice:mark_payment_overdue`| default  | 3         | Yes    | Scan + mark overdue invoices (batch clock job) |
| `events:validate`             | default  | 3         | Yes    | Post-ingestion event validation stub |
| `payments:create`             | critical | 5         | Yes    | Create payment for invoice (stub) |
| `webhooks:send_http`          | default  | 10        | No     | HTTP POST signed outbound webhook with retries |

## Dead-Letter Behavior
- On handler failure: Asynq retries with exponential backoff (default delay func)
- On `asynq.SkipRetry` error return: task moves directly to dead-letter queue (no more retries)
- Dead-letter queue observable via Asynq inspector (Redis-backed)
- All handlers log `slog.Error` on permanent failure before returning `SkipRetry`

## Tasks
1. ✅ DB migration 000008_webhooks (webhook_endpoints + webhooks tables)
2. ✅ Webhook + WebhookEndpoint GORM models
3. ✅ invoice_handlers.go — FinalizeInvoice + MarkPaymentOverdue handlers
4. ✅ event_handlers.go — ValidateEvents handler (stub)
5. ✅ payment_handlers.go — CreatePayment handler (stub)
6. ✅ webhook_handlers.go — SendHttpWebhook with HMAC-SHA256 signing
7. ✅ Task type constants + Enqueue helpers in runtime.go
8. ✅ Register all handlers in NewDefaultServeMux
9. ✅ Register cron schedules in RegisterDefaultSchedules
10. ✅ Handler tests (invoice, webhook)
11. ✅ All tests pass

## Implementation Summary (populated after completion)
