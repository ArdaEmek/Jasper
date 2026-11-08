-- migrate:up
ALTER TABLE multipart_upload_parts ADD COLUMN file_id TEXT;

-- backfill existing rows with the old naming scheme so in-flight uploads still resolve
UPDATE multipart_upload_parts
SET file_id = upload_id || '_' || part_number
WHERE file_id IS NULL;

ALTER TABLE multipart_upload_parts ALTER COLUMN file_id SET NOT NULL;

CREATE UNIQUE INDEX multipart_upload_parts_file_id_idx
    ON multipart_upload_parts (file_id);

-- migrate:down
DROP INDEX multipart_upload_parts_file_id_idx;
ALTER TABLE multipart_upload_parts DROP COLUMN file_id;