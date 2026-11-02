package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const logSelect = `
SELECT e.id, e.visited_on, e.notes, e.duration_minutes, (e.photo IS NOT NULL), e.created_at, e.updated_at,
       t.id, t.name, t.distance_miles::float8, t.elevation_gain_feet, t.grade_percent::float8,
       t.difficulty, t.geometry, t.source, t.confidence::float8, t.sample,
       p.id, p.name, p.county,
       d.code, d.label
FROM log_entries e
JOIN trails t ON t.id = e.trail_id
JOIN parks p ON p.id = t.park_id
JOIN dog_rules d ON d.code = t.dog_rule
`

func scanEntry(row scanner) (model.LogEntry, error) {
	var entry model.LogEntry
	var duration *int32
	var geometry []byte
	var dogCode, difficulty string
	err := row.Scan(
		&entry.ID,
		&entry.VisitedOn,
		&entry.Notes,
		&duration,
		&entry.HasPhoto,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&entry.Trail.ID,
		&entry.Trail.Name,
		&entry.Trail.DistanceMiles,
		&entry.Trail.ElevationGainFeet,
		&entry.Trail.GradePercent,
		&difficulty,
		&geometry,
		&entry.Trail.Source,
		&entry.Trail.Confidence,
		&entry.Trail.Sample,
		&entry.Trail.Park.ID,
		&entry.Trail.Park.Name,
		&entry.Trail.Park.County,
		&dogCode,
		&entry.Trail.DogRule.Label,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.LogEntry{}, ErrNotFound
	}
	if err != nil {
		return model.LogEntry{}, err
	}
	if duration != nil {
		minutes := int(*duration)
		entry.DurationMinutes = &minutes
	}
	entry.Trail.Difficulty = model.Difficulty(difficulty)
	entry.Trail.DogRule.Code = model.DogRuleCode(dogCode)
	if err := decodeGeometry(&entry.Trail, geometry); err != nil {
		return model.LogEntry{}, err
	}
	return entry, nil
}

func decodeGeometry(trail *model.Trail, geometry []byte) error {
	if !trail.Difficulty.Valid() {
		return fmt.Errorf("trail %s has difficulty %q", trail.ID, trail.Difficulty)
	}
	if !trail.DogRule.Code.Valid() {
		return fmt.Errorf("trail %s has dog rule %q", trail.ID, trail.DogRule.Code)
	}
	if err := json.Unmarshal(geometry, &trail.Geometry); err != nil {
		return fmt.Errorf("trail %s geometry: %w", trail.ID, err)
	}
	if _, err := trail.Geometry.Points(); err != nil {
		return fmt.Errorf("trail %s geometry: %w", trail.ID, err)
	}
	return nil
}

// ListEntries returns visits, newest first.
func (r *LogRepository) ListEntries(ctx context.Context) ([]model.LogEntry, error) {
	rows, err := r.pool.Query(ctx, logSelect+` ORDER BY e.visited_on DESC, e.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.LogEntry, error) {
		return scanEntry(row)
	})
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	return entries, nil
}

// GetEntry returns one visit.
func (r *LogRepository) GetEntry(ctx context.Context, id uuid.UUID) (model.LogEntry, error) {
	entry, err := scanEntry(r.pool.QueryRow(ctx, logSelect+` WHERE e.id = $1`, id))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return model.LogEntry{}, err
		}
		return model.LogEntry{}, fmt.Errorf("get entry: %w", err)
	}
	return entry, nil
}

// CreateEntry inserts a visit and returns its id.
func (r *LogRepository) CreateEntry(ctx context.Context, in EntryInput) (uuid.UUID, error) {
	var contentType *string
	var data []byte
	if in.Photo != nil && len(in.Photo.Data) > 0 {
		contentType = &in.Photo.ContentType
		data = in.Photo.Data
	}
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO log_entries (trail_id, visited_on, notes, duration_minutes, photo, photo_content_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		in.TrailID, in.VisitedOn, in.Notes, in.DurationMinutes, data, contentType,
	).Scan(&id)
	if err != nil {
		if isForeignKey(err) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("create entry: %w", err)
	}
	return id, nil
}

const (
	photoKeep    = 0
	photoReplace = 1
	photoClear   = 2
)

// UpdateEntry changes notes, the visit date, and optionally the photo.
func (r *LogRepository) UpdateEntry(ctx context.Context, id uuid.UUID, in EntryInput) error {
	mode := photoKeep
	var contentType *string
	var data []byte
	if in.Photo != nil && in.Photo.Clear {
		mode = photoClear
	} else if in.Photo != nil && len(in.Photo.Data) > 0 {
		mode = photoReplace
		contentType = &in.Photo.ContentType
		data = in.Photo.Data
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE log_entries
		SET visited_on = $2,
		    notes = $3,
		    duration_minutes = $4,
		    photo = CASE $5::int
		        WHEN 1 THEN $6::bytea
		        WHEN 2 THEN NULL
		        ELSE photo
		    END,
		    photo_content_type = CASE $5::int
		        WHEN 1 THEN $7
		        WHEN 2 THEN NULL
		        ELSE photo_content_type
		    END
		WHERE id = $1`,
		id, in.VisitedOn, in.Notes, in.DurationMinutes, mode, data, contentType,
	)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteEntry removes a visit.
func (r *LogRepository) DeleteEntry(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM log_entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EntryPhoto returns the stored image.
func (r *LogRepository) EntryPhoto(ctx context.Context, id uuid.UUID) (string, []byte, error) {
	var contentType *string
	var data []byte
	err := r.pool.QueryRow(ctx, `
		SELECT photo_content_type, photo FROM log_entries WHERE id = $1`, id).Scan(&contentType, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("entry photo: %w", err)
	}
	if contentType == nil || len(data) == 0 {
		return "", nil, ErrNotFound
	}
	return *contentType, data, nil
}

func isForeignKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
