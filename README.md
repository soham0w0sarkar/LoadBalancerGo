# LoadBalancerGo

A lightweight, modular HTTP load balancer written in Go with support for dynamic backend management, health checking, hot-reload configuration, and middleware.

Built as a learning project with a focus on clean architecture, concurrency, and extensibility.

---

## Features

- Round Robin load balancing
- Dynamic backend addition and removal
- Active health checks with automatic failover
- Hot-reload configuration (no restart required)
- Token bucket rate limiting
- Per-backend timeout configuration
- Automatic request retries
- Graceful shutdown
- Modular architecture for adding new algorithms and middleware

---

## Architecture

```
                Client
                   │
                   ▼
             HTTP Server
                   │
                   ▼
          Middleware Chain
                   │
                   ▼
            Reverse Proxy
                   │
                   ▼
      Load Balancing Strategy
                   │
                   ▼
             Backend Pool
          ┌────────┴────────┐
          ▼                 ▼
      Backend 1         Backend 2
```

Project layout:

```text
LoadBalancerGo/
├── cmd/
│   └── lb/
│       └── main.go
├── configs/
│   └── config.yml
└── internal/
    ├── algorithms/
    ├── backend/
    ├── config/
    ├── middleware/
    ├── proxy/
    ├── server/
    └── util/
```

---

## Quick Start

### Clone

```bash
git clone https://github.com/soham0w0sarkar/LoadBalancerGo.git
cd LoadBalancerGo
```

### Build

```bash
go mod download
go build -o loadbalancer cmd/lb/main.go
```

### Run

```bash
./loadbalancer
```

The server starts on the configured port (default: **8080**).

---

## Configuration

Example `configs/config.yml`:

```yaml
server:
  port: 8080

backends:
  - url: http://localhost:8081
    timeout: 10s

  - url: http://localhost:8082
    timeout: 10s

load_balancing:
  strategy: round_robin

middlewares:
  rate_limiter:
    enabled: false
```

Any changes to the configuration file are detected automatically and applied without restarting the server.

---

## How It Works

### Load Balancing

Requests are distributed using the configured balancing strategy.

Currently implemented:

- ✅ Round Robin

Planned:

- Weighted Round Robin
- Least Connections
- Consistent Hashing
- Random
- IP Hash

---

### Health Checks

Each backend is periodically checked through its `/health` endpoint.

Unhealthy backends are automatically removed from rotation and added back once they recover.

---

### Dynamic Backend Management

Configuration changes are applied at runtime.

When backends are added:

- They are created automatically.
- Health checks begin immediately.
- Traffic is routed once healthy.

When backends are removed:

- They stop receiving new requests.
- Existing requests are allowed to finish.
- The backend is removed gracefully.

---

### Rate Limiting

Optional token bucket rate limiting based on the `x-api-key` header.

Configurable options include:

- Bucket size
- Refill rate

---

## Extending

### Add a new load balancing algorithm

Implement:

```go
type Balancer interface {
    Select([]*backend.Backend) (*backend.Backend, error)
}
```

Register your implementation in the algorithm factory and select it through the configuration.

---

### Add middleware

Since the project uses the standard `http.Handler` interface, middleware can be chained naturally.

```go
handler := proxy.NewProxy(pool, balancer)

handler = logging.New(handler)
handler = auth.New(handler)
handler = ratelimiter.New(handler)
```

---

## Project Goals

This project focuses on learning and demonstrating:

- Go concurrency
- Reverse proxies
- HTTP middleware
- Load balancing algorithms
- Configuration hot reloading
- Health monitoring
- Clean architecture
- Interface-driven design

The goal is to build a modular codebase that's easy to extend with new features.

---

## Roadmap

- [ ] Weighted Round Robin
- [ ] Least Connections
- [ ] Consistent Hashing
- [ ] Random Strategy
- [ ] Prometheus metrics
- [ ] HTTPS support
- [ ] Circuit breaker
- [ ] Docker image
- [ ] Kubernetes deployment example

---

## Contributing

Contributions are welcome!

Feel free to open an issue or submit a pull request for:

- New balancing algorithms
- Middleware
- Performance improvements
- Bug fixes
- Documentation

---

## License

This project is licensed under the MIT License.
