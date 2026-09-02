# Outpost — Engineering Learning Journal

> A living record of architectural decisions, distributed systems concepts, failure modes, and technical clarifications learned while building **Outpost**.

---

## 01. Database Connection Pooling, TCP Sockets & Network Timeouts

**Related Code:**

- [internal/database/database.go](file:///home/larry_mosh/go-stuff/outpost/internal/database/database.go)
- [internal/config/config.go](file:///home/larry_mosh/go-stuff/outpost/internal/config/config.go)

---

### Core Concept: What is a Connection Pool?

When Go speaks to PostgreSQL, it opens a network socket (TCP connection). Opening and authenticating a connection is expensive (DNS resolution, TCP 3-way handshake, TLS handshake, PostgreSQL authentication).

Instead of opening a new connection for every single HTTP request and closing it immediately, a **connection pool (`pgxpool.Pool`)** keeps a warm set of connections open. When a request needs the database, it borrows a connection, runs its query, and immediately returns it to the pool.

---

### Questions Asked & Clarifications

#### 1. "If we didn't set `MaxConns` and Outpost receives 5,000 requests, will the server crash?"

- **Short Answer:** Yes, PostgreSQL will crash or aggressively reject traffic, causing cascading system failure.
- **The Underlying Mechanics:**
  - **Go Goroutines:** Very cheap (~2 KB of memory each). Go can easily spawn 5,000 goroutines without breaking a sweat.
  - **PostgreSQL Connections:** Very expensive. PostgreSQL does **not** use threads or goroutines; it forks a separate **operating system process** for every connection. Each process consumes **5 MB to 10 MB of RAM**.
  - **Default Limit:** PostgreSQL comes with a default ceiling of `max_connections = 100`.
  - **The Failure:** If 5,000 requests hit Outpost simultaneously with no pool limit:
    1. Connection #101 gets rejected with: `FATAL: sorry, too many clients already`.
    2. If someone configured Postgres to allow 5,000 connections, Postgres would consume ~25 GB to 50 GB of RAM, triggering the Linux kernel's **OOM (Out Of Memory) Killer** to forcefully terminate the Postgres server (`kill -9`).
    3. The Go API would throw cascading 500 errors across all endpoints.
- **How `MaxConns: 25` Protects Us:**
  - No matter how high incoming traffic spikes, Outpost will never open more than 25 connections to PostgreSQL.
  - The extra requests wait in an in-memory queue inside `pgxpool`. Because DB queries are sub-millisecond, connections get recycled rapidly and requests are served cleanly.

---

#### 2. "What is `MaxConnLifetime`? Does it kill an active query if it runs past 1 hour?"

- **Short Answer:** **No, it never kills an active query.** It is a _retirement age_, not a query timeout.
- **How it works:**
  - If a connection is running a query when the 1-hour mark hits, `pgxpool` lets the query finish completely.
  - When the query finishes and returns the connection to the pool, `pgxpool` checks its age. If it is older than 1 hour, it gracefully closes it and opens a fresh one when needed.
- **Why do we need this? (The 3 Everyday Reasons):**
  1. **Database Failover (The Database Moved):** In the cloud (like AWS RDS), if a database server fails or restarts, it moves to a backup machine with a new IP address. If connections stayed open forever, Outpost would keep trying to talk to the dead IP address. Retiring connections periodically forces Outpost to re-resolve DNS and connect to the new live server.
  2. **Silent Disconnects (Firewalls & Routers):** Cloud routers and firewalls quietly drop idle or long-lived connections without telling your Go app. Reconnecting on our own schedule prevents mysterious `broken pipe` or `connection reset by peer` errors.
  3. **Memory Cleanup:** PostgreSQL backend worker processes accumulate temporary scratchpad memory from queries over time. Closing the connection lets Postgres release that memory.
- **Analogy (The Phone Call):**
  - Keeping a connection open is like staying on a phone call.
  - If your friend moves to a new house with a new phone number while you are still holding the phone to your ear, you'll be talking into an empty line.
  - Hanging up periodically and redialing from your address book ensures you are always speaking to the right house.

---

#### 3. "What does `MaxConnIdleTime` mean?"

- **Short Answer:** "Idle" means a connection is sitting in the pool doing nothing because there are not enough incoming requests to use it.
- **The Scenario:**
  - Outpost normally has low traffic and keeps **5 connections** open (`MinConns: 5`).
  - A flash sale brings a traffic spike at 12:00 PM. The pool grows to **20 connections** to handle the load.
  - At 12:05 PM, the spike is over. You now have 15 extra connections sitting in the pool doing nothing.
  - `MaxConnIdleTime: 30 * time.Minute` tells the pool: _"If any extra connection sits completely unused for 30 minutes, close it to free up resources on Postgres."_
  - It will never close below `MinConns: 5`.
- **Analogy (Supermarket Checkout Lanes):**
  - `MaxConns: 25` = 25 total checkout lanes in the supermarket.
  - `MinConns: 5` = 5 lanes stay open even late at night with few shoppers.
  - `MaxConnIdleTime: 30m` = Extra lanes open during a rush; if a lane sees zero shoppers for 30 minutes, the cashier closes the lane and goes home to save power.

---

#### 4. "How does `context.WithTimeout(ctx, 5*time.Second)` work, and why would Go freeze without it?"

- **Why networks don't fail immediately:**
  - On the internet, data packets are frequently delayed by Wi-Fi or router congestion.
  - When your computer sends a packet to Postgres and hears nothing back, the operating system kernel does not immediately give up. It assumes: _"Maybe the wire was busy; let me retry."_
  - The operating system enters **TCP Retransmission**, waiting 1s, 3s, 7s, 15s, 30s... and can keep trying for **over 2 full minutes** before giving up!
- **The Frozen Terminal:**
  - Without a context timeout, `pool.Ping(ctx)` freezes on that line. In your terminal, you see a blinking cursor `_` with no logs and no errors for minutes.
- **How `context.WithTimeout` fixes it:**
  - `pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)` creates a 5-second egg timer strapped to the context.
  - If Postgres does not answer within 5 seconds, `pingCtx` cancels itself and `pool.Ping()` immediately aborts with `context deadline exceeded`.
  - `defer cancel()` stops the timer the moment the function finishes, preventing memory leaks if the call finishes quickly.
  - The application **fails fast** on startup with a clear error instead of hanging in limbo.

---

## 02. Host vs Container Port Mapping (`5433:5432`)

**Related Code:**

- [docker-compose.yml](file:///home/larry_mosh/go-stuff/outpost/docker-compose.yml)
- [internal/config/config.go](file:///home/larry_mosh/go-stuff/outpost/internal/config/config.go)

### The Problem

When running `docker compose up -d`, Docker failed with:
`failed to bind host port 0.0.0.0:5432/tcp: address already in use`

### Why This Happened

The host Linux machine was already running a native PostgreSQL service on port `5432`. Only one process can bind to a port on the host operating system at a time.

### The Solution: Port Translation

In `docker-compose.yml`, we mapped `"5433:5432"`:
$$\text{Host Port (5433)} \longrightarrow \text{Container Port (5432)}$$

- **Inside Docker:** PostgreSQL continues running on its default port `5432`.
- **Outside Docker:** Our host machine reaches the container on `localhost:5433`.
- **Benefit:** Zero conflict with the host machine's existing PostgreSQL service.

---

## 03. Fail-Fast Configuration & Credential Masking

**Related Code:**

- [internal/config/config.go](file:///home/larry_mosh/go-stuff/outpost/internal/config/config.go)
- [.env.example](file:///home/larry_mosh/go-stuff/outpost/.env.example)

### Key Decisions

1. **Single Typed Struct:** Avoid scattering raw `os.Getenv("SOME_KEY")` calls throughout business logic. Load everything once into a strongly typed `Config` struct.
2. **`Validate()` on Startup:** If a required setting is missing (e.g. empty database host or port), crash immediately at boot with a helpful error. Never let an improperly configured server start accepting user traffic.
3. **`MaskedConnectionString()`:** Production logs are forwarded to central monitoring services (e.g. Datadog, Grafana, CloudWatch). Printing connection strings with raw passwords is a major security vulnerability. Masking passwords (`postgres:******@localhost:5433/outpost`) ensures operational visibility without leaking secrets.

---

## 04. Entry Point Architecture & Graceful Shutdown

**Related Code:**

- [cmd/api/main.go](file:///home/larry_mosh/go-stuff/outpost/cmd/api/main.go)
- [internal/logger/logger.go](file:///home/larry_mosh/go-stuff/outpost/internal/logger/logger.go)

### Core Architectural Principles

1. **Dependency Injection in `cmd/api/main.go`:**
   - `main.go` is the orchestrator. It instantiates the logger, config, and database pool, then injects them into services and handlers.
   - Internal packages (`internal/service`, `internal/repository`) do not construct their own dependencies or touch global singletons.
2. **Goroutine-Driven Server Execution:**
   - `srv.ListenAndServe()` is a blocking call. If run on the main goroutine, execution stops there, making it impossible to listen for termination signals.
   - By launching `srv.ListenAndServe()` in a background goroutine (`go func() { ... }()`), the main goroutine stays free to listen for OS signals.
3. **Graceful Shutdown (Handling `SIGINT` & `SIGTERM`):**
   - When you press `Ctrl+C` (`SIGINT`) or Kubernetes/Docker stops a container (`SIGTERM`), you must not abruptly terminate the process while requests are in flight.
   - `signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)` intercepts the signal.
   - `srv.Shutdown(shutdownCtx)` stops accepting _new_ HTTP requests while giving existing in-flight requests up to 5 seconds to finish responding.
   - `defer dbPool.Close()` guarantees database connections are drained and closed cleanly.

---

## 05. HTTP Server Decoupling & Configurable Gin Modes

**Related Code:**

- [internal/server/server.go](file:///home/larry_mosh/go-stuff/outpost/internal/server/server.go)
- [internal/config/config.go](file:///home/larry_mosh/go-stuff/outpost/internal/config/config.go)
- [cmd/api/main.go](file:///home/larry_mosh/go-stuff/outpost/cmd/api/main.go)

### The Architectural Problem

Putting Gin router configuration, route groups (`/api/v1/...`), CORS middleware, and HTTP server lifecycle settings directly inside `main.go` creates a bloated file. As routes grow, `main.go` becomes unmaintainable.

### The Solution: Dedicated Server Package

1. **Encapsulated Router & Lifecycle:**  
   `internal/server/server.go` manages the Gin engine, global middlewares (Recovery, CORS), and wraps the `*http.Server` with clean `Start()` and `Shutdown(ctx)` methods.
2. **Avoiding the "God Object" Trap:**  
   Instead of attaching 30+ endpoint handler functions directly to the `Server` struct, `Server` acts purely as a routing multiplexer. Domain handlers (`ApplicationHandler`, `EndpointHandler`) remain isolated in `internal/handler/` and are injected into the server.
3. **Environment-Driven `GIN_MODE`:**  
   Rather than hardcoding `gin.SetMode()`, we bind it to typed configuration (`cfg.Server.GinMode`) loaded from `GIN_MODE` with a fallback to `"debug"`. This allows seamless switching between debug logging locally and release performance in production.
4. **Lean `main.go`:**  
   `main.go` is reduced to an ultra-clean, readable orchestrator (~60 lines) whose only responsibility is wiring dependencies and waiting for OS shutdown signals.

---

## 06. Router vs. Server Separation & In-Memory HTTP Testing

**Related Code:**

- [internal/router/router.go](file:///home/larry_mosh/go-stuff/outpost/internal/router/router.go)
- [internal/server/server.go](file:///home/larry_mosh/go-stuff/outpost/internal/server/server.go)
- [internal/router/router_test.go](file:///home/larry_mosh/go-stuff/outpost/internal/router/router_test.go)

### The Concept
Separating the **Router** (`internal/router/router.go`) from the **Server** (`internal/server/server.go`):
- **`router.New(cfg)`:** Builds and returns a `*gin.Engine`. It defines routes, route groups, and middlewares.
- **`server.New(cfg, handler)`:** Takes that `*gin.Engine` (which implements `http.Handler`) and handles TCP listening, timeouts, and OS shutdown signals.

### Why This Is a Testing Superpower
- Without this separation, every test must start a live `*http.Server`, bind to a real TCP port (risking port conflicts), and send real network requests.
- With this separation, unit tests can call `router.New()` and use Go's standard `httptest.NewRecorder()`:
  ```go
  r := router.New(cfg)
  rec := httptest.NewRecorder()
  req, _ := http.NewRequest("GET", "/health", nil)
  r.ServeHTTP(rec, req) // In-memory execution in 0.01s!
  ```
- No real network sockets are opened, tests run in parallel with zero port collision risk, and test suites execute in milliseconds.

---

## 07. Buffered Channels & OS Signal Traps (`signal.Notify`)

**Related Code:**

- [cmd/api/main.go](file:///home/larry_mosh/go-stuff/outpost/cmd/api/main.go)

### The Question Asked
*"What if we didn't make it buffered and just said `make(chan os.Signal)`? How is it different from a buffered channel?"*

### The Core Rule of Go Channels
- **Receiving from an EMPTY channel (`sig := <-quit`) ALWAYS pauses/blocks**, whether buffered or unbuffered.
- Buffering only affects the **sender**.

### Why Unbuffered Channels Break With `signal.Notify`
- Per Go's official documentation: `signal.Notify` **does not block** when sending. If the channel is not ready to receive at the exact instant the signal fires, **the signal is silently discarded!**
- With an unbuffered channel (`make(chan os.Signal)`), if the main goroutine is even a fraction of a microsecond busy during a garbage collection pause or thread context switch when you press `Ctrl+C`, the signal is dropped. Your server ignores `Ctrl+C` and refuses to terminate!
- With a 1-slot buffered channel (`make(chan os.Signal, 1)`), the signal is safely placed into the mailbox without waiting. Zero dropped signals.

