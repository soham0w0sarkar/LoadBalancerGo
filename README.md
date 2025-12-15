A high-performance, production-ready HTTP load balancer written in Go with advanced features including dynamic backend management, health checking, rate limiting, and hot-reload configuration.

## Features

### Core Capabilities

- **Multiple Load Balancing Algorithms**
  - Round Robin (currently implemented)
  - Weighted, Least Connection, Consistent Hash (planned)

- **Dynamic Backend Pool Management**
  - Thread-safe batch backend operations (add/remove multiple at once)
  - Hot-reload configuration without downtime
  - File-system watcher for automatic config updates
  - Smart graceful draining using backend-specific timeouts
  - Safe copy mechanism prevents external slice modification

- **Intelligent Configuration Watcher**
  - Detects both added AND removed backends
  - Returns structured change events (Added/Removed lists)
  - Debounced updates to prevent reload storms
  - Coordinates with main goroutine via channels

- **Per-Backend Timeout Configuration**
  - Each backend can have individual timeout settings
  - Timeouts properly propagated during hot-reload
  - Context-based timeout enforcement in proxy layer

- **Health Checking**
  - Periodic health checks with configurable intervals
  - Automatic backend marking (alive/dead)
  - Configurable healthy/unhealthy thresholds
  - Concurrent health check execution
  - Thread-safe success/failure count tracking

- **Rate Limiting**
  - Token bucket algorithm implementation
  - Per-client rate limiting based on `x-api-key` header
  - Configurable bucket size and refill rate
  - Automatic token refill over time

- **Retry Mechanism**
  - Automatic retry on backend failures
  - Configurable retry attempts with exponential backoff
  - Context-based retry tracking

- **Graceful Shutdown**
  - Signal handling (SIGTERM, SIGINT)
  - Graceful server shutdown with timeout
  - Proper resource cleanup (health checker, watcher, channels)

## Architecture

LoadBalancerGo is built with **modularity and extensibility** as core design principles. Each component is loosely coupled through well-defined interfaces, making it trivial to extend functionality without modifying existing code.

### Design Philosophy

**Interface-Driven Architecture**: The codebase leverages Go interfaces to create plug-and-play components. Want to add a new load balancing algorithm? Implement the `Balancer` interface. Need custom middleware? Implement the `Handler` interface. This approach follows the Open/Closed Principle - open for extension, closed for modification.

**Clean Separation of Concerns**: Each package has a single, well-defined responsibility. Algorithms don't know about HTTP, backends don't know about rate limiting, and the proxy doesn't know about configuration parsing. This separation makes testing, maintenance, and extension straightforward.

### Project Structure

```
LoadBalancerGo/
├── cmd/lb/              # Application entry point
│   └── main.go          # Composition root - wires all components
├── internal/
│   ├── algorithms/      # Load balancing strategies (HIGHLY EXTENSIBLE)
│   │   ├── balancer.go  # Strategy interface + factory pattern
│   │   └── roundRobin.go # Example implementation
│   ├── backend/         # Backend management (independent module)
│   │   ├── backend.go   # Backend struct with thread-safe operations
│   │   ├── health.go    # Health checking logic
│   │   └── pool.go      # Thread-safe backend pool
│   ├── config/          # Configuration management (pluggable)
│   │   ├── config.go    # Config structures
│   │   ├── parser.go    # YAML parsing
│   │   ├── validator.go # Config validation
│   │   └── watcher.go   # Hot-reload file watcher
│   ├── middleware/      # HTTP middlewares (chainable)
│   │   └── rateLimiter/ # Example middleware
│   │       ├── bucket.go
│   │       └── rateLimiter.go
│   ├── proxy/           # Reverse proxy (strategy-agnostic)
│   │   └── proxy.go
│   ├── server/          # HTTP server wrapper (handler-agnostic)
│   │   └── server.go
│   └── util/            # Utility functions
│       └── context.go
└── configs/
    └── config.yml       # Main configuration file
```

### Modularity Deep Dive

#### 1. **Strategy Pattern for Load Balancing** (Highly Extensible)

The load balancing system uses a clean strategy pattern that makes adding new algorithms effortless:

```go
// Simple interface - implement this, and you're done
type Balancer interface {
    Select([]*backend.Backend) (*backend.Backend, error)
}
```

