-- migrate:up
CREATE TABLE multipart_uploads
(
    upload_id           VARCHAR(255) PRIMARY KEY,
    bucket_name         VARCHAR(63) NOT NULL REFERENCES buckets (name) ON DELETE CASCADE,
    object_key          TEXT        NOT NULL,
    user_id             INTEGER     NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    content_type        VARCHAR(255),
    content_disposition TEXT,
    content_language    VARCHAR(50),
    custom_metadata     JSONB                    DEFAULT '{}'::jsonb,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_multipart_uploads_key ON multipart_uploads (bucket_name, object_key);

CREATE TABLE multipart_upload_parts
(
    id          SERIAL PRIMARY KEY,
    upload_id   VARCHAR(255) NOT NULL,
    part_number INT          NOT NULL CHECK (part_number > 0 AND part_number <= 10000),
    etag        VARCHAR(255) NOT NULL,
    size        BIGINT       NOT NULL CHECK (size >= 0
) ,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (upload_id, part_number),

    CONSTRAINT fk_upload
        FOREIGN KEY (upload_id)
            REFERENCES multipart_uploads (upload_id)
            ON DELETE CASCADE
);

-- migrate:down

DROP TABLE IF EXISTS multipart_upload_parts;
DROP INDEX IF EXISTS idx_multipart_uploads_key;
DROP TABLE IF EXISTS multipart_uploads;
