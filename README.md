# Jasper

A lightweight, S3-compatible cloud storage server written in Go. Self-host your own object storage with a familiar S3 API — store, retrieve, and manage objects without relying on AWS.

## Features

- **HTTP/3 (QUIC) + HTTP/2 + HTTP/1.1** — dual-stack server with automatic protocol upgrade via `Alt-Svc`
- S3-compatible REST API
- Bucket & object management
- **Public and private objects** — objects are private by default; `x-amz-acl: public-read` makes an object readable without credentials
- **Instant multipart completion** — parts are kept as separate files and served as one logical object (no data copying)
- PostgreSQL-backed metadata storage
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
git clone https://github.com/ArdaEmek/jasper.git
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

5. Apply database migrations

```bash
make migrate
```

6. Run the application

```bash
make run
```

The server will start on `https://localhost:8080` (both TCP for HTTP/1.1+HTTP/2 and UDP for HTTP/3).

### Pull the published Docker image

Pushes to the default branch and version tags publish an image to GitHub Container Registry:

```bash
docker pull ghcr.io/ardaemek/jasper:latest
```

For a private package, authenticate first with a GitHub token that has `read:packages`:

```bash
echo "$CR_PAT" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

The image contains the server binary, migration files, and `dbmate`. Mount your TLS
certificates and provide the database, Redis, and other settings with environment
variables when starting it. The application listens on port `8080` by default. Both
TCP and UDP access are required when HTTP/3 is enabled.

The image also contains the database migrations and `dbmate`. Run migrations as a
separate one-shot task before starting Jasper:

```bash
docker run --rm \
  --env-file jasper.env \
  ghcr.io/ardaemek/jasper:1.0.96 \
  migrate up
```

The same versioned image can then be started normally:

```bash
docker run -d \
  --name jasper \
  --env-file jasper.env \
  -p 8080:8080 \
  -v ./certs:/app/certs:ro \
  ghcr.io/ardaemek/jasper:1.0.96
