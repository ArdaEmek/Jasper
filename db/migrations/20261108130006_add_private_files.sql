-- migrate:up
ALTER TABLE objects ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE multipart_uploads ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT false;

-- migrate:down
ALTER TABLE objects DROP COLUMN is_public;
ALTER TABLE multipart_uploads DROP COLUMN is_public;