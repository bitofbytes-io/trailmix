// Command trailmix-migrate applies the embedded goose migrations.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bitofbytes-io/trailmix/internal/config"
	"github.com/bitofbytes-io/trailmix/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

func main() {
	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx, databaseURL); err != nil {
		log.Fatal(err)
	}
}

// run applies pending migrations and records them in goose_db_version.
func run(ctx context.Context, databaseURL string) error {
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close migration database: %v", err)
		}
	}()
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS, goose.WithSessionLocker(locker))
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return err
	}
	for _, result := range results {
		log.Printf("applied %s in %s", result.Source.Path, result.Duration)
	}
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	log.Printf("database at migration %d", version)
	fmt.Println("migrations applied")
	return nil
}