```

`DATABASE_URL` must be available to the migration task. The server container also
requires `CERTIFICATE_PUBLIC_KEY` and `CERTIFICATE_PRIVATE_KEY`; when using the
example certificate mount, set them to `/app/certs/cert.pem` and
`/app/certs/key.pem`. `STORAGE_DIR` defaults to `/app/data`.

The consuming project should run the migration task successfully before rolling out
the Jasper container; Jasper does not apply migrations automatically during server
startup.

### HTTP/3 Protocol Upgrade

HTTP/3 and TLS are enabled by default. Set `HTTP3_ENABLED=false` to run only a
plain TCP HTTP/1.1 server without loading certificate files.

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
| `HTTP3_ENABLED`          | Enable the HTTP/3 (QUIC) server and `Alt-Svc` advertisement | `true` |
| `STORAGE_DIR`            | Local directory for object storage   | `./data`           |
| `CERTIFICATE_PUBLIC_KEY` | Path to TLS certificate (PEM)        | `./certs/cert.pem` |
| `CERTIFICATE_PRIVATE_KEY`| Path to TLS private key (PEM)        | `./certs/key.pem`  |
| `DB_HOST`                | PostgreSQL host                      | `localhost`        |
| `DB_PORT`                | PostgreSQL port                      | `5432`             |
| `DB_DATABASE`            | Database name                        | `jasper`           |
| `DB_USERNAME`            | Database user                        | —                  |
| `DB_PASSWORD`            | Database password                    | —                  |
| `DB_SCHEMA`              | Database schema                      | `public`           |
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
S3 endpoints require AWS Signature V4 authentication via `Authorization` header or presigned URLs. The one exception is reading a public object (see [Public Objects](#public-objects)).

<details>
<summary><strong>Bucket Operations</strong></summary>

| Method | Endpoint    | Description         |
|--------|-------------|---------------------|
| `GET`  | `/`      | List all buckets owned by the authenticated user |
| `GET`    | `/{bucket}`       | List objects in a bucket (`ListObjectsV2`) |
| `PUT`    | `/{bucket}`       | Create a new bucket |
| `DELETE` | `/{bucket}`       | Delete an empty bucket |

</details>

<details>
<summary><strong>Object Operations</strong></summary>

| Method | Endpoint            | Description                               |
|--------|---------------------|-------------------------------------------|
| `PUT`    | `/{bucket}/{key}` | Upload/overwrite an object (optional `x-amz-acl: public-read`) |
| `GET`    | `/{bucket}/{key}` | Retrieve object (supports `Range` requests; anonymous if the object is public) |
| `HEAD`   | `/{bucket}/{key}` | Get object metadata without downloading (anonymous if the object is public) |
| `DELETE` | `/{bucket}/{key}` | Delete an object |

</details>

<details>
<summary><strong>Multipart Uploads</strong></summary>

> **Note:** Completing a multipart upload does not copy any data. Each uploaded part stays on disk as its own file, and the object is stored as an ordered list of segments that is read back as one continuous object. Completion time therefore does not depend on object size, and range requests that cross part boundaries are supported.

| Method | Endpoint                                       | Description                     |
|--------|------------------------------------------------|---------------------------------|
| `POST` | `/{bucket}/{key}?uploads`                  | Create a multipart upload (optional `x-amz-acl: public-read`) |
| `PUT`  | `/{bucket}/{key}?partNumber=n&uploadId=id` | Upload a part |
| `POST` | `/{bucket}/{key}?uploadId=id`              | Complete a multipart upload |
| `GET`  | `/{bucket}/{key}?uploadId=id`              | List parts of a multipart upload |
| `DELETE` | `/{bucket}/{key}?uploadId=id`              | Abort a multipart upload and remove part files   |
| `POST` | `/{bucket}?delete`                         | Multi-Object Delete (Batch delete objects) |

</details>

#### Range Requests
GET requests support the standard HTTP `Range` header for partial downloads:
```
Range: bytes=0-1023     # First 1024 bytes
Range: bytes=500-       # From byte 500 to end
Range: bytes=-500       # Last 500 bytes
```

The server responds with `206 Partial Content` and the appropriate `Content-Range` header. A single range is supported; malformed or multi-range headers are ignored and the full object is returned.

## Public Objects

Objects are **private by default**. To make an object publicly readable, upload it with the standard `x-amz-acl: public-read` header (or `ACL: "public-read"` in an SDK). `private` is also accepted; any other ACL value returns `NotImplemented`.

```bash
aws s3 cp photo.jpg s3://my-bucket/photo.jpg --acl public-read --endpoint-url https://localhost:8080
```

For multipart uploads, set the ACL on `CreateMultipartUpload`; the finished object inherits it.

How public access behaves:

- **Anonymous access covers `GET` and `HEAD` of a single object only.** Listing, uploading, deleting, and every other operation still require a signed request.
- **Private and missing keys look the same.** An anonymous request for a private object or for a key that doesn't exist returns `AccessDenied`, so keys can't be probed.
- **Overwriting resets visibility.** Uploading over a public key without `public-read` makes it private again, as in S3.
- **Headers on anonymous responses:** `Cache-Control: public, max-age=300`, `X-Content-Type-Options: nosniff`, and `Content-Security-Policy: sandbox`, so uploaded HTML and SVG can't run scripts with your server's privileges. Authenticated responses use `Cache-Control: private, no-store`.
- Changing visibility of an existing object (`PutObjectAcl`) is not implemented yet; re-upload the object with the desired ACL.

## Supported S3 Operations

| Operation | Status | Notes                                  |
|-----------|--------|----------------------------------------|
| **Bucket Operations** | |                                        |
| CreateBucket | ✅ | Supported                              |
| DeleteBucket | ✅ | Supported                              |
| ListBuckets | ✅ | Supported                              |
| GetBucketLocation | ❌ | Multi-location is not supported        |
| **Object Operations** | |                                        |
| PutObject | ✅ | Supported (`private` and `public-read` ACLs) |
| GetObject | ✅ | Supported (anonymous for public objects) |
| HeadObject | ✅ | Supported (anonymous for public objects) |
| DeleteObject | ✅ | Supported (Multiple objects supported) |
| CopyObject | ❌ | Not yet implemented                    |
| **Multipart Upload** | |                                        |
| CreateMultipartUpload | ✅ | Supported (`private` and `public-read` ACLs) |
| UploadPart | ✅ | Supported                              |
| CompleteMultipartUpload | ✅ | Supported (no data copy; parts are stored as segments) |
| ListParts | ✅ | Supported                              |
| AbortMultipartUpload | ✅ | Supported                              |
| **Object Listing** | |                                        |
| ListObjects | ✅ | Supported                              |
| ListObjectsV2 | ✅ | Supported                              |
| **ACL & Permissions** | |                                        |
| PutObjectAcl | ❌ | Not yet implemented                    |
| GetObjectAcl | ❌ | Not yet implemented                    |
| **Tagging** | |                                        |
| PutObjectTagging | ❌ | Not yet implemented                    |
| GetObjectTagging | ❌ | Not yet implemented                    |

## Usage Examples

### Creating a Bucket
```bash
aws s3 mb s3://my-bucket --endpoint-url https://localhost:8080
```

### Uploading an Object
```bash
aws s3 cp myfile.txt s3://my-bucket/myfile.txt --endpoint-url https://localhost:8080
```

### Uploading a Public Object
```bash
aws s3 cp myfile.txt s3://my-bucket/myfile.txt --acl public-read --endpoint-url https://localhost:8080
```

### Downloading an Object
```bash
aws s3 cp s3://my-bucket/myfile.txt ./myfile.txt --endpoint-url https://localhost:8080
```

### Downloading a Public Object Without Credentials
```bash
curl https://localhost:8080/my-bucket/myfile.txt
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
