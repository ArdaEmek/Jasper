-- migrate:up
ALTER TABLE objects ALTER COLUMN object_key TYPE TEXT COLLATE "C";
ALTER TABLE multipart_uploads ALTER COLUMN object_key TYPE TEXT COLLATE "C";
ALTER TABLE objects ADD CONSTRAINT objects_key_len CHECK (octet_length(object_key) <= 1024);

DROP INDEX IF EXISTS idx_objects_key;
DROP INDEX IF EXISTS idx_buckets_name;

CREATE INDEX idx_api_keys_user_id ON api_keys (user_id);
CREATE INDEX idx_buckets_owner_id ON buckets (owner_id);
CREATE INDEX idx_multipart_uploads_user_id ON multipart_uploads (user_id);
CREATE INDEX idx_multipart_uploads_created_at ON multipart_uploads (created_at);

ALTER TABLE objects ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE objects ALTER COLUMN etag SET NOT NULL;
ALTER TABLE objects ALTER COLUMN custom_metadata SET NOT NULL;
ALTER TABLE multipart_uploads ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE multipart_uploads ALTER COLUMN custom_metadata SET NOT NULL;
ALTER TABLE multipart_upload_parts ALTER COLUMN created_at SET NOT NULL;

-- migrate:down
ALTER TABLE multipart_upload_parts ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE multipart_uploads ALTER COLUMN custom_metadata DROP NOT NULL;
ALTER TABLE multipart_uploads ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE objects ALTER COLUMN custom_metadata DROP NOT NULL;
ALTER TABLE objects ALTER COLUMN etag DROP NOT NULL;
ALTER TABLE objects ALTER COLUMN created_at DROP NOT NULL;

DROP INDEX IF EXISTS idx_multipart_uploads_created_at;
DROP INDEX IF EXISTS idx_multipart_uploads_user_id;
DROP INDEX IF EXISTS idx_buckets_owner_id;
DROP INDEX IF EXISTS idx_api_keys_user_id;

CREATE INDEX idx_buckets_name ON buckets (name);
CREATE INDEX idx_objects_key ON objects (bucket_id, object_key);

ALTER TABLE objects DROP CONSTRAINT IF EXISTS objects_key_len;

ALTER TABLE multipart_uploads ALTER COLUMN object_key TYPE TEXT COLLATE "default";
ALTER TABLE objects ALTER COLUMN object_key TYPE TEXT COLLATE "default";