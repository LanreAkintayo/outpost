# Outpost — Feature Roadmap

> When every checkbox in this document is checked, **Outpost is complete and portfolio-ready.**

---

## Milestone 1: Foundation
> *"I have a running API that manages applications, endpoints, event types, and subscriptions."*

### 1.1 Project Setup
- [x] Initialize Go module (`github.com/LanreAkintayo/outpost`)
- [x] Create project directory structure per architecture guidelines
- [x] Create `Makefile` with targets: `build`, `run`, `test`, `lint`, `migrate-up`, `migrate-down`
- [x] Create `docker-compose.yml` with PostgreSQL 16
- [x] Create `.env.example` with all required environment variables
- [x] Create `Dockerfile` (multi-stage build: build stage + minimal runtime stage)
- [x] Verify: `docker-compose up -d` starts PostgreSQL, `make run` starts the server

### 1.2 Configuration
- [x] Create `internal/config/config.go` — typed Config struct loaded from environment variables
- [x] Config fields: `DatabaseURL`, `ServerPort`, `WorkerCount`, `MaxRetries`, `RetryBaseDelay`
- [x] Validate required fields on startup (fail fast if missing)
- [x] Verify: app prints loaded config on startup (masking sensitive values)

### 1.3 Database & Migrations
- [x] Set up database connection pool using `pgx` or `database/sql` + `lib/pq`
- [ ] Integrate `golang-migrate` for migration management
- [x] Migration 001: Create `applications` table
- [x] Migration 002: Create `endpoints` table
- [x] Migration 003: Create `event_types` table
- [x] Migration 004: Create `subscriptions` table
- [ ] Migration 005: Create `events` table
- [ ] Migration 006: Create `delivery_attempts` table
- [ ] Add performance indexes on frequently queried columns
- [ ] Verify: `make migrate-up` creates all tables, `make migrate-down` drops them cleanly

### 1.4 Structured Logging
- [x] Set up structured logger (`slog` or `zerolog`)
- [x] Log format: JSON in production, human-readable in development
- [x] Logger is injected via dependency injection (not a global variable)
- [x] Verify: all log output is structured with timestamp, level, message, and context fields

### 1.5 Application Management
- [x] `internal/models/application.go` — Application domain model
- [x] `internal/dto/application_dto.go` — CreateApplicationRequest, ApplicationResponse
- [x] `internal/repository/application_repo.go` — Interface + PostgreSQL implementation
  - [x] `Create(app)` — insert new application
  - [x] `GetByID(id)` — fetch by UUID
  - [x] `GetByAPIKey(key)` — fetch by API key (for auth middleware)
- [x] `internal/service/application_service.go` — business logic
  - [x] Generate unique API key with prefix `op_live_` on creation
- [x] `internal/handler/application_handler.go` — HTTP handler
  - [x] `POST /api/v1/applications` — create application, return API key
- [x] Verify with curl: create an application, receive a UUID and API key in response

### 1.6 Middleware
- [x] `internal/middleware/auth.go` — API key authentication
  - [x] Extract API key from `Authorization: Bearer <key>` header
  - [x] Look up application by API key in database
  - [x] Attach application to request context
  - [x] Return `401 Unauthorized` if key is missing or invalid
- [x] `internal/middleware/logging.go` — request logging
  - [x] Log method, path, status code, latency, client IP
- [x] `internal/middleware/recovery.go` — panic recovery
  - [x] Catch panics, log stack trace, return 500
- [x] Verify: unauthenticated requests get 401, valid key proceeds, all requests are logged, panics don't crash the server

### 1.7 Endpoint Management
- [x] `internal/models/endpoint.go` — Endpoint domain model
- [x] `internal/dto/endpoint_dto.go` — CreateEndpointRequest, EndpointResponse
- [x] `internal/repository/endpoint_repo.go` — Interface + PostgreSQL implementation
  - [x] `Create(endpoint)` — insert new endpoint
  - [x] `GetByID(id)` — fetch by UUID
  - [x] `ListByApplication(appID)` — list all endpoints for an application
  - [x] `Delete(id)` — soft delete or hard delete an endpoint
  - [x] `Update(id, fields)` — update URL, description, or active status
