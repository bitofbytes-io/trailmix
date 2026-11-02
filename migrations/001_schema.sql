-- +goose Up
-- +goose StatementBegin
CREATE TABLE parks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name       TEXT NOT NULL,
    county     TEXT NOT NULL,
    UNIQUE (name)
);

CREATE TABLE dog_rules (
    code  TEXT PRIMARY KEY,
    label TEXT NOT NULL
);

INSERT INTO dog_rules (code, label) VALUES
    ('allowed_leashed', 'Leashed'),
    ('allowed_offleash', 'Off leash'),
    ('prohibited', 'No dogs'),
    ('unknown', 'Unknown');

-- geometry is a GeoJSON LineString object. PostGIS is intentionally not
-- enabled; a later migration can add a geography column beside this JSON.
CREATE TABLE trails (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name                TEXT NOT NULL,
    park_id             UUID NOT NULL REFERENCES parks (id),
    dog_rule            TEXT NOT NULL REFERENCES dog_rules (code),
    distance_miles      NUMERIC(6, 2) NOT NULL CHECK (distance_miles > 0),
    elevation_gain_feet INTEGER NOT NULL CHECK (elevation_gain_feet >= 0),
    grade_percent       NUMERIC(5, 2) NOT NULL CHECK (grade_percent >= 0),
    difficulty          TEXT NOT NULL CHECK (difficulty IN ('easy', 'moderate', 'hard')),
    geometry            JSONB NOT NULL,
    source              TEXT NOT NULL,
    confidence          NUMERIC(3, 2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    sample              BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT trails_geometry_linestring CHECK (
        jsonb_typeof(geometry) = 'object'
        AND geometry->>'type' = 'LineString'
        AND jsonb_typeof(geometry->'coordinates') = 'array'
    )
);

CREATE INDEX idx_trails_park_id ON trails (park_id);
CREATE INDEX idx_trails_dog_rule ON trails (dog_rule);

CREATE TABLE log_entries (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trail_id           UUID NOT NULL REFERENCES trails (id) ON DELETE CASCADE,
    visited_on         DATE NOT NULL,
    notes              TEXT NOT NULL DEFAULT '',
    duration_minutes  INTEGER CHECK (duration_minutes IS NULL OR (duration_minutes >= 0 AND duration_minutes <= 1440)),
    photo              BYTEA,
    photo_content_type TEXT,
    CONSTRAINT log_entries_photo_pair CHECK (
        (photo IS NULL AND photo_content_type IS NULL)
        OR (photo IS NOT NULL AND photo_content_type IS NOT NULL)
    )
);

CREATE INDEX idx_log_entries_trail_id ON log_entries (trail_id);
CREATE INDEX idx_log_entries_visited_on ON log_entries (visited_on DESC);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_trails_updated_at
    BEFORE UPDATE ON trails
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_log_entries_updated_at
    BEFORE UPDATE ON log_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS log_entries;
DROP TABLE IF EXISTS trails;
DROP TABLE IF EXISTS parks;
DROP TABLE IF EXISTS dog_rules;
DROP FUNCTION IF EXISTS update_updated_at_column();
-- +goose StatementEnd
