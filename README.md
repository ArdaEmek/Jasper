# Project Jasper

A lightweight, S3-compatible cloud storage server written in Go. Self-host your own object storage with a familiar S3 API — store, retrieve, and manage objects without relying on AWS.

## Features

- **HTTP/3 (QUIC) + HTTP/2 + HTTP/1.1** — dual-stack server with automatic protocol upgrade via `Alt-Svc`
- S3-compatible REST API
- Bucket & object management
- PostgreSQL-backed metadata storage
- Redis caching layer
- TLS 1.3 enforced
- Token-based admin authentication
- CORS middleware
- Graceful shutdown support
- Docker support for easy deployment

## Getting Started

### Prerequisites

- [Go 1.25+](https://golang.org/dl/)
- [Docker](https://www.docker.com/get-started) (optional, for containerized deployment)
- [Make](https://www.gnu.org/software/make/)
- [OpenSSL](https://www.openssl.org/) (for generating TLS certificates)

### Installation

1. Clone the repository

```bash
git clone https://github.com/yourusername/jasper.git
cd jasper
```

2. Copy the example environment file and configure it

```bash
cp env.example .env
```

3. Generate TLS certificates (required)

```bash
openssl req -x509 -newkey rsa:2048 -nodes -keyout certs/key.pem -out certs/cert.pem -days 365 -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"
```

> **Note:** For production, use a certificate from a trusted CA (e.g., Let's Encrypt). For local development, you can trust the self-signed cert by adding it to your OS certificate store.

4. Start the database (optional — if using a local PostgreSQL instance)

```bash
make docker-run
```

5. Run the application

```bash
make run
```

The server will start on `https://localhost:8080` (both TCP for HTTP/1.1+HTTP/2 and UDP for HTTP/3).

### HTTP/3 Protocol Upgrade

Browsers discover HTTP/3 automatically:

1. First request connects via TCP (HTTP/1.1 or HTTP/2 over TLS)
2. Server responds with the `Alt-Svc: h3=":8080"` header
3. Subsequent requests upgrade to HTTP/3 (QUIC over UDP)

> **Chrome + self-signed certs:** Chrome won't upgrade to HTTP/3 with untrusted certificates. Either add the cert to your OS trust store or launch Chrome with `--origin-to-force-quic-on=localhost:8080`.

## Environment Variables

| Variable                 | Description                          | Default            |
|--------------------------|--------------------------------------|--------------------|
| `APP_ENV`                | Application environment (`dev`, `development`, `local`, `production`) | `dev`    |
| `PORT`                   | Port the server listens on (TCP + UDP) | `8080`           |
| `STORAGE_DIR`            | Local directory for object storage   | `./data`           |
| `CERTIFICATE_PUBLIC_KEY` | Path to TLS certificate (PEM)        | `./certs/cert.pem` |
| `CERTIFICATE_PRIVATE_KEY`| Path to TLS private key (PEM)        | `./certs/key.pem`  |
| `JASPER_DB_HOST`         | PostgreSQL host                      | `localhost`        |
| `JASPER_DB_PORT`         | PostgreSQL port                      | `5432`             |
| `JASPER_DB_DATABASE`     | Database name                        | `jasper`           |
| `JASPER_DB_USERNAME`     | Database user                        | —                  |
| `JASPER_DB_PASSWORD`     | Database password                    | —                  |
| `JASPER_DB_SCHEMA`       | Database schema                      | `public`           |
| `REDIS_ADDR`             | Redis server address                 | —                  |
| `REDIS_PASSWORD`         | Redis password                       | —                  |

## API Endpoints

| Method | Endpoint         | Auth     | Description        |
|--------|------------------|----------|--------------------|
| `GET`  | `/`              | No       | Homepage / API info |
| `GET`  | `/admin/user`    | Token    | Get user by ID     |
| `POST` | `/admin/user`    | Token    | Create a user      |

Admin endpoints require an `Authorization` header with a valid token.

## Makefile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```

Create DB container
```bash
make docker-run
```

Shutdown DB container
```bash
make docker-down
```

DB integration tests
```bash
make itest
```

Live reload the application
```bash
make watch
```

Run the test suite
```bash
make test
```

Clean up binary from the last build
```bash
make clean
```

## License

MIT License — see [LICENSE](LICENSE) for details.