- [x] `internal/service/endpoint_service.go` — business logic
  - [x] Generate unique secret with prefix `whsec_` on creation
- [x] `internal/handler/endpoint_handler.go` — HTTP handlers
  - [x] `POST /api/v1/endpoints` — register a new endpoint
  - [x] `GET /api/v1/endpoints` — list all endpoints for the authenticated application
  - [x] `GET /api/v1/endpoints/:id` — get endpoint details
  - [x] `PUT /api/v1/endpoints/:id` — update an endpoint
  - [x] `DELETE /api/v1/endpoints/:id` — remove an endpoint
- [x] Verify with curl: full CRUD operations on endpoints

### 1.8 Event Type Management
- [x] `internal/models/event_type.go` — EventType domain model
- [x] `internal/dto/event_type_dto.go` — CreateEventTypeRequest, EventTypeResponse
- [x] `internal/repository/event_type_repo.go` — Interface + PostgreSQL implementation
  - [x] `Create(eventType)` — insert new event type
  - [x] `GetByName(appID, name)` — look up event type by name within an application
  - [x] `ListByApplication(appID)` — list all event types for an application
- [x] `internal/service/event_type_service.go` — business logic
  - [x] Validate event type names follow dot-notation (e.g., `payment.completed`, `order.created`)
- [x] `internal/handler/event_type_handler.go` — HTTP handlers
  - [x] `POST /api/v1/event-types` — create an event type
  - [x] `GET /api/v1/event-types` — list all event types
- [x] Verify with curl: create and list event types

### 1.9 Subscription Management
- [x] `internal/models/subscription.go` — Subscription domain model
- [x] `internal/dto/subscription_dto.go` — CreateSubscriptionRequest, SubscriptionResponse
- [x] `internal/repository/subscription_repo.go` — Interface + PostgreSQL implementation
  - [x] `Create(subscription)` — subscribe an endpoint to an event type
  - [x] `Delete(endpointID, eventTypeID)` — unsubscribe
  - [x] `ListByEndpoint(endpointID)` — what event types does this endpoint listen to?
  - [x] `GetSubscribedEndpoints(appID, eventTypeName)` — which endpoints want this event?
- [x] `internal/handler/subscription_handler.go` — HTTP handlers
  - [x] `POST /api/v1/endpoints/:id/subscriptions` — subscribe to an event type
  - [x] `DELETE /api/v1/endpoints/:id/subscriptions/:event_type_id` — unsubscribe
  - [x] `GET /api/v1/endpoints/:id/subscriptions` — list subscriptions
- [x] Verify with curl: subscribe an endpoint to an event type, list subscriptions

### 1.10 Router Setup
- [x] `internal/router/router.go` — centralized route definitions
- [x] Group routes under `/api/v1/`
- [x] Apply auth middleware to all routes except `POST /api/v1/applications`
- [x] Apply logging and recovery middleware globally
- [x] Verify: all endpoints respond correctly, unauthenticated routes are protected

---

### ✅ Milestone 1 Acceptance Test
```
1. docker-compose up -d              → PostgreSQL running
2. make migrate-up                   → All tables created
3. make run                          → Server starts on configured port
4. POST /api/v1/applications         → Returns UUID + API key
5. POST /api/v1/event-types          → Creates "payment.completed" (authenticated)
6. POST /api/v1/endpoints            → Registers a URL with generated secret (authenticated)
7. POST /endpoints/:id/subscriptions → Subscribes endpoint to "payment.completed"
8. GET /api/v1/endpoints             → Lists the registered endpoint
9. Unauthenticated request           → Returns 401
```

---

## Milestone 2: Core Delivery Engine
> *"When I send an event, Outpost delivers it to all subscribed endpoints with a signed payload."*

### 2.1 Event Ingestion
- [ ] `internal/models/event.go` — Event domain model
- [ ] `internal/dto/event_dto.go` — SendEventRequest, EventResponse
- [ ] `internal/repository/event_repo.go` — Interface + PostgreSQL implementation
  - [ ] `Create(event)` — store the event
  - [ ] `GetByID(id)` — fetch event by UUID
  - [ ] `GetByIdempotencyKey(key)` — check for duplicate events
