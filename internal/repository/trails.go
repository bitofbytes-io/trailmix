package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const trailSelect = `
SELECT t.id, t.name, t.distance_miles::float8, t.elevation_gain_feet, t.grade_percent::float8,
       t.difficulty, ST_AsGeoJSON(t.geometry)::text, ST_X(t.trailhead), ST_Y(t.trailhead),
       t.source, t.confidence::float8, t.sample,
       p.id, p.name, p.county,
       d.code, d.label
FROM trails t
JOIN parks p ON p.id = t.park_id
JOIN dog_rules d ON d.code = t.dog_rule
`

type scanner interface {
	Scan(dest ...any) error
}

func scanTrail(row scanner) (model.Trail, error) {
	var trail model.Trail
	var geometry []byte
	var headLon, headLat float64
	var dogCode string
	var difficulty string
	err := row.Scan(
		&trail.ID,
		&trail.Name,
		&trail.DistanceMiles,
		&trail.ElevationGainFeet,
		&trail.GradePercent,
		&difficulty,
		&geometry,
		&headLon,
		&headLat,
		&trail.Source,
		&trail.Confidence,
		&trail.Sample,
		&trail.Park.ID,
		&trail.Park.Name,
		&trail.Park.County,
		&dogCode,
		&trail.DogRule.Label,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trail{}, ErrNotFound
	}
	if err != nil {
		return model.Trail{}, err
	}
	trail.Difficulty = model.Difficulty(difficulty)
	trail.DogRule.Code = model.DogRuleCode(dogCode)
	trail.Trailhead = model.Point{Lon: headLon, Lat: headLat}
	if err := decodeGeometry(&trail, geometry); err != nil {
		return model.Trail{}, err
	}
	return trail, nil
}

// ListTrails returns the whole catalog, ordered by name.
func (r *TrailRepository) ListTrails(ctx context.Context) ([]model.Trail, error) {
	rows, err := r.pool.Query(ctx, trailSelect+` ORDER BY t.name`)
	if err != nil {
		return nil, fmt.Errorf("list trails: %w", err)
	}
	trails, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Trail, error) {
		return scanTrail(row)
	})
	if err != nil {
		return nil, fmt.Errorf("list trails: %w", err)
	}
	return trails, nil
}

// ListParks returns parks that have at least one trail, by name.
func (r *TrailRepository) ListParks(ctx context.Context) ([]model.Park, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.name, p.county
		FROM parks p
		WHERE EXISTS (SELECT 1 FROM trails t WHERE t.park_id = p.id)
		ORDER BY p.name`)
	if err != nil {
		return nil, fmt.Errorf("list parks: %w", err)
	}
	parks, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Park, error) {
		var park model.Park
		if err := row.Scan(&park.ID, &park.Name, &park.County); err != nil {
			return model.Park{}, err
		}
		return park, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list parks: %w", err)
	}
	return parks, nil
}

// ListDogRules returns the dog-rule lookup table in a stable order.
func (r *TrailRepository) ListDogRules(ctx context.Context) ([]model.DogRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, label FROM dog_rules
		ORDER BY CASE code
			WHEN 'allowed_leashed' THEN 1
			WHEN 'allowed_offleash' THEN 2
			WHEN 'prohibited' THEN 3
			WHEN 'unknown' THEN 4
			ELSE 5
		END`)
	if err != nil {
		return nil, fmt.Errorf("list dog rules: %w", err)
	}
	rules, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.DogRule, error) {
		var rule model.DogRule
		var code string
		if err := row.Scan(&code, &rule.Label); err != nil {
			return model.DogRule{}, err
		}
		rule.Code = model.DogRuleCode(code)
		return rule, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list dog rules: %w", err)
	}
	return rules, nil
}

// GetTrail returns one trail.
func (r *TrailRepository) GetTrail(ctx context.Context, id uuid.UUID) (model.Trail, error) {
	trail, err := scanTrail(r.pool.QueryRow(ctx, trailSelect+` WHERE t.id = $1`, id))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return model.Trail{}, err
		}
		return model.Trail{}, fmt.Errorf("get trail: %w", err)
	}
	return trail, nil
}

// RandomTrail picks one trail from the catalog. The catalog is the set of
// sample hikes near Cary, so this is the "near me" draw until a location
// provider exists.
func (r *TrailRepository) RandomTrail(ctx context.Context) (model.Trail, error) {
	trail, err := scanTrail(r.pool.QueryRow(ctx, trailSelect+` ORDER BY random() LIMIT 1`))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return model.Trail{}, err
		}
		return model.Trail{}, fmt.Errorf("random trail: %w", err)
	}
	return trail, nil
}
