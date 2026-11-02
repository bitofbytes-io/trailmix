// Package dbtest opens a throwaway Postgres schema for integration tests.
package dbtest

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bitofbytes-io/trailmix/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Schema creates an empty schema and returns a URL whose search_path points at
// it. The schema is dropped when the test ends. Tests skip when
// TRAILMIX_TEST_DATABASE_URL is unset.
func Schema(t *testing.T) (*pgx.Conn, string, string) {
	t.Helper()
	databaseURL := os.Getenv("TRAILMIX_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TRAILMIX_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})

	schema := "trailmix_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error(err)
		}
	})

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	// public stays on the path so PostGIS, installed in public, resolves
	// while tables live in the throwaway schema.
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	return admin, schema, parsed.String()
}

// Pool migrates a fresh schema and returns a pool connected to it.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, _ := open(t, "", false)
	return pool
}

// PoolInTimeZone is Pool with the session TimeZone set on every connection.
func PoolInTimeZone(t *testing.T, zone string) *pgxpool.Pool {
	t.Helper()
	if zone == "" {
		t.Fatal("time zone is required")
	}
	pool, _ := open(t, zone, false)
	return pool
}

// PoolWithProvider is Pool plus the goose provider, so a test can migrate
// down and back up. The provider's database stays open until the test ends.
func PoolWithProvider(t *testing.T) (*pgxpool.Pool, *goose.Provider) {
	t.Helper()
	return open(t, "", true)
}

func open(t *testing.T, zone string, keepProvider bool) (*pgxpool.Pool, *goose.Provider) {
	t.Helper()
	_, _, databaseURL := Schema(t)
	if zone != "" {
		parsed, err := url.Parse(databaseURL)
		if err != nil {
			t.Fatal(err)
		}
		query := parsed.Query()
		query.Set("TimeZone", zone)
		parsed.RawQuery = query.Encode()
		databaseURL = parsed.String()
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	closeDB := func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}
	if keepProvider {
		t.Cleanup(closeDB)
	} else {
		defer closeDB()
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if !keepProvider {
		return pool, nil
	}
	return pool, provider
}
