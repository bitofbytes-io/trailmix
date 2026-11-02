package repository

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/bitofbytes-io/trailmix/internal/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostGISGeometryMigration(t *testing.T) {
	ctx := context.Background()
	pool, provider := dbtest.PoolWithProvider(t)

	assertPostGISTrails(t, ctx, pool)
	if _, err := provider.DownTo(ctx, 2); err != nil {
		t.Fatal(err)
	}
	assertGeoJSONTrails(t, ctx, pool)
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	assertPostGISTrails(t, ctx, pool)
}

func TestGeometryUpdateRecomputesLengthAndTrailhead(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM trails ORDER BY name LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	tag, err := pool.Exec(ctx, `
		UPDATE trails
		SET geometry = ST_SetSRID(ST_GeomFromGeoJSON('{"type":"LineString","coordinates":[[-78.8,35.0],[-78.8,36.0]]}'), 4326),
		    elevation_gain_feet = 528
		WHERE id = $1`, id)
	if err != nil {
		t.Fatal(err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("updated %d rows", tag.RowsAffected())
	}

	var miles, raw, grade, lon, lat float64
	var gain int
	err = pool.QueryRow(ctx, `
		SELECT distance_miles::float8,
		       (ST_Length(geometry::geography) / 1609.344)::float8,
		       grade_percent::float8,
		       ST_X(trailhead),
		       ST_Y(trailhead),
		       elevation_gain_feet
		FROM trails WHERE id = $1`, id).Scan(&miles, &raw, &grade, &lon, &lat, &gain)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(miles-raw) > 0.01 {
		t.Fatalf("stored miles %v, geodesic %v", miles, raw)
	}
	wantGrade := float64(gain) / (miles * 5280) * 100
	if math.Abs(grade-wantGrade) > 0.01 {
		t.Fatalf("grade %v, want %v", grade, wantGrade)
	}
	if math.Abs(lon+78.8) > 1e-6 || math.Abs(lat-35) > 1e-6 {
		t.Fatalf("trailhead = %v,%v", lon, lat)
	}

	trail, err := NewTrailRepository(pool).GetTrail(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	points, err := trail.Geometry.Points()
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(points[0][0]-trail.Trailhead.Lon) > 1e-6 || math.Abs(points[0][1]-trail.Trailhead.Lat) > 1e-6 {
		t.Fatalf("read trailhead %+v, first point %v", trail.Trailhead, points[0])
	}
}

func assertPostGISTrails(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	var version, extensionSchema string
	if err := pool.QueryRow(ctx, `
		SELECT PostGIS_Version(), n.nspname
		FROM pg_extension e
		JOIN pg_namespace n ON n.oid = e.extnamespace
		WHERE e.extname = 'postgis'
	`).Scan(&version, &extensionSchema); err != nil {
		t.Fatal(err)
	}
	if version == "" || extensionSchema != "public" {
		t.Fatalf("postgis %q in schema %q", version, extensionSchema)
	}

	var n int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM trails
		WHERE ST_GeometryType(geometry) = 'ST_LineString'
		  AND ST_SRID(geometry) = 4326
		  AND ST_GeometryType(trailhead) = 'ST_Point'
		  AND ST_SRID(trailhead) = 4326
		  AND ST_Equals(trailhead, ST_StartPoint(geometry))
		  AND distance_miles = public.trails_distance_miles(geometry)
		  AND grade_percent = public.trails_grade_percent(elevation_gain_feet, distance_miles)
		  AND abs(distance_miles - ST_Length(geometry::geography) / 1609.344) < 0.01
		  AND distance_miles > 0
	`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 12 {
		t.Fatalf("postgis trails = %d, want 12", n)
	}

	for _, name := range []string{"idx_trails_geometry", "idx_trails_trailhead"} {
		var indexDef string
		err := pool.QueryRow(ctx, `
			SELECT indexdef FROM pg_indexes
			WHERE schemaname = current_schema() AND indexname = $1
		`, name).Scan(&indexDef)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(strings.ToLower(indexDef), "gist") {
			t.Fatalf("%s = %s", name, indexDef)
		}
	}

	trails, err := NewTrailRepository(pool).ListTrails(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(trails) != 12 {
		t.Fatalf("listed %d trails", len(trails))
	}
	for _, trail := range trails {
		points, err := trail.Geometry.Points()
		if err != nil {
			t.Fatalf("%s geometry: %v", trail.Name, err)
		}
		if math.Abs(points[0][0]-trail.Trailhead.Lon) > 1e-6 || math.Abs(points[0][1]-trail.Trailhead.Lat) > 1e-6 {
			t.Fatalf("%s trailhead %+v, first point %v", trail.Name, trail.Trailhead, points[0])
		}
	}
}

func assertGeoJSONTrails(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	var dataType string
	if err := pool.QueryRow(ctx, `
		SELECT data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'trails' AND column_name = 'geometry'
	`).Scan(&dataType); err != nil {
		t.Fatal(err)
	}
	if dataType != "jsonb" {
		t.Fatalf("geometry type after down = %s", dataType)
	}
	var n int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM trails
		WHERE jsonb_typeof(geometry) = 'object'
		  AND geometry->>'type' = 'LineString'
		  AND jsonb_typeof(geometry->'coordinates') = 'array'
	`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 12 {
		t.Fatalf("geojson trails = %d", n)
	}
	var trailhead bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'trails' AND column_name = 'trailhead'
		)`).Scan(&trailhead); err != nil {
		t.Fatal(err)
	}
	if trailhead {
		t.Fatal("trailhead column survived down")
	}
}