- [ ] `internal/service/event_service.go` — business logic
  - [ ] Validate event type exists
  - [ ] Check idempotency key (skip if already processed)
  - [ ] Store event in database
  - [ ] Look up all subscribed endpoints
  - [ ] Create a `delivery_attempt` row (status: `pending`) for each subscribed endpoint
  - [ ] Return `202 Accepted` to the caller
- [ ] `internal/handler/event_handler.go` — HTTP handler
  - [ ] `POST /api/v1/events` — accept an event for delivery
- [ ] Verify: sending an event creates pending delivery_attempt rows in the database

### 2.2 HMAC-SHA256 Payload Signing
- [ ] `internal/engine/signer.go`
  - [ ] `Sign(payload []byte, secret string) string` — generate HMAC-SHA256 signature
  - [ ] `Verify(payload []byte, secret string, signature string) bool` — verify a signature
  - [ ] Signature format: `sha256=<hex-encoded hash>`
- [ ] Unit tests for signer (known input → known output)
- [ ] Verify: signing the same payload with the same secret always produces the same signature

### 2.3 HTTP Deliverer
- [ ] `internal/engine/deliverer.go`
  - [ ] Send HTTP POST to endpoint URL with:
    - [ ] `Content-Type: application/json` header
    - [ ] `X-Outpost-Event-ID: <event UUID>` header
    - [ ] `X-Outpost-Timestamp: <unix timestamp>` header
    - [ ] `X-Outpost-Signature: sha256=<signature>` header
    - [ ] Event payload as the request body
  - [ ] Configurable timeout per request (default: 30 seconds)
  - [ ] Return delivery result: success (status code), or failure (error message)
- [ ] Verify: a test HTTP server receives the webhook with correct headers and body

### 2.4 Worker Pool
- [ ] `internal/engine/worker_pool.go`
  - [ ] Create a fixed pool of N goroutines (N = config.WorkerCount)
  - [ ] Workers read delivery tasks from a Go channel
  - [ ] Each worker: receive task → sign payload → deliver → log result
  - [ ] Workers run continuously until the pool is shut down
- [ ] Verify: multiple deliveries are processed concurrently (check logs for interleaved processing)

### 2.5 Dispatcher
- [ ] `internal/engine/dispatcher.go`
  - [ ] Run in a background goroutine
  - [ ] Poll the database at a configurable interval (default: every 2 seconds)
  - [ ] Query: `SELECT * FROM delivery_attempts WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= NOW()) LIMIT batch_size`
  - [ ] Send each delivery task to the worker pool channel
  - [ ] Mark tasks as `processing` to prevent other dispatchers from picking them up (locking)
- [ ] `internal/repository/delivery_repo.go` — Interface + PostgreSQL implementation
  - [ ] `GetPendingDeliveries(limit)` — fetch pending tasks ready for delivery
  - [ ] `UpdateStatus(id, status, httpStatus, responseBody, errorMessage)` — record result
  - [ ] `CreateAttempt(eventID, endpointID)` — create a new delivery attempt
- [ ] Verify: sending an event triggers automatic delivery to subscribed endpoints within seconds

---

### ✅ Milestone 2 Acceptance Test
```
1. Start a test webhook receiver (webhook.site or simple Go HTTP server)
2. Register the receiver URL as an endpoint in Outpost
3. Subscribe it to "order.created" event type
4. Send an event: POST /api/v1/events { "event_type": "order.created", "payload": {...} }
5. The test receiver gets an HTTP POST with:
   - The event payload in the body
   - X-Outpost-Signature header (valid HMAC)
   - X-Outpost-Event-ID header
   - X-Outpost-Timestamp header
6. delivery_attempts table shows status = "delivered" with the HTTP status code
```

---

## Milestone 3: Reliability & Fault Tolerance
> *"When deliveries fail, Outpost retries automatically. Nothing is silently lost."*

### 3.1 Exponential Backoff with Jitter
- [ ] `internal/engine/retry.go`
  - [ ] `CalculateNextRetry(attemptNumber int) time.Duration`
  - [ ] Backoff schedule: 30s → 2m → 10m → 1h → 4h (configurable base delay)
  - [ ] Add random jitter (±20%) to prevent thundering herd
