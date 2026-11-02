-- Trail geometry moves from a GeoJSON jsonb document to a PostGIS line.
-- Sample rows inserted by 002 are converted in place.
--
-- Miles are the geodesic length of the line (ST_Length on geography).
-- Grade stays the fixture definition: elevation gain / (miles * 5280) * 100.
-- Elevation gain is still the stored fixture; a 2D line cannot produce it.
-- Distance and grade helpers are created in the same schema as trails.
-- Down drops only that schema's copies. PostGIS itself stays in public.

-- +goose Up
-- +goose StatementBegin
-- Parallel tests each enable PostGIS in public. The lock keeps those
-- CREATE EXTENSION calls from colliding. Helper functions are not shared.
SELECT pg_advisory_xact_lock(874231001);

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;

-- Geodesic length in statute miles, rounded to 0.01 and never zero, so the
-- distance_miles check (greater than zero) stays valid for a short line.
-- search_path is pinned so ST_Length resolves to PostGIS in public.
CREATE OR REPLACE FUNCTION trails_distance_miles(line public.geometry)
RETURNS numeric
LANGUAGE sql
STABLE
SET search_path = public
AS $fn$
    SELECT GREATEST(ROUND((ST_Length(line::geography) / 1609.344)::numeric, 2), 0.01::numeric);
$fn$;

CREATE OR REPLACE FUNCTION trails_grade_percent(gain integer, miles numeric)
RETURNS numeric
LANGUAGE sql
IMMUTABLE
AS $fn$
    SELECT LEAST(ROUND((gain::numeric / (miles * 5280) * 100), 2), 999.99::numeric);
$fn$;

ALTER TABLE trails ADD COLUMN geom geometry(LineString, 4326);

UPDATE trails
SET geom = ST_SetSRID(ST_Force2D(ST_GeomFromGeoJSON(geometry::text)), 4326);

DO $chk$
BEGIN
    IF EXISTS (SELECT 1 FROM trails WHERE geom IS NULL OR NOT ST_IsValid(geom)) THEN
        RAISE EXCEPTION 'trail GeoJSON did not convert to a valid LineString';
    END IF;
END
$chk$;

ALTER TABLE trails ALTER COLUMN geom SET NOT NULL;
ALTER TABLE trails DROP CONSTRAINT trails_geometry_linestring;
ALTER TABLE trails DROP COLUMN geometry;
ALTER TABLE trails RENAME COLUMN geom TO geometry;

ALTER TABLE trails ADD COLUMN trailhead geometry(Point, 4326);

-- No search_path pin: the helpers live in this schema, and PostGIS calls
-- are schema-qualified. Fires on every update so a direct write to
-- distance_miles, grade_percent, or trailhead is overwritten.
CREATE OR REPLACE FUNCTION trails_sync_geometry_measures()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $fn$
BEGIN
    NEW.distance_miles := trails_distance_miles(NEW.geometry);
    NEW.grade_percent := trails_grade_percent(NEW.elevation_gain_feet, NEW.distance_miles);
    NEW.trailhead := public.ST_StartPoint(NEW.geometry);
    RETURN NEW;
END;
$fn$;

CREATE TRIGGER trails_sync_geometry_measures
    BEFORE INSERT OR UPDATE
    ON trails
    FOR EACH ROW
    EXECUTE FUNCTION trails_sync_geometry_measures();

UPDATE trails SET geometry = geometry;

ALTER TABLE trails ALTER COLUMN trailhead SET NOT NULL;

DO $chk$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM trails
        WHERE distance_miles <> trails_distance_miles(geometry)
           OR grade_percent <> trails_grade_percent(elevation_gain_feet, distance_miles)
           OR NOT ST_Equals(trailhead, ST_StartPoint(geometry))
           OR ST_SRID(geometry) <> 4326
           OR ST_SRID(trailhead) <> 4326
    ) THEN
        RAISE EXCEPTION 'trail length, grade, or trailhead was not derived from the line';
    END IF;
END
$chk$;

CREATE INDEX idx_trails_geometry ON trails USING GIST (geometry);
CREATE INDEX idx_trails_trailhead ON trails USING GIST (trailhead);

COMMENT ON COLUMN trails.geometry IS 'WGS84 trail centerline. Length and grade are derived from it.';
COMMENT ON COLUMN trails.trailhead IS 'First point of geometry, WGS84.';
COMMENT ON COLUMN trails.distance_miles IS 'Geodesic length of geometry in miles, rounded to 0.01.';
COMMENT ON COLUMN trails.grade_percent IS 'Elevation gain over distance_miles, percent.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop only this schema's helpers. Another schema migrated to 003 has its
-- own copies, and a shared public function would break those schemas.
DO $drop$
BEGIN
    EXECUTE format('DROP TRIGGER IF EXISTS trails_sync_geometry_measures ON %I.trails', current_schema());
    EXECUTE format('DROP FUNCTION IF EXISTS %I.trails_sync_geometry_measures()', current_schema());
    EXECUTE format('DROP FUNCTION IF EXISTS %I.trails_distance_miles(public.geometry)', current_schema());
    EXECUTE format('DROP FUNCTION IF EXISTS %I.trails_grade_percent(integer, numeric)', current_schema());
END
$drop$;

ALTER TABLE trails ADD COLUMN geometry_json JSONB;
UPDATE trails SET geometry_json = ST_AsGeoJSON(geometry)::jsonb;

DROP INDEX IF EXISTS idx_trails_geometry;
DROP INDEX IF EXISTS idx_trails_trailhead;

ALTER TABLE trails DROP COLUMN trailhead;
ALTER TABLE trails DROP COLUMN geometry;
ALTER TABLE trails RENAME COLUMN geometry_json TO geometry;
ALTER TABLE trails ALTER COLUMN geometry SET NOT NULL;

ALTER TABLE trails ADD CONSTRAINT trails_geometry_linestring CHECK (
    jsonb_typeof(geometry) = 'object'
    AND geometry->>'type' = 'LineString'
    AND jsonb_typeof(geometry->'coordinates') = 'array'
);

-- distance_miles and grade_percent stay at the values derived from the
-- line. The original hard-coded fixture numbers are not restored.
-- The PostGIS extension stays installed. It is database-wide, and other
-- schemas may be using it.
-- +goose StatementEnd
