# api-go — Project Summary

## What It Is

`api-go` is the **Go-native backend service** of the [Lago](https://github.com/getlago/lago) open-source usage-based billing platform. It is a **partial Go re-implementation** of the Lago Rails API, designed to handle high-throughput paths (event ingestion, billing operations) with better performance while sharing the same PostgreSQL database with the Ruby/Rails monolith.

---

## Technology Stack

| Concern | Technology |
|---|---|
| Language | Go 1.24 |
| HTTP Framework | Gin (`gin-gonic/gin`) |
| GraphQL | gqlgen (code-generated) |
| ORM | GORM + pgx/v5 (Postgres) |
| Background Jobs | Asynq (Redis-backed) |
| Message Streaming | Kafka via `franz-go` |
| Auth | API Key (Bearer) + JWT |
| Observability | `slog`, Sentry, OpenTelemetry |
| Testing | `testify`, `go-sqlmock`, `miniredis` |

---

## High-Level Architecture

```mermaid
graph TB
    subgraph Clients
        UI[Front-end / Dashboard]
        ExtAPI[External API Consumers]
        PSP[Payment Providers\nStripe / GoCardless]
    end

    subgraph api-go["api-go Service (Go)"]
        direction TB
        GIN[Gin HTTP Server\n:3000]
        MW[Middleware Layer\nAPIKeyAuth · JWT · CORS · Recovery · Logging]
        REST[REST Handlers /api/v1]
        GQL[GraphQL Handler /graphql]
        AUTH[Auth Handlers /users]
        WH_IN[Inbound Webhooks\n/webhooks/stripe\n/webhooks/gocardless]
        SVC[Service Layer]
        DOM[Domain Layer\nState Machines]
        MODELS[Models / GORM]
        KAFKA_P[Kafka Producer]
    end

    subgraph Worker["worker (Go)"]
        ASYNQ[Asynq Worker]
        SCHED[Asynq Scheduler]
        JOB_H[Job Handlers\nInvoice · Webhook · Payment · Events]
    end

    subgraph Infra
        PG[(PostgreSQL\nShared DB)]
        REDIS[(Redis\nJob Queue)]
        KAFKA[(Kafka / Redpanda\nevents-raw topic)]
    end

    UI -->|GraphQL| GQL
    ExtAPI -->|REST + Bearer Token| REST
    PSP -->|Webhook POST| WH_IN

    GIN --> MW --> REST
    GIN --> MW --> GQL
    GIN --> AUTH
    GIN --> WH_IN

    REST --> SVC
    GQL --> SVC
    AUTH --> SVC
    WH_IN --> SVC

    SVC --> DOM
    SVC --> MODELS --> PG
    SVC --> KAFKA_P --> KAFKA

    ASYNQ -->|dequeue| REDIS
    SCHED -->|enqueue cron| REDIS
    ASYNQ --> JOB_H --> PG

    Worker -.->|shares DB| PG
```

---

## File & Folder Structure

```
api-go/
├── cmd/
│   ├── api/main.go          # HTTP server entry point
│   └── worker/main.go       # Asynq background worker entry point
├── config/
│   ├── config.go            # Env-var-based Config struct
│   ├── database/database.go # GORM + pgx connection pool
│   └── redis/redis.go       # go-redis connection
├── internal/
│   ├── server/server.go     # Gin engine wiring + all routes
│   ├── middleware/          # Auth, permissions, logging, recovery, GraphQL
│   ├── models/              # GORM models (shared schema with Rails)
│   ├── handlers/            # HTTP handlers (one sub-package per resource)
│   ├── services/            # Business logic services
│   ├── domain/              # Pure domain logic (state machines, totals)
│   ├── chargemodels/        # Charge calculation strategies (Strategy pattern)
│   ├── graphql/             # gqlgen resolver + generated code + dataloaders
│   ├── jobs/                # Asynq runtime, scheduler, mux
│   │   └── handlers/        # Job handler functions
│   └── kafka/               # Kafka producer (EventPublisher interface)
├── migrations/              # SQL migration files (up/down)
├── utils/                   # Logger, Sentry error tracker, Result type
├── schema.graphql           # GraphQL schema (source of truth for gqlgen)
├── gqlgen.yml               # gqlgen code-gen configuration
├── go.mod / go.sum
├── Makefile
├── Dockerfile / Dockerfile.dev
└── .env.dist                # Environment variable reference
```

---

## Module & Feature Details

### 1. HTTP API (`internal/server` + `internal/handlers`)

Two API surfaces exist:

```mermaid
graph LR
    subgraph Public Routes
        A[POST /users/login]
        B[POST /users/register]
        C[GET /health]
        D[GET /ready]
    end
    subgraph "REST /api/v1 — API Key Auth"
        E[Customers CRUD]
        F[Events ingestion + batch + fee estimate]
        G[Billable Metrics CRUD]
        H[Plans CRUD]
        I[Subscriptions CRUD + current_usage]
        J[Invoices CRUD + finalize + void]
        K[Webhook Endpoints CRUD]
        L[Webhooks list/show]
    end
    subgraph GraphQL /graphql
        M[BillableMetrics]
        N[Invoices]
        O[Plans]
        P[Subscriptions]
        Q[WebhookEndpoints]
    end
    subgraph "Inbound Webhooks"
        R[POST /webhooks/stripe/:org_id]
        S[POST /webhooks/gocardless/:org_id]
    end
```

### 2. Authentication & Authorization

```mermaid
flowchart TD
    A[Request] --> B{Route type?}
    B -->|/api/v1/*| C[APIKeyAuth middleware\nBearer token → DB lookup]
    B -->|/graphql| D[GraphQLAPIKeyContext middleware\nAPI key → org context]
    B -->|/users/*| E[No auth - public]
    B -->|/webhooks/*| F[Signature validation\nStripe HMAC / GoCardless]
    C --> G[RequirePermission middleware\nresource + action RBAC]
    D --> H[JWT optional\nfor user identity]
    G --> I[Handler]
    H --> I
```

- **API keys** stored in `api_keys` table with scoped permissions JSON
- **JWT tokens** issued on login/register (used by GraphQL / dashboard)
- **RBAC**: `RequirePermission(resource, action)` middleware on every route

### 3. Domain Layer (`internal/domain`)

Pure Go domain logic with no framework dependencies.

**Invoice State Machine:**

```mermaid
stateDiagram-v2
    [*] --> Draft
    Draft --> Finalized : ApplyFinalize()
    Draft --> CannotVoid : ❌ void not allowed
    Generating --> Finalized : ApplyFinalize()
    Finalized --> Voided : ApplyVoid()
    Voided --> Terminal : ❌ terminal state

    note right of Finalized : stamps FinalizedAt
    note right of Voided : stamps VoidedAt
```

**Subscription State Machine:**

```mermaid
stateDiagram-v2
    [*] --> Pending
    Pending --> Active : activate
    Active --> Terminated : terminate
    Active --> Canceled : cancel
```

### 4. Charge Models (`internal/chargemodels`)

Strategy pattern with 8 implementations:

```mermaid
classDiagram
    class Strategy {
        <<interface>>
        +Compute(units, properties) Result
    }
    Strategy <|-- StandardStrategy
    Strategy <|-- GraduatedStrategy
    Strategy <|-- PackageStrategy
    Strategy <|-- PercentageStrategy
    Strategy <|-- VolumeStrategy
    Strategy <|-- GraduatedPercentageStrategy
    Strategy <|-- CustomStrategy
    Strategy <|-- DynamicStrategy

    class Result {
        +Amount decimal
        +UnitAmount decimal
        +AmountDetails map
    }
```

| Model | Description |
|---|---|
| `standard` | Flat per-unit price |
| `graduated` | Tiered pricing with price ranges |
| `package` | Per-package bulk pricing |
| `percentage` | % of transaction value |
| `volume` | Volume-based (all units at same tier price) |
| `graduated_percentage` | Tiered percentage |
| `custom` | Custom aggregator expression |
| `dynamic` | Dynamic unit price from event properties |

### 5. Event Ingestion & Kafka

```mermaid
sequenceDiagram
    participant Client
    participant EventHandler
    participant EventService
    participant PostgreSQL
    participant KafkaProducer
    participant KafkaTopic as events-raw topic

    Client->>EventHandler: POST /api/v1/events
    EventHandler->>EventService: Create(event)
    EventService->>PostgreSQL: INSERT event
    EventService->>KafkaProducer: PublishRawEvent()
    KafkaProducer->>KafkaTopic: Produce(JSON message)
    EventHandler-->>Client: 200 OK
```

- Events are stored in PostgreSQL **and** published to Kafka
- `EventPublisher` interface → `KafkaPublisher` or `NoopPublisher` (when Kafka not configured)
- Supports SCRAM-SHA-256/512 authentication and TLS

### 6. Background Jobs (`internal/jobs`)

```mermaid
graph LR
    subgraph "Scheduler (cron)"
        S1["*/1 min → FinalizeInvoice"]
        S2["*/10 min → MarkPaymentOverdue"]
        S3["*/5 min → ValidateEvents"]
        S4["*/5 min → RuntimeProbe"]
    end

    subgraph "Queues (priority 6:3:1)"
        Q1[critical]
        Q2[default]
        Q3[low]
    end

    subgraph Handlers
        H1[HandleFinalizeInvoice]
        H2[HandleMarkPaymentOverdue]
        H3[HandleValidateEvents]
        H4[HandleSendHTTPWebhook]
        H5[HandleCreatePayment]
    end

    Redis[(Redis)] --> Worker[Asynq Worker\n20 concurrency]
    S1 --> Q1 --> H1
    S2 --> Q2 --> H2
    S3 --> Q2 --> H3
    Worker --> H1
    Worker --> H4
    Worker --> H5
```

### 7. Data Models (`internal/models`)

```mermaid
erDiagram
    Organization ||--o{ APIKey : has
    Organization ||--o{ Customer : has
    Organization ||--o{ BillableMetric : has
    Organization ||--o{ Plan : has
    Organization ||--o{ Invoice : has
    Organization ||--o{ WebhookEndpoint : has

    Customer ||--o{ Subscription : has
    Customer ||--o{ Invoice : has
    Customer ||--o{ Event : generates

    Plan ||--o{ Charge : has
    Plan ||--o{ Subscription : used_in
    BillableMetric ||--o{ Charge : measured_by
    BillableMetric ||--o{ BillableMetricFilter : has
    Charge ||--o{ ChargeFilter : has

    Subscription ||--o{ Fee : generates
    Invoice ||--o{ Fee : contains

    WebhookEndpoint ||--o{ Webhook : triggers
```

### 8. GraphQL Layer (`internal/graphql`)

- **Schema-first** with gqlgen code generation (`schema.graphql` → `generated/generated.go`)
- **DataLoaders** via `graph-gophers/dataloader` (prevents N+1 queries)
- **Cursor-based pagination** (`internal/graphql/pagination`)
- `graphcontext` package extracts org/user identity from the request context
- `ErrorPresenter` maps domain errors to GraphQL-shaped error responses

### 9. Observability & Utilities (`utils/`)

| Utility | Purpose |
|---|---|
| `logger.go` | `LevelHandler` wrapping `slog` with level filtering |
| `error_tracker.go` | Sentry integration helpers |
| `result.go` | Generic `Result[T]` type for service return values |
| `env.go` | `GetEnvOrDefault` helper |

### 10. Configuration

All configuration is via environment variables:

| Variable | Purpose | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL DSN | — |
| `DATABASE_POOL` | Max DB connections | `10` |
| `REDIS_URL` | Redis URL | `redis://localhost:6379` |
| `SECRET_KEY_BASE` | JWT signing secret | — |
| `SENTRY_DSN` | Sentry error tracking | — |
| `PORT` | HTTP listen port | `3000` |
| `ENV` | Environment name | `development` |
| `LAGO_KAFKA_BOOTSTRAP_SERVERS` | Kafka brokers | — |
| `LAGO_KAFKA_RAW_EVENTS_TOPIC` | Kafka topic for raw events | `events-raw` |
| `LAGO_KAFKA_TLS` | Enable Kafka TLS | `false` |
| `LAGO_KAFKA_SCRAM_ALGORITHM` | SASL SCRAM algorithm | — |

---

## Design Patterns Used

| Pattern | Where |
|---|---|
| Strategy | `chargemodels` — 8 charge computation strategies |
| State Machine | `domain/invoices`, `domain/subscriptions` |
| Repository (via GORM) | `models` + `services` |
| Dependency Injection | Services injected into handlers via constructors |
| Interface-driven design | `EventPublisher`, `Service` interfaces |
| Middleware chain | Gin middleware stack (auth → permissions → handler) |
| DataLoader | GraphQL N+1 prevention |

---

## Summary

`api-go` is a **focused Go microservice** implementing the performance-sensitive paths of Lago's billing platform — event ingestion, invoice lifecycle management, and outbound webhook delivery — while sharing a PostgreSQL database with the Ruby on Rails monolith. It follows a clean **layered architecture**: HTTP handlers → service layer → domain logic + GORM models, with a Strategy pattern for charge computation and Asynq for background job processing.

---

# `/api/v1` Controller → Service Dependency Map (Full Repo)

## Who Calls `/api/v1`?

```mermaid
graph TB
    subgraph Callers["Who calls /api/v1?"]
        EXT[External Developers\nBearer API key]
        FRONT[Front-end Dashboard\nApollo GraphQL only]
        PSP[Payment Providers\nStripe / GoCardless webhooks]
    end

    subgraph Rails["/api — Rails API (full surface)"]
        V1["/api/v1 controllers\n~30 controllers"]
    end

    subgraph GoAPI["/api-go — Go API (high-throughput subset)"]
        GOV1["/api/v1 handlers\n9 handler groups"]
    end

    EXT -->|"REST Bearer Token"| V1
    EXT -->|"REST Bearer Token"| GOV1
    FRONT -->|"GraphQL only (/graphql)"| Rails
    PSP -->|"/webhooks/stripe · /webhooks/gocardless"| Rails
    PSP -->|"/webhooks/stripe · /webhooks/gocardless"| GoAPI
```

> **Key facts:**
> - The **front-end never calls `/api/v1`** — it uses Apollo GraphQL exclusively.
> - **External API clients** (developers integrating with Lago) call `/api/v1` on both Rails and Go.
> - **Events-processor** and **connectors** never call `/api/v1` — they consume Kafka / SQS.
> - **Rails** is the full platform; **api-go** covers the high-throughput subset.

---

## Rails `/api/v1` — Controller → Service Map

### Core Billing

```mermaid
graph LR
    CUST["CustomersController\n─────────────────\ncreate · index · show · destroy\nportal_url · checkout_url"]
    CUST --> CS1["Customers::UpsertFromApiService"]
    CUST --> CS2["Customers::DestroyService"]
    CUST --> CS3["CustomerPortal::GenerateUrlService"]
    CUST --> CS4["Customers::GenerateCheckoutUrlService"]

    SUB["SubscriptionsController\n─────────────────\ncreate · update · show · index · terminate"]
    SUB --> SS1["Subscriptions::CreateService"]
    SUB --> SS2["Subscriptions::TerminateService"]
    SUB --> SS3["Subscriptions::UpdateService"]
    SUB --> SS4["BillingEntities::ResolveService"]
    SUB --> SS5["PaymentProviders::Stripe::Payments::AuthorizeService"]

    PLAN["PlansController\n─────────────────\ncreate · update · show · index · destroy"]
    PLAN --> PS1["Plans::CreateService"]
    PLAN --> PS2["Plans::UpdateService"]
    PLAN --> PS3["Plans::PrepareDestroyService"]
    PLAN --> PS4["PlansQuery"]
```

```mermaid
graph LR
    INV["InvoicesController\n─────────────────\ncreate · update · show · index\nfinalize · void · download_pdf · download_xml\nrefresh · retry · retry_payment\nresend_email · payment_url\nlose_dispute · sync_salesforce_id"]
    INV --> IS1["Invoices::CreateOneOffService"]
    INV --> IS2["Invoices::UpdateService"]
    INV --> IS3["Invoices::RefreshDraftService"]
    INV --> IS4["Invoices::RefreshDraftAndFinalizeService"]
    INV --> IS5["Invoices::VoidService"]
    INV --> IS6["Invoices::LoseDisputeService"]
    INV --> IS7["Invoices::RetryService"]
    INV --> IS8["Invoices::Payments::RetryService"]
    INV --> IS9["Invoices::Payments::GeneratePaymentUrlService"]
    INV --> IS10["Invoices::SyncSalesforceIdService"]
    INV --> IJ1["Invoices::GeneratePdfJob ⚡"]
    INV --> IJ2["Invoices::GenerateXmlJob ⚡"]
    INV --> IJ3["Emails::ResendService"]

    EV["EventsController\n─────────────────\ncreate · batch · show · index\nestimate_fees · batch_estimate_instant_fees\nindex_enriched"]
    EV --> ES1["Events::CreateService"]
    EV --> ES2["Events::CreateBatchService"]
    EV --> ES3["Fees::EstimatePayInAdvanceService"]
    EV --> ES4["Fees::EstimateInstant::PayInAdvanceService"]
    EV --> ES5["Fees::EstimateInstant::BatchPayInAdvanceService"]
    EV --> ES6["EventsQuery"]
```

```mermaid
graph LR
    BM["BillableMetricsController\n─────────────────\ncreate · update · destroy · show · index\nevaluate_expression"]
    BM --> BMS1["BillableMetrics::CreateService"]
    BM --> BMS2["BillableMetrics::UpdateService"]
    BM --> BMS3["BillableMetrics::DestroyService"]
    BM --> BMS4["BillableMetrics::EvaluateExpressionService"]
    BM --> BMS5["BillableMetricsQuery"]
```

### Finance

```mermaid
graph LR
    FEE["FeesController\n─────────────────\nshow · update · index · destroy"]
    FEE --> FS1["Fees::UpdateService"]
    FEE --> FS2["Fees::DestroyService"]
    FEE --> FS3["FeesQuery"]

    CN["CreditNotesController\n─────────────────\ncreate · update · show · index · void\ndownload_pdf · download_xml\nestimate · resend_email"]
    CN --> CNS1["CreditNotes::CreateService"]
    CN --> CNS2["CreditNotes::UpdateService"]
    CN --> CNS3["CreditNotes::VoidService"]
    CN --> CNS4["CreditNotes::EstimateService"]
    CN --> CNJ1["CreditNotes::GeneratePdfJob ⚡"]
    CN --> CNJ2["CreditNotes::GenerateXmlJob ⚡"]
    CN --> CNS5["Emails::ResendService"]

    PAY["PaymentsController\n─────────────────\ncreate · index · show"]
    PAY --> PS["Payments::ManualCreateService"]

    PAYREQ["PaymentRequestsController\n─────────────────\ncreate · index · show"]
    PAYREQ --> PRS["PaymentRequests::CreateService"]
```

```mermaid
graph LR
    WLT["WalletsController\n─────────────────\ncreate · update · terminate · show · index"]
    WLT --> WS1["Wallets::CreateService"]
    WLT --> WS2["Wallets::UpdateService"]
    WLT --> WS3["Wallets::TerminateService"]

    WLTTX["WalletTransactionsController\n─────────────────\ncreate · show · index · payment_url\nconsumptions · fundings"]
    WLTTX --> WTS1["WalletTransactions::CreateFromParamsService"]
    WLTTX --> WTS2["WalletTransactions::Payments::GeneratePaymentUrlService"]
    WLTTX --> WTS3["WalletTransactionsQuery"]
    WLTTX --> WTS4["WalletTransactionConsumptionsQuery"]

    APCOUPON["AppliedCouponsController\n─────────────────\ncreate · index"]
    APCOUPON --> ACS["AppliedCoupons::CreateService"]

    ACTERM["Customers::AppliedCouponsController\n─────────────────\nindex · destroy"]
    ACTERM --> ACTS["AppliedCoupons::TerminateService"]
```

### Catalog

```mermaid
graph LR
    ADDON["AddOnsController\n─────────────────\ncreate · update · destroy · show · index"]
    ADDON --> AS1["AddOns::CreateService"]
    ADDON --> AS2["AddOns::UpdateService"]
    ADDON --> AS3["AddOns::DestroyService"]
    ADDON --> AS4["AddOnsQuery"]

    COUPON["CouponsController\n─────────────────\ncreate · update · destroy · show · index"]
    COUPON --> CPS1["Coupons::CreateService"]
    COUPON --> CPS2["Coupons::UpdateService"]
    COUPON --> CPS3["Coupons::DestroyService"]
    COUPON --> CPS4["CouponsQuery"]

    TAX["TaxesController\n─────────────────\ncreate · update · destroy · show · index"]
    TAX --> TS1["Taxes::CreateService"]
    TAX --> TS2["Taxes::UpdateService"]
    TAX --> TS3["Taxes::DestroyService"]
    TAX --> TS4["TaxesQuery"]

    FEAT["FeaturesController\n─────────────────\ncreate · update · destroy · show · index"]
    FEAT --> FTS1["Entitlement::FeatureCreateService"]
    FEAT --> FTS2["Entitlement::FeatureUpdateService"]
    FEAT --> FTS3["Entitlement::FeatureDestroyService"]
    FEAT --> FTS4["Entitlement::FeaturesQuery"]
```

### Configuration & Webhooks

```mermaid
graph LR
    ORG["OrganizationsController\n─────────────────\nshow · update"]
    ORG --> OS["Organizations::UpdateService"]

    BE["BillingEntitiesController\n─────────────────\nindex · show · create · update"]
    BE --> BES1["BillingEntities::CreateService"]
    BE --> BES2["BillingEntities::UpdateService"]

    WE["WebhookEndpointsController\n─────────────────\ncreate · update · show · index · destroy"]
    WE --> WES1["WebhookEndpoints::CreateService"]
    WE --> WES2["WebhookEndpoints::UpdateService"]
    WE --> WES3["WebhookEndpoints::DestroyService"]
    WE --> WES4["WebhookEndpointsQuery"]

    WH["WebhooksController\n─────────────────\nindex · show (read-only)"]
```

### Analytics & Data

```mermaid
graph LR
    ANA["Analytics Controllers\n(5 controllers)"]
    ANA --> ANA1["Analytics::GrossRevenuesService\n→ GrossRevenuesController#index"]
    ANA --> ANA2["Analytics::InvoiceCollectionsService\n→ InvoiceCollectionsController#index"]
    ANA --> ANA3["Analytics::InvoicedUsagesService\n→ InvoicedUsagesController#index"]
    ANA --> ANA4["Analytics::MrrsService\n→ MrrsController#index"]
    ANA --> ANA5["Analytics::OverdueBalancesService\n→ OverdueBalancesController#index"]

    CU["Customers::UsageController\n─────────────────\ncurrent · past"]
    CU --> CUS1["Invoices::CustomerUsageService"]
    CU --> CUS2["PastUsageQuery"]

    SLA["Subscriptions::AlertsController\n─────────────────\ncreate · update · show · index · destroy · destroy_all"]
    SLA --> SLAS1["UsageMonitoring::CreateAlertService"]
    SLA --> SLAS2["UsageMonitoring::UpdateAlertService"]
    SLA --> SLAS3["UsageMonitoring::DestroyAlertService"]
    SLA --> SLAS4["UsageMonitoring::Alerts::DestroyAllService"]
    SLA --> SLAS5["UsageMonitoring::AlertsQuery"]

    SLU["Subscriptions::LifetimeUsagesController\n─────────────────\nshow · update"]
    SLU --> SLUS["LifetimeUsages::UpdateService"]

    DAPI["DataApi::UsagesController\n─────────────────\nindex"]
    DAPI --> DAPIS["DataApi::UsagesService"]

    ALOGS["ActivityLogsController\n─────────────────\nindex · show"]
    ALOGS --> ALOGQ["ActivityLogsQuery"]
```

---

## Rails `/api/v1` — Full Reference Table

| Controller | File | Actions | Services / Jobs |
|---|---|---|---|
| **Customers** | `customers_controller.rb` | create, index, show, destroy, portal_url, checkout_url | `Customers::UpsertFromApiService`, `DestroyService`, `CustomerPortal::GenerateUrlService`, `GenerateCheckoutUrlService` |
| **Subscriptions** | `subscriptions_controller.rb` | create, update, show, index, terminate | `Subscriptions::CreateService`, `TerminateService`, `UpdateService`, `BillingEntities::ResolveService`, `Stripe::Payments::AuthorizeService` |
| **Plans** | `plans_controller.rb` | CRUD | `Plans::CreateService`, `UpdateService`, `PrepareDestroyService`, `PlansQuery` |
| **Invoices** | `invoices_controller.rb` | create, update, show, index, finalize, void, refresh, retry, download_pdf, download_xml, resend_email, payment_url, lose_dispute, sync_salesforce_id | `Invoices::CreateOneOffService`, `RefreshDraftAndFinalizeService`, `VoidService`, `LoseDisputeService`, `RetryService`, `Payments::RetryService`, `GeneratePaymentUrlService`, `SyncSalesforceIdService`, **`GeneratePdfJob`**, **`GenerateXmlJob`**, `Emails::ResendService` |
| **Events** | `events_controller.rb` | create, batch, show, index, estimate_fees, batch_estimate_instant_fees | `Events::CreateService`, `CreateBatchService`, `Fees::EstimatePayInAdvanceService`, `EstimateInstant::PayInAdvanceService`, `EstimateInstant::BatchPayInAdvanceService` |
| **BillableMetrics** | `billable_metrics_controller.rb` | CRUD, evaluate_expression | `BillableMetrics::CreateService`, `UpdateService`, `DestroyService`, `EvaluateExpressionService`, `BillableMetricsQuery` |
| **Fees** | `fees_controller.rb` | show, update, index, destroy | `Fees::UpdateService`, `DestroyService`, `FeesQuery` |
| **CreditNotes** | `credit_notes_controller.rb` | create, update, show, index, void, download_pdf, download_xml, estimate, resend_email | `CreditNotes::CreateService`, `UpdateService`, `VoidService`, `EstimateService`, **`GeneratePdfJob`**, **`GenerateXmlJob`**, `Emails::ResendService` |
| **Payments** | `payments_controller.rb` | create, index, show | `Payments::ManualCreateService` |
| **PaymentRequests** | `payment_requests_controller.rb` | create, index, show | `PaymentRequests::CreateService` |
| **Wallets** | `wallets_controller.rb` | create, update, terminate, show, index | `Wallets::CreateService`, `UpdateService`, `TerminateService` |
| **WalletTransactions** | `wallet_transactions_controller.rb` | create, show, index, payment_url, consumptions, fundings | `WalletTransactions::CreateFromParamsService`, `Payments::GeneratePaymentUrlService`, `WalletTransactionsQuery` |
| **Coupons** | `coupons_controller.rb` | CRUD | `Coupons::CreateService`, `UpdateService`, `DestroyService`, `CouponsQuery` |
| **AppliedCoupons** | `applied_coupons_controller.rb` | create, index | `AppliedCoupons::CreateService` |
| **AddOns** | `add_ons_controller.rb` | CRUD | `AddOns::CreateService`, `UpdateService`, `DestroyService`, `AddOnsQuery` |
| **Taxes** | `taxes_controller.rb` | CRUD | `Taxes::CreateService`, `UpdateService`, `DestroyService`, `TaxesQuery` |
| **Features** | `features_controller.rb` | CRUD | `Entitlement::FeatureCreateService`, `FeatureUpdateService`, `FeatureDestroyService`, `FeaturesQuery` |
| **BillingEntities** | `billing_entities_controller.rb` | index, show, create, update | `BillingEntities::CreateService`, `UpdateService` |
| **Organizations** | `organizations_controller.rb` | show, update | `Organizations::UpdateService` |
| **WebhookEndpoints** | `webhook_endpoints_controller.rb` | CRUD | `WebhookEndpoints::CreateService`, `UpdateService`, `DestroyService`, `WebhookEndpointsQuery` |
| **Webhooks** | `webhooks_controller.rb` | index, show | query only |
| **Analytics::GrossRevenues** | `analytics/gross_revenues_controller.rb` | index | `Analytics::GrossRevenuesService` |
| **Analytics::InvoiceCollections** | `analytics/invoice_collections_controller.rb` | index | `Analytics::InvoiceCollectionsService` |
| **Analytics::InvoicedUsages** | `analytics/invoiced_usages_controller.rb` | index | `Analytics::InvoicedUsagesService` |
| **Analytics::Mrrs** | `analytics/mrrs_controller.rb` | index | `Analytics::MrrsService` |
| **Analytics::OverdueBalances** | `analytics/overdue_balances_controller.rb` | index | `Analytics::OverdueBalancesService` |
| **Customers::Usage** | `customers/usage_controller.rb` | current, past | `Invoices::CustomerUsageService`, `PastUsageQuery` |
| **Customers::AppliedCoupons** | `customers/applied_coupons_controller.rb` | index, destroy | `AppliedCoupons::TerminateService` |
| **Customers::Wallets** | `customers/wallets_controller.rb` | create, update, terminate, show, index | — (delegates to Wallets) |
| **Subscriptions::Alerts** | `subscriptions/alerts_controller.rb` | create, update, show, index, destroy, destroy_all | `UsageMonitoring::CreateAlertService`, `UpdateAlertService`, `DestroyAlertService`, `Alerts::DestroyAllService`, `AlertsQuery` |
| **Subscriptions::LifetimeUsages** | `subscriptions/lifetime_usages_controller.rb` | show, update | `LifetimeUsages::UpdateService` |
| **Subscriptions::Charges** | `subscriptions/charges_controller.rb` | index, show, update | — |
| **Subscriptions::Charges::Filters** | `subscriptions/charges/filters_controller.rb` | CRUD | `ChargeFilters::{Create,Update,Destroy}Service` |
| **Subscriptions::FixedCharges** | `subscriptions/fixed_charges_controller.rb` | index, show, update | — |
| **Subscriptions::Entitlements** | `subscriptions/entitlements_controller.rb` | index, destroy, update | — |
| **Plans::Charges** | `plans/charges_controller.rb` | CRUD | `Charges::{Create,Update,Destroy}Service` |
| **Plans::Charges::Filters** | `plans/charges/filters_controller.rb` | CRUD | `ChargeFilters::{Create,Update,Destroy}Service` |
| **Plans::FixedCharges** | `plans/fixed_charges_controller.rb` | CRUD | `FixedCharges::{Create,Update,Destroy}Service` |
| **Plans::Entitlements** | `plans/entitlements_controller.rb` | index, show, create, destroy | `Entitlement::PlanEntitlementsUpdateService` |
| **Plans::Entitlements::Privileges** | `plans/entitlements/privileges_controller.rb` | destroy | `Entitlement::PlanEntitlementPrivilegeDestroyService` |
| **Plans::Metadata** | `plans/metadata_controller.rb` | create, update, destroy | `Plans::UpdateService`, `Metadata::DeleteItemKeyService` |
| **Customers::ProjectedUsage** | `customers/projected_usage_controller.rb` | current | `Invoices::CustomerUsageService` |
| **Customers::PaymentMethods** | `customers/payment_methods_controller.rb` | index, destroy, set_as_default | `PaymentMethods::{Destroy,SetAsDefault}Service` |
| **PaymentReceipts** | `payment_receipts_controller.rb` | index, show, resend_email | `Emails::ResendService` |
| **ApiLogs** | `api_logs_controller.rb` | index, show | query only |
| **SecurityLogs** | `security_logs_controller.rb` | index, show | query only |
| **DataApi::Usages** | `data_api/usages_controller.rb` | index | `DataApi::UsagesService` |
| **ActivityLogs** | `activity_logs_controller.rb` | index, show | `ActivityLogsQuery` |

---

## `api-go` `/api/v1` — Handler → Service Map

| Handler Package | Routes | Service | Extra |
|---|---|---|---|
| `handlers/customers` | POST, GET (list/show), DELETE `/customers`, GET `portal_url` | `customers.NewService` | — |
| `handlers/events` | POST `/events`, POST `/events/batch`, GET `/events`, GET `/events/estimate_fees` | `events.NewService` | Publishes to **Kafka** via `EventPublisher` |
| `handlers/invoices` | POST, GET (list/show), PUT `/finalize`, PUT `/void` | `invoices.NewService` | Domain state machine in `domain/invoices` |
| `handlers/plans` | CRUD `/plans` | `plans.NewService` | — |
| `handlers/subscriptions` | CRUD + DELETE (terminate) + GET `current_usage` | `subscriptions.NewService` | — |
| `handlers/billable_metrics` | CRUD `/billable_metrics` | `billable_metrics.NewService` | — |
| `handlers/webhook_endpoints` | CRUD + GET `event_types` | `webhook_endpoints.NewService` | — |
| `handlers/webhooks` | GET `/webhooks`, GET `/webhooks/:id` | direct GORM query | — |
| `handlers/organizations` | GET, PUT `/organizations` | `organizations.NewService` | — |
| `handlers/auth` | POST `/users/login`, POST `/users/register` | `users.NewAuthService` (JWT) | Public, no API key |

---

## Key Design Observations

1. **Controllers are thin** — all logic is in `*Service` objects; controllers only parse params, invoke a service, and render JSON.
2. **Background jobs are triggered by controllers** — `GeneratePdfJob`, `GenerateXmlJob`, `GeneratePdfJob` (credit notes) are enqueued directly from controller actions.
3. **`api-go` is a strict subset of Rails** — covers 9 of ~30 Rails controller groups, focused on the high-throughput paths.
4. **Front-end is GraphQL-only** — zero direct `/api/v1` REST calls from the dashboard; all UI data goes through Apollo → GraphQL → Rails resolvers.
5. **Events-processor & connectors never call `/api/v1`** — events-processor consumes Kafka; connectors use SQS/HTTP (Benthos config files).
6. **Payment provider webhooks** hit `/webhooks/stripe` and `/webhooks/gocardless` — not `/api/v1` — and have their own signature validation middleware.
7. **Total scope**: 66+ Ruby controllers (across main + nested namespaces) + 11 Go handler groups = **77+ endpoint groups**, **37+ unique service classes**, **4 background jobs**.