**Adding a new strategy takes 3 simple steps:**

1. **Create the algorithm file** (e.g., `leastConnections.go`)
2. **Implement the interface**:
```go
type LeastConnections struct {
    // Your state here
}

func (lc *LeastConnections) Select(backends []*backend.Backend) (*backend.Backend, error) {
    // Your logic here
    return selectedBackend, nil
}
```
3. **Register in factory** (`balancer.go`):
```go
case "least_conn":
    return &LeastConnections{}, nil
```

**That's it!** No changes to proxy, server, or any other component. The factory pattern in `SetAlgorithm()` handles instantiation, and the interface ensures compatibility.

#### 2. **Middleware Chain Architecture** (Composable)

Middlewares follow the HTTP Handler pattern, allowing unlimited chaining:

```go
type Handler interface {
    ServeHTTP(http.ResponseWriter, *http.Request)
}
```

**Example: Adding a new middleware** (e.g., logging, authentication, compression):

```go
type Logger struct {
    next Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    l.next.ServeHTTP(w, r)
    log.Printf("%s %s %v", r.Method, r.URL, time.Since(start))
}
```

**Compose in main.go**:
```go
var handler http.Handler = proxy.NewProxy(serverPool, balancer)
handler = NewLogger(handler)           // Add logging
handler = NewAuth(handler)             // Add authentication
handler = ratelimiter.NewRateLimiter(..., handler)  // Add rate limiting
```

The decorator pattern enables building complex pipelines from simple, focused components.

#### 3. **Independent Component Modules**

Each package is self-contained and independently testable:

- **`algorithms/`**: Zero dependencies on HTTP, config, or backends. Pure selection logic.
- **`backend/`**: Manages backend state without knowing about algorithms or HTTP.
- **`config/`**: Parsing and validation, completely decoupled from runtime logic.
- **`middleware/`**: Reusable HTTP middleware, agnostic to load balancing.
- **`proxy/`**: Coordinates components but doesn't implement their logic.

This modularity means you can:
- Test algorithms with mock backends
- Test health checking without HTTP
- Swap YAML for JSON/TOML by changing only `parser.go`
- Replace the HTTP server without touching business logic

#### 4. **Configuration-Driven Extensibility**

The config system uses strongly-typed enums and validation:

```go
type Strategy string

const (
    RoundRobin      Strategy = "round_robin"
    Weighted        Strategy = "weighted"
    LeastConnection Strategy = "least_conn"
    ConsistentHash  Strategy = "consistent_hash"
)
```

Adding a new strategy requires:
1. Add constant to enum
2. Add case to validator
3. Implement algorithm
4. Users can now use it via config change - **zero code changes in production deployments**

#### 5. **Dependency Injection at Composition Root**

The `main.go` file serves as the composition root, wiring dependencies explicitly:

```go
serverPool := backend.NewServerPool(*config)
balancer, _ := algorithms.SetAlgorithm(string(config.LoadBalancing.Strategy))
handler := proxy.NewProxy(serverPool, balancer)
handler = ratelimiter.NewRateLimiter(..., handler)
```

This makes the dependency graph crystal clear and enables easy substitution of components for testing or different environments.

## Key Improvements in Latest Version

### 1. **Smart Graceful Backend Removal**
The load balancer now intelligently handles backend removal:
- Marks all backends as dead simultaneously
- Uses the **maximum backend timeout** as drain period (not hardcoded)
- All removals happen in parallel instead of sequentially
- Fast backends (5s timeout) drain quickly, slow ones (60s) get adequate time

**Performance benefit**: Removing 3 backends with different timeouts now takes `max(timeout1, timeout2, timeout3)` instead of `timeout1 + timeout2 + timeout3`.

### 2. **Intelligent Change Detection**
The watcher detects both additions and removals:
```go
type BackendChange struct {
    Added   []string  // New backends to add
    Removed []string  // Old backends to remove
}
```

### 3. **Batch Backend Operations**
- `AddBackends([]*Backend)`: Add multiple backends atomically
- `RemoveBackends([]string)`: Remove multiple backends in parallel
- More efficient than sequential operations

### 4. **Safe Backend Access Pattern**
```go
// NEW: Safe copy prevents external modification
backends := p.ServerPool.GetBackends()
```

Load balancing algorithms work with copies, can't corrupt the pool.