- [ ] Unit tests: verify backoff durations increase correctly
- [ ] Verify: retry intervals are not identical when multiple deliveries fail simultaneously

### 3.2 Automatic Retry on Failure
- [ ] When a delivery fails (HTTP 5xx, timeout, connection error):
  - [ ] Increment `attempt_number`
  - [ ] Calculate `next_retry_at` using exponential backoff
  - [ ] Update delivery attempt status to `pending` with the new `next_retry_at`
- [ ] Dispatcher picks up retries when `next_retry_at <= NOW()`
- [ ] Distinguish between retryable failures (5xx, timeout) and permanent failures (4xx)
  - [ ] 4xx responses: do NOT retry (the endpoint explicitly rejected it)
  - [ ] 5xx responses: retry
  - [ ] Connection errors / timeouts: retry
- [ ] Verify: send event to a failing endpoint, observe automatic retries in the logs with increasing intervals

### 3.3 Dead-Letter Queue
- [ ] After `max_retries` (default: 5) failed attempts, mark delivery as `dead_letter`
- [ ] Dead-lettered deliveries are preserved in the database — never deleted
- [ ] Verify: after 5 failed retries, status changes to `dead_letter` and retries stop

### 3.4 Delivery Logs API
- [ ] `internal/handler/delivery_handler.go` — HTTP handlers
  - [ ] `GET /api/v1/events/:id/deliveries` — all delivery attempts for a specific event
  - [ ] `GET /api/v1/deliveries` — list deliveries with filters:
    - [ ] Filter by `status` (pending, delivered, failed, dead_letter)
    - [ ] Filter by `endpoint_id`
    - [ ] Filter by date range (`from`, `to`)
    - [ ] Pagination (page, per_page)
  - [ ] `GET /api/v1/deliveries/:id` — get a single delivery attempt detail
- [ ] Response includes: attempt number, status, HTTP status code, error message, timestamps
- [ ] Verify: query delivery logs after sending events, see complete history of all attempts

### 3.5 Manual Retry
- [ ] `POST /api/v1/deliveries/:id/retry` — manually retry a specific delivery
  - [ ] Only allowed for `failed` or `dead_letter` status
  - [ ] Resets attempt counter and sets status back to `pending`
- [ ] Verify: dead-lettered delivery can be manually retried and succeeds if endpoint is now healthy

