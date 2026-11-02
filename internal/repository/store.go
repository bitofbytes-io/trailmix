// Package repository is the Postgres data-access layer for trails, parks,
// dog rules, and log entries.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a trail or log entry does not exist.
var ErrNotFound = errors.New("not found")

// Photo is an uploaded image. An empty Data slice with Clear set removes it.
type Photo struct {
	Clear       bool
	ContentType string
	Data        []byte
}

// EntryInput is a new or edited log entry. On update, a nil Photo keeps the
// current image.
type EntryInput struct {
	TrailID         uuid.UUID
	VisitedOn       time.Time
	Notes           string
	DurationMinutes *int
	Photo           *Photo
}

// TrailStore reads the trail catalog.
type TrailStore interface {
	ListTrails(ctx context.Context) ([]model.Trail, error)
	ListParks(ctx context.Context) ([]model.Park, error)
	ListDogRules(ctx context.Context) ([]model.DogRule, error)
	GetTrail(ctx context.Context, id uuid.UUID) (model.Trail, error)
	RandomTrail(ctx context.Context) (model.Trail, error)
}

// LogStore reads and writes the personal trail log.
type LogStore interface {
	ListEntries(ctx context.Context) ([]model.LogEntry, error)
	GetEntry(ctx context.Context, id uuid.UUID) (model.LogEntry, error)
	CreateEntry(ctx context.Context, in EntryInput) (uuid.UUID, error)
	UpdateEntry(ctx context.Context, id uuid.UUID, in EntryInput) error
	DeleteEntry(ctx context.Context, id uuid.UUID) error
	EntryPhoto(ctx context.Context, id uuid.UUID) (contentType string, data []byte, err error)
}

// TrailRepository implements TrailStore.
type TrailRepository struct {
	pool *pgxpool.Pool
}

// NewTrailRepository returns a trail catalog backed by pool.
func NewTrailRepository(pool *pgxpool.Pool) *TrailRepository {
	return &TrailRepository{pool: pool}
}

// LogRepository implements LogStore.
type LogRepository struct {
	pool *pgxpool.Pool
}

// NewLogRepository returns a log backed by pool.
func NewLogRepository(pool *pgxpool.Pool) *LogRepository {
	return &LogRepository{pool: pool}
}

var (
	_ TrailStore = (*TrailRepository)(nil)
	_ LogStore   = (*LogRepository)(nil)
)