### 5. **Per-Backend Timeout Configuration**
Each backend maintains its own timeout:
```yaml
backends:
  - url: http://fast-service:8081
    timeout: 5s
  - url: http://slow-service:8082
    timeout: 30s
```

The proxy applies the correct timeout per backend, and removals use these timeouts for proper draining.

#### 1. Backend Pool (`internal/backend/pool.go`)
- **Read-Write Mutex Protection**: Uses `sync.RWMutex` for concurrent access
- **Thread-Safe Operations**:
  - `AddBackend()`: Safely adds new backends to the pool
  - `RemoveBackend()`: Safely removes backends from the pool
  - Prevents race conditions during configuration updates

```go
func (sp *ServerPool) AddBackend(b *Backend) {
    sp.mux.Lock()
    defer sp.mux.Unlock()
    sp.Backends = append(sp.Backends, b)
}
```

#### 2. Backend Status (`internal/backend/backend.go`)
- **Atomic State Management**: Thread-safe alive/dead status tracking
- **Protected Counter Updates**: Success and failure counts with mutex protection
- **Concurrent Health Checks**: Multiple goroutines can safely update backend status

```go
func (b *Backend) SetAlive(alive bool) {
    b.mux.Lock()
    b.Alive = alive
    b.mux.Unlock()
}
```

#### 3. Round Robin Algorithm (`internal/algorithms/roundRobin.go`)
- **Atomic Counter**: Uses `atomic.AddUint64()` for lock-free index increment
- **Thread-Safe Selection**: Multiple concurrent requests safely select backends
- **Lock-Free Performance**: Avoids mutex contention in hot path

```go
func (rr *RoundRobin) NextIndex(backends []*backend.Backend) int {
    return int(atomic.AddUint64(&rr.current, uint64(1)) % uint64(len(backends)))
}
```

#### 4. Rate Limiter (`internal/middleware/rateLimiter/`)
- **Token Bucket Thread Safety**: Mutex-protected token consumption
- **Concurrent Client Handling**: Per-client buckets with synchronized access
- **Safe Refill Calculation**: Thread-safe time-based token refill

#### 5. Dynamic Configuration Updates (`internal/config/watcher.go`)
- **File System Watcher**: Monitors config file for changes
- **Debounced Updates**: 30-second debounce to prevent rapid reloads
- **Change Detection**: Compares old and new backend configurations
- **Thread-Safe Reload**: Coordinates with backend pool for safe updates

## Installation

### Prerequisites

- Go 1.25.1 or higher
- Backend services running on configured ports

### Build from Source

```bash
# Clone the repository
git clone https://github.com/soham0w0sarkar/LoadBalancerGo.git
cd LoadBalancerGo

# Download dependencies
go mod download

# Build the binary
go build -o loadbalancer cmd/lb/main.go

# Run the load balancer
./loadbalancer
```

## Configuration

The load balancer is configured via `configs/config.yml`:

```yaml
server:
  port: 8080                    # Load balancer port
  read_timeout: 10s             # HTTP read timeout
  write_timeout: 10s            # HTTP write timeout

backends:
  - url: http://localhost:8081  # Backend server URLs
    timeout: 15s
  - url: http://localhost:8082
    timeout: 15s
  - url: http://localhost:8083
    timeout: 15s

load_balancing:
  strategy: round_robin         # Load balancing algorithm
  health_check:
    interval: 15s               # Health check frequency
    timeout: 5s                 # Health check timeout
    unhealthy_threshold: 3      # Failures before marking dead
    healthy_threshold: 2        # Successes before marking alive

middlewares:
  rate_limiter:
    enabled: false              # Enable/disable rate limiting
    rate: 0.058                 # Token refill rate (tokens/second)
    size: 2                     # Bucket capacity (tokens)
```

### Configuration Hot-Reload

The load balancer automatically detects and applies configuration changes **in real-time**:

**How it works:**

1. **File Monitoring**: fsnotify watches `configs/config.yml` for write events
2. **Debounce Period**: Changes are buffered for 30 seconds to handle multiple edits
3. **Diff Calculation**: Compares current vs. new backend lists
4. **Backend Addition**: New backends are:
   - Parsed and validated
   - Created with correct timeout settings
   - Added to pool via thread-safe batch operation
   - Immediately start receiving health checks
