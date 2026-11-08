-- migrate:up
CREATE TABLE object_segments
(
    object_id    VARCHAR(27) NOT NULL REFERENCES objects (object_id) ON DELETE CASCADE,
    idx          INT         NOT NULL,
    file_id      TEXT        NOT NULL,
    size         BIGINT      NOT NULL CHECK (size >= 0),
    start_offset BIGINT      NOT NULL CHECK (start_offset >= 0),
    PRIMARY KEY (object_id, idx)
);

CREATE UNIQUE INDEX object_segments_file_id_idx ON object_segments (file_id);

-- existing objects become one segment (their file is named after object_id)
INSERT INTO object_segments (object_id, idx, file_id, size, start_offset)
SELECT object_id, 0, object_id, size_bytes, 0
FROM objects;

-- migrate:down
DROP TABLE object_segments;