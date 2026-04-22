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

> **Windows / Git Bash Users:** If you receive a format error for the subject name, Git Bash is likely trying to convert the path. Use `//CN=localhost` instead (with two slashes):
> ```bash
> openssl req -x509 -newkey rsa:2048 -nodes -keyout certs/key.pem -out certs/cert.pem -days 365 -subj "//CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"
> ```

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
| `DB_HOST`                | PostgreSQL host                      | `localhost`        |
| `DB_PORT`                | PostgreSQL port                      | `5432`             |
| `DB_DATABASE`            | Database name                        | `jasper`           |
| `DB_USERNAME`            | Database user                        | —                  |
| `DB_PASSWORD`            | Database password                    | —                  |
| `DB_SCHEMA`              | Database schema                      | `public`           |
| `REDIS_ADDR`             | Redis server address                 | —                  |
| `REDIS_PASSWORD`         | Redis password                       | —                  |
| `ADMIN_ENDPOINT_ENABLED` | Enable admin endpoints               | `false`            |
| `ADMIN_ENDPOINT_KEY`     | Admin endpoint authentication token  | —                  |

## API Endpoints

### Admin Endpoints
Admin endpoints require the `ADMIN_ENDPOINT_KEY` in the `Authorization` header.

| Method | Endpoint         | Description        |
|--------|------------------|--------------------|
| `GET`  | `/admin/user`    | Get user by ID (requires JSON body: `{"id": 1}`) |
| `POST` | `/admin/user`    | Create a new user (requires JSON body: `{"username": "user", "email": "user@example.com"}`) |
| `POST` | `/admin/apikey`  | Generate API key for user (requires JSON body: `{"user_id": 1, "permission_level": "read_write"}`) |
| `GET`  | `/admin/apikey`  | Get all API keys for user (requires JSON body: `{"id": 1}`) |

### S3-Compatible Endpoints
S3 endpoints require AWS Signature V4 authentication via `Authorization` header or presigned URLs.

<details>
<summary><strong>Bucket Operations</strong></summary>

| Method | Endpoint    | Description         |
|--------|-------------|---------------------|
| `PUT`  | `/{bucket}` | Create a new bucket |

</details>

<details>
<summary><strong>Object Operations</strong></summary>

| Method | Endpoint            | Description                               |
|--------|---------------------|-------------------------------------------|
| `PUT`  | `/{bucket}/{key}`   | Upload/overwrite an object                |
| `GET`  | `/{bucket}/{key}`   | Retrieve object (supports `Range` requests)|
| `HEAD` | `/{bucket}/{key}`   | Get object metadata without downloading   |

</details>

<details>
<summary><strong>Multipart Uploads</strong></summary>

| Method | Endpoint                                       | Description                     |
|--------|------------------------------------------------|---------------------------------|
| `POST` | `/{bucket}/{key}?uploads`                      | Create a multipart upload       |
| `PUT`  | `/{bucket}/{key}?partNumber=n&uploadId=id`     | Upload a part                   |
| `POST` | `/{bucket}/{key}?uploadId=id`                  | Complete a multipart upload     |
| `GET`  | `/{bucket}/{key}?uploadId=id`                  | List parts of a multipart upload|

</details>

#### Range Requests
GET requests support the standard HTTP `Range` header for partial downloads:
```
Range: bytes=0-1023     # First 1024 bytes
Range: bytes=500-       # From byte 500 to end
```

The server responds with `206 Partial Content` and the appropriate `Content-Range` header.

## Supported S3 Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| **Bucket Operations** | | |
| CreateBucket | ✅ | Put bucket endpoint |
| DeleteBucket | ❌ | Not yet implemented |
| ListBuckets | ❌ | Not yet implemented |
| GetBucketLocation | ❌ | Not yet implemented |
| **Object Operations** | | |
| PutObject | ✅ | Upload/overwrite objects with metadata |
| GetObject | ✅ | Download objects with Range request support |
| HeadObject | ✅ | Get object metadata without body |
| DeleteObject | ❌ | Not yet implemented |
| CopyObject | ❌ | Not yet implemented |
| **Multipart Upload** | | |
| CreateMultipartUpload | ✅ | Supported |
| UploadPart | ✅ | Supported |
| CompleteMultipartUpload | ✅ | Supported (Zero-copy Linux optimized) |
| ListParts | ✅ | Supported |
| AbortMultipartUpload | ❌ | Not yet implemented |
| **Object Listing** | | |
| ListObjects | ❌ | Not yet implemented |
| ListObjectsV2 | ❌ | Not yet implemented |
| **ACL & Permissions** | | |
| PutObjectAcl | ❌ | Not yet implemented |
| GetObjectAcl | ❌ | Not yet implemented |
| **Tagging** | | |
| PutObjectTagging | ❌ | Not yet implemented |
| GetObjectTagging | ❌ | Not yet implemented |

## Usage Examples

### Creating a Bucket
```bash
aws s3 mb s3://my-bucket --endpoint-url https://localhost:8080
```

### Uploading an Object
```bash
aws s3 cp myfile.txt s3://my-bucket/myfile.txt --endpoint-url https://localhost:8080
```

### Downloading an Object
```bash
aws s3 cp s3://my-bucket/myfile.txt ./myfile.txt --endpoint-url https://localhost:8080
```

### Getting Object Metadata (HEAD)
```bash
aws s3api head-object --bucket my-bucket --key myfile.txt --endpoint-url https://localhost:8080
```

### Range Request Example
```bash
curl -H "Range: bytes=0-99" https://localhost:8080/my-bucket/myfile.txt
```

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
