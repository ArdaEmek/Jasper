# Project Jasper

A lightweight, S3-compatible cloud storage server written in Go. Self-host your own object storage with a familiar S3 API — store, retrieve, and manage objects without relying on AWS.

## Features

- S3-compatible REST API
- Bucket & object management
- PostgreSQL-backed metadata storage
- Graceful shutdown support
- CORS middleware
- Docker support for easy deployment

## Getting Started

### Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [Docker](https://www.docker.com/get-started)
- [Make](https://www.gnu.org/software/make/)

### Installation

1. Clone the repository

```bash
git clone https://github.com/yourusername/jasper.git
cd jasper
```

2. Copy the example environment file and configure it

```bash
cp .env.example .env
```

3. Start the database

```bash
make docker-run
```

4. Run the application

```bash
make run
```

The server will start on `http://localhost:8080`.

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `PORT` | Port the server listens on | `8080` |
| `APP_ENV` | Application environment (`local`, `production`) | `local` |
| `BLUEPRINT_DB_HOST` | PostgreSQL host | `localhost` |
| `BLUEPRINT_DB_PORT` | PostgreSQL port | `5432` |
| `BLUEPRINT_DB_DATABASE` | Database name | `blueprint` |
| `BLUEPRINT_DB_USERNAME` | Database user | — |
| `BLUEPRINT_DB_PASSWORD` | Database password | — |

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/` | Homepage / API info |
| `GET` | `/health` | Health check |
| `GET` | `/buckets` | List all buckets |
| `POST` | `/buckets` | Create a bucket |
| `GET` | `/buckets/{bucket}` | List objects in a bucket |
| `PUT` | `/buckets/{bucket}/{key}` | Upload an object |
| `GET` | `/buckets/{bucket}/{key}` | Download an object |
| `DELETE` | `/buckets/{bucket}/{key}` | Delete an object |

## Project Structure

```
jasper/
├── cmd/
│   └── api/
│       └── main.go         # Application entrypoint & graceful shutdown
├── internal/
│   ├── database/
│   │   ├── database.go     # Database service & connection
│   │   └── database_test.go
│   └── server/
│       ├── server.go       # HTTP server setup
│       ├── routes.go       # Route registration
│       ├── middleware.go   # CORS & other middleware
│       └── handlers/
│           ├── handlers.go # Handler struct & endpoint registration
│           └── health.go   # Health check handler
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

## MakeFile

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

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```

## License

MIT License — see [LICENSE](LICENSE) for details.

