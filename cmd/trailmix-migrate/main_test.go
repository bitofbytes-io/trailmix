package main

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bitofbytes-io/trailmix/internal/dbtest"
	"github.com/bitofbytes-io/trailmix/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"
)

func TestRunRequiresDatabaseURL(t *testing.T) {
	if err := run(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("run error = %v", err)
	}
}

func TestRunAppliesSampleSeed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin, schema, databaseURL := dbtest.Schema(t)

	for range 2 {
		if err := run(ctx, databaseURL); err != nil {
			t.Fatal(err)
		}
	}
	assertSeed(ctx, t, admin, schema)

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 0); err != nil {
		t.Fatal(err)
	}
	assertTables(ctx, t, admin, schema, nil)
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	assertSeed(ctx, t, admin, schema)
}

func assertSeed(ctx context.Context, t *testing.T, admin *pgx.Conn, schema string) {
	t.Helper()
	assertTables(ctx, t, admin, schema, []string{"dog_rules", "log_entries", "parks", "trails"})
	identifier := pgx.Identifier{schema}.Sanitize()
	var trails, sampleTrails, dogRules int
	if err := admin.QueryRow(ctx, "SELECT count(*) FROM "+identifier+".trails").Scan(&trails); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, "SELECT count(*) FROM "+identifier+".trails WHERE sample AND source = 'sample-fixture'").Scan(&sampleTrails); err != nil {
		t.Fatal(err)
	}
	if trails != 12 || sampleTrails != 12 {
		t.Fatalf("trails = %d sample = %d, want 12 and 12", trails, sampleTrails)
	}
	if err := admin.QueryRow(ctx, "SELECT count(DISTINCT dog_rule) FROM "+identifier+".trails").Scan(&dogRules); err != nil {
		t.Fatal(err)
	}
	if dogRules != 4 {
		t.Fatalf("distinct dog rules = %d, want 4", dogRules)
	}
}

func assertTables(ctx context.Context, t *testing.T, admin *pgx.Conn, schema string, want []string) {
	t.Helper()
	var tables []string
	if err := admin.QueryRow(ctx, `SELECT COALESCE(array_agg(tablename::text ORDER BY tablename), '{}'::text[])
		FROM pg_tables WHERE schemaname = $1 AND tablename <> 'goose_db_version'`, schema).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables == nil {
		tables = []string{}
	}
	if want == nil {
		want = []string{}
	}
	if !slices.Equal(tables, want) {
		t.Fatalf("tables = %v, want %v", tables, want)
	}
}