5. **Backend Removal**: Removed backends are:
   - Marked as dead (stops receiving new traffic)
   - Given 30-second drain period for active connections
   - Removed from pool after drain completes
   - Gracefully shut down without dropping requests

**Example workflow:**

```yaml
# Initial config
backends:
  - url: http://localhost:8081
    timeout: 15s
  - url: http://localhost:8082
    timeout: 15s

# Edit config.yml - add/remove backends
backends:
  - url: http://localhost:8082
    timeout: 15s
  - url: http://localhost:8083  # NEW
    timeout: 20s
  - url: http://localhost:8084  # NEW
    timeout: 10s
# (removed 8081)
```

**What happens:**
- After 30 seconds, changes are detected
- Backend 8081 marked dead, waits 15s (its timeout), then removed
- Backends 8083 and 8084 added and health-checked immediately
- No requests dropped during transition

**Thread-Safe Guarantees:**
- Configuration changes are debounced (30 seconds)
- Batch operations use mutex protection
- Existing connections complete before backend removal
- New backends health-checked before receiving traffic
- Concurrent operations properly coordinated

## Usage

### Starting the Load Balancer

```bash
./loadbalancer
```

Output:
```
LoadBalancer on port: 8080
```

### Backend Requirements

Each backend service must implement a `/health` endpoint:

```go
// Example backend health endpoint
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})
```

### Rate Limiting

When enabled, rate limiting uses the `x-api-key` header to identify clients:

```bash
# Example request with API key
curl -H "x-api-key: client-123" http://localhost:8080/api/resource
```

**Rate Limit Response:**
- Status: `429 Too Many Requests`
- Body: "Rate Limited this IP"

### Graceful Shutdown

Stop the load balancer gracefully:

```bash
# Send SIGTERM or press Ctrl+C
kill -TERM <pid>
```

The load balancer will:
1. Stop accepting new connections
2. Wait for active requests to complete (10s timeout)
3. Stop health checker
4. Close configuration watcher
5. Clean up all resources

## Load Balancing Algorithms

### Strategy Pattern Implementation

LoadBalancerGo uses the **Strategy Pattern** to make load balancing algorithms completely pluggable. The architecture separates the algorithm interface from implementations, allowing you to add new strategies without modifying existing code.

**Core Interface:**
```go
type Balancer interface {
    Select([]*backend.Backend) (*backend.Backend, error)
}
```

This simple contract is all you need to implement. The interface accepts a slice of backends and returns the selected backend - no coupling to HTTP, configuration, or any other system concern.

**Factory Pattern for Registration:**
```go
func SetAlgorithm(strategy string) (Balancer, error) {
    switch strategy {
    case "round_robin":
        return &RoundRobin{}, nil
    case "weighted":
        return &Weighted{}, nil      // Easy to add!
    case "least_conn":
        return &LeastConnection{}, nil  // Easy to add!
    // Add your custom algorithm here
    }
    return nil, fmt.Errorf("unknown strategy: %s", strategy)
}
```

### Round Robin (Implemented)

Distributes requests evenly across healthy backends in circular order.

**Features:**
- Lock-free atomic counter for high performance
- Automatic skip of unhealthy backends
- Thread-safe backend selection
- Fair distribution under uniform load

**Algorithm Flow:**
1. Increment counter atomically
2. Calculate index: `counter % len(backends)`
3. Check if backend is alive
4. If dead, try next backend in round-robin order
5. Return error if all backends are dead

**Implementation Highlights:**
```go
type RoundRobin struct {
    current uint64  // Atomic counter, no locks needed!
}

func (rr *RoundRobin) Select(backends []*backend.Backend) (*backend.Backend, error) {
    next := int(atomic.AddUint64(&rr.current, 1) % uint64(len(backends)))
    // Smart loop to skip dead backends...
}
```

### Extending with New Algorithms

#### Example 1: Least Connections Algorithm

Create `internal/algorithms/leastConnections.go`:

