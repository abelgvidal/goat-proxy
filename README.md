# goat-proxy

🐐 A reverse proxy built from scratch in Go.

## Running

``` bash
BACKENDS=localhost:8050 PROXY_PORT=8500 go run main.go
```

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `BACKENDS` | yes | — | Comma-separated list of backend addresses (`host:port`) |
| `PROXY_PORT` | no | `8080` | Port the proxy listens on |
| `BACKEND_TIMEOUT` | no | `2` | Seconds to wait when connecting to a backend before returning 502 |

### Example with multiple backends and custom timeout

``` bash
BACKENDS=localhost:8050,localhost:8051 PROXY_PORT=8500 BACKEND_TIMEOUT=5 go run main.go
```