### 3.6 Idempotency
- [ ] `POST /api/v1/events` accepts optional `idempotency_key` field
- [ ] If an event with the same idempotency key already exists, return the existing event (don't create a duplicate)
- [ ] Database enforces uniqueness via UNIQUE constraint on `idempotency_key`
- [ ] Verify: sending the same event twice with the same idempotency key only creates one event

---

### ✅ Milestone 3 Acceptance Test
```
1. Register an endpoint pointing to http://localhost:9999 (nothing running — will fail)
2. Subscribe it to "payment.completed"
3. Send a "payment.completed" event
4. Watch logs:
   - Attempt 1: failed — retry in ~30s
   - Attempt 2: failed — retry in ~2m
   - Attempt 3: failed — retry in ~10m
   - Attempt 4: failed — retry in ~1h
   - Attempt 5: failed — moved to dead_letter
5. GET /api/v1/events/:id/deliveries → shows all 5 attempts with timestamps
6. Start a server on port 9999
7. POST /api/v1/deliveries/:id/retry → manually retry the dead-lettered delivery
8. Delivery succeeds ✅
9. Send the same event again with the same idempotency_key → returns existing event, no duplicate
```

---

## Milestone 4: Advanced Features
> *"Outpost handles edge cases like a production system."*

### 4.1 Rate Limiting Per Endpoint
- [ ] Configurable rate limit per endpoint (default: 10 requests/second)
- [ ] Store rate limit in endpoints table (`rate_limit` column)
- [ ] Worker respects rate limit before sending — waits if limit is exceeded
- [ ] Verify: send 50 events to one endpoint rapidly, observe they're delivered at the configured rate

### 4.2 Endpoint Health Tracking
- [ ] Track consecutive failure count per endpoint
- [ ] If an endpoint fails N times in a row (e.g., 20), auto-disable it (`is_active = false`)
- [ ] Auto-disabled endpoints are skipped during delivery
- [ ] API endpoint to re-enable: `PUT /api/v1/endpoints/:id` with `{ "is_active": true }`
- [ ] Verify: consistently failing endpoint gets auto-disabled, re-enabling works

### 4.3 Event Replay
- [ ] `POST /api/v1/events/:id/replay` — re-deliver a specific event to all subscribed endpoints
  - [ ] Creates new delivery_attempt rows (fresh attempt_number = 1)
- [ ] `POST /api/v1/replay` — batch replay with filters:
  - [ ] Filter by status (`dead_letter`, `failed`)
  - [ ] Filter by date range
  - [ ] Filter by endpoint_id
- [ ] Verify: replaying a failed event creates new delivery attempts that succeed

### 4.4 Graceful Shutdown
- [ ] Listen for OS signals: `SIGINT` (Ctrl+C) and `SIGTERM`
- [ ] On signal received:
  - [ ] Stop accepting new HTTP requests
  - [ ] Stop the dispatcher from picking up new tasks
  - [ ] Wait for all in-progress worker deliveries to complete (with a timeout)
  - [ ] Close database connections
  - [ ] Exit cleanly
- [ ] Verify: send events, press Ctrl+C, observe in-progress deliveries complete before exit

### 4.5 Delivery Statistics API
- [ ] `GET /api/v1/stats` — return aggregate delivery statistics:
  - [ ] Total events sent (today, this week, all time)
  - [ ] Total deliveries by status (delivered, failed, pending, dead_letter)
  - [ ] Success rate percentage
  - [ ] Average delivery latency
  - [ ] Per-endpoint health summary
- [ ] Verify: stats reflect real delivery data accurately

---

### ✅ Milestone 4 Acceptance Test
```
1. Set rate limit on an endpoint to 5/second
2. Send 20 events rapidly → observe they're delivered at 5/sec pace
3. Point an endpoint to a permanently failing URL
4. After 20 consecutive failures → endpoint is auto-disabled
5. Re-enable the endpoint via PUT
6. Replay all dead-lettered events → they re-deliver successfully
7. Send events, then Ctrl+C → in-progress deliveries finish before shutdown
8. GET /api/v1/stats → shows accurate counts, success rates, and latency
```

---

## Milestone 5: Polish & Production Readiness
> *"Outpost looks, feels, and works like a real open-source product."*

### 5.1 Dashboard UI
- [ ] `internal/handler/dashboard_handler.go`
- [ ] `GET /dashboard` — renders an HTML dashboard showing:
  - [ ] Total events today / this week / all time
  - [ ] Delivery success rate (visual indicator)
  - [ ] Recent deliveries table (timestamp, event type, endpoint, status, HTTP code)
  - [ ] Endpoint health overview (active, disabled, failure counts)
  - [ ] Dead-letter queue count (events awaiting manual attention)
- [ ] Use Go `html/template` for rendering
- [ ] Basic CSS styling (clean, readable — doesn't need to be flashy)
- [ ] Verify: dashboard loads in browser and shows real data

### 5.2 API Documentation (Swagger)
- [ ] Add `swaggo/swag` annotations to all handler functions
- [ ] Generate Swagger/OpenAPI spec
- [ ] Serve Swagger UI at `/docs`
- [ ] Document all endpoints: method, path, request body, response body, status codes
- [ ] Verify: `/docs` renders interactive API documentation in browser

### 5.3 Unit Tests
- [ ] **Engine tests:**
  - [ ] Signer: known input → known output, verify valid signatures, reject tampered payloads
  - [ ] Retry: backoff duration calculations, jitter ranges, max retry enforcement
- [ ] **Service tests:**
  - [ ] Event service: idempotency, subscription lookup, delivery task creation
  - [ ] Endpoint service: secret generation, validation
  - [ ] Application service: API key generation, uniqueness
- [ ] **Handler tests:**
  - [ ] Test HTTP status codes for valid/invalid requests
  - [ ] Test authentication middleware (valid key, invalid key, missing key)
  - [ ] Test request validation (missing fields, invalid formats)
- [ ] **Repository tests:**
  - [ ] Test against a real test database (Docker test container)
  - [ ] Test CRUD operations, edge cases, constraint violations
- [ ] Target: **80%+ code coverage**
- [ ] Verify: `make test` passes, `make test-coverage` shows 80%+

### 5.4 Integration Tests
- [ ] End-to-end test: create application → register endpoint → subscribe → send event → verify delivery
- [ ] Retry flow test: send to failing endpoint → verify retries → fix endpoint → verify eventual delivery
- [ ] Idempotency test: send duplicate events → verify only one is processed
- [ ] Verify: `make test-integration` passes

### 5.5 Code Quality
- [ ] Set up `golangci-lint` with a `.golangci.yml` config
- [ ] Add `lint` target to Makefile
- [ ] Fix all linting warnings
- [ ] Verify: `make lint` passes with zero warnings

### 5.6 Docker & Deployment
- [ ] `Dockerfile` — multi-stage build (build stage with Go toolchain, runtime stage with minimal image)
- [ ] `docker-compose.yml` — runs PostgreSQL + Outpost with one command
- [ ] Health check endpoint: `GET /health` — returns 200 if the server and database are healthy
- [ ] Verify: `docker-compose up` starts everything, `GET /health` returns 200

### 5.7 README
- [ ] **What Outpost is** — one-paragraph description
- [ ] **Architecture diagram** — system overview (can be ASCII or Mermaid)
- [ ] **Quick Start** — clone → docker-compose up → create application → send first event (under 2 minutes)
- [ ] **API Reference** — every endpoint with example requests and responses
- [ ] **Configuration** — all environment variables documented
- [ ] **Example Integration** — sample Go code showing how to integrate Outpost into an app
- [ ] **Contributing** — how to run locally, run tests, submit PRs
- [ ] Verify: a stranger could read the README and run Outpost in under 5 minutes

---

### ✅ Milestone 5 Acceptance Test (THE FINAL TEST)
```
1.  git clone the repo on a fresh machine
2.  docker-compose up                      → everything starts
3.  Open http://localhost:8080/health       → 200 OK
4.  Open http://localhost:8080/docs         → Swagger UI loads
5.  Open http://localhost:8080/dashboard    → Dashboard shows (empty state)
6.  POST /api/v1/applications              → Get API key
7.  POST /api/v1/event-types               → Create "order.created"
8.  Register 3 endpoints:
      Endpoint A: webhook.site URL (will succeed)
      Endpoint B: http://localhost:9999 (will fail — nothing running)
      Endpoint C: a slow server (5-second delay)
9.  Subscribe all 3 to "order.created"
10. POST /api/v1/events                    → Send "order.created" event
11. Observe:
      Endpoint A: delivered immediately ✅
      Endpoint B: fails → retries with backoff → dead-letters after 5 attempts ☠️
      Endpoint C: delivered (within timeout) ✅
12. GET /api/v1/events/:id/deliveries      → Shows all attempts for all 3 endpoints
13. Start a server on port 9999
14. POST /api/v1/deliveries/:id/retry      → Replay dead-lettered delivery → succeeds ✅
15. GET /api/v1/stats                      → Shows accurate stats
16. Open dashboard                         → Shows delivery data, success rates, endpoint health
17. make test                              → All tests pass, 80%+ coverage
18. make lint                              → Zero warnings
19. Ctrl+C                                 → Graceful shutdown, in-progress deliveries complete
```

**If all 19 steps pass — Outpost is DONE. Ship it. Post it. Be proud.** 🚀

---

## Summary

| Milestone | What You Achieve | Estimated Time |
|---|---|---|
| **M1: Foundation** | Working REST API with full CRUD | 1-2 weeks |
| **M2: Core Delivery** | Webhooks actually get delivered | 1-2 weeks |
| **M3: Reliability** | Automatic retries, dead-letter queue, idempotency | 1-2 weeks |
| **M4: Advanced** | Rate limiting, health tracking, replay, graceful shutdown | 1-2 weeks |
| **M5: Polish** | Dashboard, tests, docs, Docker, README | 1-2 weeks |
| **TOTAL** | **Production-grade webhook delivery engine** | **5-10 weeks** |
