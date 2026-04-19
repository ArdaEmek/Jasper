-- migrate:up

CREATE TABLE users
(
    id    SERIAL PRIMARY KEY,
    name  VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NULL
);

CREATE TABLE api_keys
(
    id               SERIAL PRIMARY KEY,
    user_id          INTEGER             NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    access_key       VARCHAR(255) UNIQUE NOT NULL,
    secret_key       VARCHAR(255)        NOT NULL,
    permission_level VARCHAR(50)              DEFAULT 'full_access',
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE buckets
(
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(63) UNIQUE NOT NULL,
    owner_id   INTEGER            NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    region     VARCHAR(50)              DEFAULT 'us-east-1',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_buckets_name ON buckets (name);

CREATE TABLE objects
(
    object_id           VARCHAR(27) PRIMARY KEY,
    bucket_id           INTEGER NOT NULL REFERENCES buckets (id) ON DELETE CASCADE,
    object_key          TEXT    NOT NULL,
    size_bytes          BIGINT  NOT NULL,
    etag                VARCHAR(255),

    content_type        VARCHAR(255),
    content_disposition TEXT,
    content_language    VARCHAR(50),
    custom_metadata     JSONB                    DEFAULT '{}'::jsonb,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (bucket_id, object_key)
);

CREATE INDEX idx_objects_key ON objects (bucket_id, object_key);

-- migrate:down

DROP TABLE IF EXISTS objects;
DROP TABLE IF EXISTS buckets;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;