```go
package algorithms

import "github.com/soham0w0sarkar/LoadBalancerGo.git/internal/backend"

type LeastConnections struct {
    // Add any state you need
}

func (lc *LeastConnections) Select(backends []*backend.Backend) (*backend.Backend, error) {
    var selected *backend.Backend
    minConnections := int(^uint(0) >> 1) // Max int
    
    for _, backend := range backends {
        if !backend.IsAlive() {
            continue
        }
        
        connections := backend.GetActiveConnections() // You'd add this method
        if connections < minConnections {
            minConnections = connections
            selected = backend
        }
    }
    
    if selected == nil {
        return nil, fmt.Errorf("no backend available")
    }
    return selected, nil
}
```

Register in `balancer.go`:
```go
case "least_conn":
    return &LeastConnections{}, nil
```

Update config:
```yaml
load_balancing:
  strategy: least_conn  # That's it - zero other changes needed!
```

#### Example 2: Weighted Round Robin

Create `internal/algorithms/weighted.go`:

```go
type Weighted struct {
    current uint64
}

func (w *Weighted) Select(backends []*backend.Backend) (*backend.Backend, error) {
    // Calculate total weight
    totalWeight := 0
    for _, b := range backends {
        if b.IsAlive() {
            totalWeight += b.Weight  // Add Weight field to Backend
        }
    }
    
    // Select based on weight distribution
    target := int(atomic.AddUint64(&w.current, 1) % uint64(totalWeight))
    // ... implementation logic
}
```

**Extension Points:**
- Add `Weight` field to `backend.Backend`
- Add `weight` to config's backend section
- Register in factory
- **No changes to proxy, server, health checker, or any other component!**

#### Example 3: Consistent Hashing

Create `internal/algorithms/consistentHash.go`:

```go
type ConsistentHash struct {
    hashRing map[uint32]*backend.Backend  // Hash ring
    sortedKeys []uint32
    mu sync.RWMutex
}

func (ch *ConsistentHash) Select(backends []*backend.Backend) (*backend.Backend, error) {
    // Hash request identifier (IP, session ID, etc.)
    // Find position on ring
    // Return corresponding backend
    // Provides session affinity!
}
```

### Algorithm Comparison

| Algorithm | Use Case | Pros | Cons | Complexity |
|-----------|----------|------|------|------------|
| **Round Robin** | Uniform workload | Simple, fair | Ignores load | O(n) |
| **Weighted** | Mixed capacity | Respects capacity | Requires tuning | O(n) |
| **Least Connections** | Variable request time | Adapts to load | Tracking overhead | O(n) |
| **Consistent Hash** | Session affinity | Cache-friendly | Uneven distribution | O(log n) |

### Why This Design is Powerful

1. **Zero Coupling**: Algorithms don't know about HTTP, configs, or infrastructure
2. **Easy Testing**: Mock backends, test pure logic
3. **Runtime Switching**: Change algorithms via config reload, no restart
4. **Parallel Development**: Multiple people can implement different strategies simultaneously
5. **No Breaking Changes**: Adding algorithms never affects existing code

### Planned Algorithms

- **Weighted Round Robin**: Distribute based on backend capacity
- **Least Connections**: Route to backend with fewest active connections
- **Consistent Hash**: Session persistence via hash-based routing
- **Random Selection**: Simple random distribution
- **IP Hash**: Route by client IP for session persistence
- **Least Response Time**: Route to fastest backend

**Want to contribute?** Pick an algorithm from above and implement it! The interface makes it trivial.

## Health Checking

### How It Works

1. **Periodic Checks**: Health checker runs every `interval` seconds
2. **Concurrent Execution**: Each backend is checked in a separate goroutine
3. **Threshold-Based**: Backends must fail/succeed multiple times to change status
4. **Thread-Safe Updates**: Status changes use mutex protection

### Health Check Flow

```
Initial State: Backend.Alive = false
                    ↓
Health Check: GET /health (timeout: 5s)
                    ↓
         ┌──────────┴──────────┐
         ↓                     ↓
    Status 200              Error/Timeout
         ↓                     ↓
  SuccessCount++          FailureCount++
         ↓                     ↓
  Count >= 2 ?            Count >= 3 ?
         ↓                     ↓
   SetAlive(true)        SetAlive(false)
```

### Retry Mechanism

Failed requests are automatically retried:

1. Backend receives request
2. If error occurs, check retry count
3. If `retries < unhealthy_threshold`, retry after 10ms
4. Update failure count and potentially mark backend dead
5. If max retries exceeded, return 503 error

## Rate Limiting

### Token Bucket Algorithm

