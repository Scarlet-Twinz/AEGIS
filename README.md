# AEGIS

**A small, concurrent HTTP reverse proxy with the operational features that make edge services useful.** Built entirely with Go's standard library.

```text
client ──► AEGIS :8080 ──► selected upstream
              │
              ├─ path routing
              ├─ per-IP token-bucket limits
              ├─ request IDs + JSON logs
              └─ health and live metrics
```

## Features

- Reverse proxies HTTP traffic with Go's production-grade transport
- Selects upstreams by longest matching path prefix
- Enforces independent, in-memory per-IP token-bucket limits
- Creates or preserves `X-Request-ID` and forwards it upstream
- Emits concise structured JSON request logs, including latency
- Serves `GET /health` and `GET /metrics`
- Drains in-flight requests on `SIGINT`/`SIGTERM`
- Ships with tests and a minimal non-root Docker image

## Run it

Start an upstream service on port 9000, then:

```bash
go run ./cmd/aegis
curl -i http://localhost:8080/anything
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

## Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `AEGIS_LISTEN_ADDR` | `:8080` | Listener address |
| `AEGIS_ROUTES` | `/=http://localhost:9000` | Comma-separated `prefix=upstream` routes |
| `AEGIS_RATE_LIMIT_RPM` | `120` | Tokens replenished per client per minute |
| `AEGIS_RATE_LIMIT_BURST` | `30` | Immediate requests allowed per client |
| `AEGIS_SHUTDOWN_TIMEOUT` | `10s` | Maximum graceful-drain time |
| `AEGIS_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |

For example, direct API and auth paths to different services:

```bash
AEGIS_ROUTES='/api=http://localhost:9000,/auth=http://localhost:9001,/=http://localhost:9002' go run ./cmd/aegis
```

The most specific matching prefix wins. Management endpoints are served by AEGIS itself and are never proxied.

## Docker

```bash
docker build -t aegis .
docker run --rm -p 8080:8080 -e AEGIS_ROUTES=/=http://host.docker.internal:9000 aegis
```

## Verify

```bash
go test ./...
go vet ./...
```

## Deliberate boundaries

This is intentionally a lightweight single-process edge component: limits and metrics reset on restart, there is no TLS termination, and upstream health/circuit breaking are left to a load balancer or a future extension. That keeps the core small enough to understand while retaining real network and concurrency behavior.

## Project Links

- **Repository:** https://github.com/Scarlet-Twinz/AEGIS
- **Author:** Anthony Emmanuella Mmasinachi
- **GitHub:** https://github.com/Scarlet-Twinz