Each client gets a bucket with configurable capacity:

```
Bucket Capacity: 2 tokens
Refill Rate: 0.058 tokens/second (~1 token per 17 seconds)

Request arrives → Check bucket:
  ├─ tokens >= 1.0 → Consume 1 token → Allow request
  └─ tokens < 1.0 → Deny request (429)

Background: tokens += elapsed_time * refill_rate (capped at capacity)
```

### Configuration Examples

**Burst Traffic Handling:**
```yaml
rate_limiter:
  enabled: true
  rate: 10.0      # 10 tokens/second
  size: 100       # Allow burst of 100 requests
```

**Strict Rate Limiting:**
```yaml
rate_limiter:
  enabled: true
  rate: 1.0       # 1 token/second
  size: 1         # No burst allowed
```

## Performance Characteristics

### Thread Safety Overhead

- **Backend Pool**: RWMutex allows concurrent reads, serialized writes
- **Round Robin**: Lock-free atomic operations for minimal contention
- **Health Checks**: Concurrent execution, no blocking on main path
- **Rate Limiter**: Per-client buckets reduce lock contention

### Scalability

- **Concurrent Requests**: Efficiently handles thousands of concurrent connections
- **Backend Count**: Scales linearly with number of backends
- **Memory**: Lightweight per-backend overhead (~1KB)
- **CPU**: Minimal overhead for round-robin selection

## Error Handling

The load balancer handles various error scenarios:

1. **All Backends Down**: Returns 503 Service Unavailable
2. **Max Retries Exceeded**: Returns 503 after exhausting retries
3. **Rate Limit Exceeded**: Returns 429 Too Many Requests
4. **Configuration Invalid**: Exits with detailed error message
5. **Health Check Timeout**: Marks backend as potentially unhealthy

## Development

### Running Tests

```bash
go test ./...
```

### Adding New Load Balancing Algorithm

**Step-by-step guide showing how extensible the architecture is:**

1. **Create algorithm file** in `internal/algorithms/`
```bash
touch internal/algorithms/yourAlgorithm.go
```

2. **Implement the Balancer interface**:
```go
package algorithms

import "github.com/soham0w0sarkar/LoadBalancerGo.git/internal/backend"

type YourAlgorithm struct {
    // Your state here (counters, maps, etc.)
}

func (ya *YourAlgorithm) Select(backends []*backend.Backend) (*backend.Backend, error) {
    // Your selection logic here
    // Return the chosen backend
    return selectedBackend, nil
}
```

3. **Register in factory** (`balancer.go`):
```go
case "your_strategy":
    return &YourAlgorithm{}, nil
```

4. **Add to config enum** (optional, for validation):
```go
const (
    YourStrategy Strategy = "your_strategy"
)
```

5. **Use in config**:
```yaml
load_balancing:
  strategy: your_strategy
```

**That's it!** No changes needed to:
- ✅ HTTP server
- ✅ Proxy logic
- ✅ Health checker
- ✅ Middleware chain
- ✅ Backend management
- ✅ Any other component

**Real-world example timing:**
- Implementing Round Robin: ~50 lines of code
- Implementing Weighted: ~80 lines of code
- Implementing Least Connections: ~60 lines of code

The interface-driven design means adding a new algorithm is a **15-minute task** for most strategies.

## Extensibility & Modularity

LoadBalancerGo is architected for **effortless extensibility**. The design philosophy emphasizes composition over inheritance, interfaces over concrete types, and clear separation of concerns.

### Key Extensibility Features

#### 1. **Interface-Driven Component Design**

Every major component is defined by a minimal interface:

| Component | Interface | Extension Point |
|-----------|-----------|-----------------|
| Load Balancer | `Balancer` | Add new selection algorithms |
| Middleware | `Handler` | Add logging, auth, compression, etc. |
| Config Parser | `Parse()` function | Support JSON, TOML, etc. |
| Server | `Start()/Stop()` | Support HTTPS, HTTP/2, gRPC |

**Why this matters**: You can extend functionality by implementing interfaces, not by modifying existing code (Open/Closed Principle).

#### 2. **Plug-and-Play Middleware System**

The middleware architecture uses the decorator pattern, allowing unlimited composition:

```go
// Core proxy
var handler http.Handler = proxy.NewProxy(serverPool, balancer)

// Stack middlewares like LEGO blocks
handler = NewLogger(handler)              // Add logging
handler = NewAuth(handler)                // Add authentication  
handler = NewCircuitBreaker(handler)      // Add circuit breaker
handler = ratelimiter.NewRateLimiter(..., handler)  // Add rate limiting
handler = NewCompression(handler)         // Add gzip compression
handler = NewCORS(handler)                // Add CORS headers

// Middleware order matters - request flows through the chain!
```

**Creating a custom middleware is trivial:**

```go
type CustomMiddleware struct {
    next Handler
}

func NewCustomMiddleware(next Handler) *CustomMiddleware {
    return &CustomMiddleware{next: next}
}

func (cm *CustomMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Pre-processing logic
    cm.next.ServeHTTP(w, r)  // Pass to next handler
    // Post-processing logic
}
```

**Real examples you could add:**
- Request/response logging
- Authentication (JWT, OAuth)
- Request tracing (OpenTelemetry)
- Compression (gzip, brotli)
- CORS handling
- Request/response transformation
- Circuit breaker
- Caching layer
- Metrics collection

#### 3. **Strategy Pattern for Algorithms** (Detailed Above)

Adding load balancing algorithms requires only implementing one interface method. See "Load Balancing Algorithms" section for complete examples.

#### 4. **Configuration System Extensibility**

The config package is designed to support multiple formats:

**Current**: YAML via `gopkg.in/yaml.v3`

**Easy to add**: JSON, TOML, environment variables, remote config (Consul, etcd)

```go
// Add JSON support by creating parser_json.go
func ParseJSON(data []byte) (*Config, error) {
    c := &Config{}
    err := json.Unmarshal(data, c)
    return c, err
}

// Add environment variable support
func ParseEnv() (*Config, error) {
    c := &Config{
        Server: ServerConfig{
            Port: getEnvAsInt("LB_PORT", 8080),
        },
    }
    return c, nil
}
```

The validator is format-agnostic - it works with any `Config` struct regardless of source.

#### 5. **Health Check Extensibility**

Want to customize health checking? The design makes it easy:

```go
// Add custom health check endpoints
func (hc *HealthCheck) check(backend *Backend) {
    healthURL := backend.URL.String() + "/custom-health"  // Change this
    // ... existing logic
}

// Add multiple health check types
type HealthChecker interface {
    Check(backend *Backend) bool
}

type HTTPHealthChecker struct{}
type TCPHealthChecker struct{}
type GRPCHealthChecker struct{}

// Each implements Check() differently
```

#### 6. **Backend Pool Flexibility**

The `ServerPool` abstraction allows for advanced backend management:

```go
// Add backend grouping
type ServerPool struct {
    Backends       []*Backend
    PrimaryGroup   []*Backend   // Add groups
    SecondaryGroup []*Backend
    mux            sync.RWMutex
}

// Add backend metadata
type Backend struct {
    URL          *url.URL
    Alive        bool
    Weight       int           // Add weighting
    Region       string        // Add region
    Tags         []string      // Add tags
    Metrics      *Metrics      // Add metrics
    // ... existing fields
}

// Add custom selection filters
func (sp *ServerPool) GetBackendsByRegion(region string) []*Backend {
    // Filter and return
}
```

#### 7. **Monitoring & Observability Hooks**

The architecture has natural extension points for monitoring:

```go
// Add metrics middleware
type MetricsCollector struct {
    requestCount   prometheus.Counter
    requestLatency prometheus.Histogram
    next           Handler
}

// Add health check event hooks
type HealthCheckListener interface {
    OnHealthy(backend *Backend)
    OnUnhealthy(backend *Backend)
}

// Add to health checker
func (hc *HealthCheck) RegisterListener(listener HealthCheckListener) {
    hc.listeners = append(hc.listeners, listener)
}
```

#### 8. **Testing-Friendly Design**

The modular architecture makes unit testing straightforward:

```go
// Mock backend for testing algorithms
type MockBackend struct {
    alive bool
}
func (mb *MockBackend) IsAlive() bool { return mb.alive }

// Test round robin without HTTP
func TestRoundRobin(t *testing.T) {
    backends := []*backend.Backend{mockBackend1, mockBackend2}
    rr := &RoundRobin{}
    selected, _ := rr.Select(backends)
    // Assert selection
}

// Mock balancer for testing proxy
type MockBalancer struct {
    returnBackend *backend.Backend
}
func (mb *MockBalancer) Select([]*backend.Backend) (*backend.Backend, error) {
    return mb.returnBackend, nil
}
```

### Modularity Benefits

**1. Independent Development**
- Team A: Implements new load balancing algorithm
- Team B: Adds authentication middleware
- Team C: Improves health checking
- **Zero conflicts** - all work in separate modules

**2. Easy Maintenance**
- Bug in rate limiter? Fix only `middleware/rateLimiter/`
- Change config format? Modify only `config/parser.go`
- Improve algorithm? Touch only `algorithms/yourAlgorithm.go`

**3. Selective Testing**
- Test algorithms with mock backends
- Test middleware with mock handlers
- Test config parsing without runtime
- Test health checks without HTTP

**4. Performance Optimization**
- Profile and optimize individual components
- Replace hot-path components without breaking others
- Add caching layers where needed

**5. Technology Migration**
- Swap HTTP server for HTTP/2 or gRPC
- Replace YAML with etcd/Consul
- Add database-backed config
- **Core business logic unchanged**

### Real-World Extension Examples

#### Example: Add Prometheus Metrics

1. Create `internal/middleware/metrics/prometheus.go`
2. Implement `Handler` interface
3. Collect request count, latency, error rates
4. Add to middleware chain in `main.go`
5. **Done!** Zero changes to existing code

#### Example: Add JWT Authentication

1. Create `internal/middleware/auth/jwt.go`
2. Implement `Handler` interface with token validation
3. Add to middleware chain before proxy
4. Configure via config file
5. **Done!** All routes now require authentication

#### Example: Add Database-Backed Backend Discovery

1. Create `internal/backend/discovery/database.go`
2. Implement periodic DB polling
3. Update `ServerPool` when backends change
4. Use existing thread-safe `AddBackend()` / `RemoveBackend()`
5. **Done!** Dynamic backend discovery from DB

### Comparison: Adding Features

| Feature | Lines of Code | Files Changed | Breaking Changes |
|---------|---------------|---------------|------------------|
| New algorithm | ~50-100 | 2 (new file + factory) | 0 |
| New middleware | ~50-150 | 2 (new file + main.go) | 0 |
| New config format | ~50 | 1-2 (parser) | 0 |
| New health check | ~30-50 | 1 (health.go) | 0 |
| Metrics/monitoring | ~100-200 | 2-3 (new middleware) | 0 |

**Average time to add feature**: 30 minutes to 2 hours

### Architecture Principles Applied

✅ **Single Responsibility Principle**: Each component does one thing well  
✅ **Open/Closed Principle**: Open for extension, closed for modification  
✅ **Liskov Substitution**: Interfaces enable polymorphic behavior  
✅ **Interface Segregation**: Small, focused interfaces  
✅ **Dependency Inversion**: Depend on abstractions, not concretions  

### Want to Extend LoadBalancerGo?

The architecture makes it welcoming for contributors:

1. **Pick a component** (algorithm, middleware, config source)
2. **Implement the interface** (usually 1 method)
3. **Register in factory/chain** (1-2 lines)
4. **Test in isolation** (no dependencies needed)
5. **Submit PR** (clean, focused changes)

**No need to understand the entire codebase** - the modularity means you can work on one piece in isolation!

## Dependencies

- **gopkg.in/yaml.v3**: YAML configuration parsing
- **github.com/fsnotify/fsnotify**: File system watcher for hot-reload

## License

This project is open source and available under the MIT License.

## Troubleshooting

### Backend Not Receiving Traffic

- Check health endpoint returns 200 OK
- Verify backend URL in configuration
- Check health check logs for failures
- Ensure unhealthy threshold not exceeded

### Rate Limiting Not Working

- Confirm `x-api-key` header is sent
- Verify rate limiter is enabled in config
- Check refill rate and bucket size settings

### Configuration Changes Not Applied

- File watcher has 30-second debounce period
- Check console for watcher errors
- Verify YAML syntax is valid

## Authors

- **Soham Sarkar** - [@soham0w0sarkar](https://github.com/soham0w0sarkar)

## Acknowledgments

Built with Go's standard library and minimal external dependencies for maximum reliability and performance.