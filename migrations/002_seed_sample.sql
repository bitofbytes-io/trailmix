-- Sample fixture trails for local development.
-- Distances, elevation, grades, dog rules, and lines are illustrative.
-- They are not a survey of Wake, Durham, or Orange county trails.
-- Every trail row has sample = true and source = 'sample-fixture'.
-- The off-leash row exists so that filter has a fixture; it is not a
-- verified off-leash trail.
--
-- Average grade percent is elevation gain / (miles * 5280) * 100.

-- +goose Up
-- +goose StatementBegin
INSERT INTO parks (id, name, county) VALUES
    ('10000000-0000-4000-8000-000000000001', 'Fred G. Bond Metro Park', 'Wake'),
    ('10000000-0000-4000-8000-000000000002', 'Cary Greenways', 'Wake'),
    ('10000000-0000-4000-8000-000000000003', 'William B. Umstead State Park', 'Wake'),
    ('10000000-0000-4000-8000-000000000004', 'Hemlock Bluffs Nature Preserve', 'Wake'),
    ('10000000-0000-4000-8000-000000000005', 'Shelley Lake Park', 'Wake'),
    ('10000000-0000-4000-8000-000000000006', 'American Tobacco Trail', 'Wake'),
    ('10000000-0000-4000-8000-000000000007', 'Eno River State Park', 'Durham'),
    ('10000000-0000-4000-8000-000000000008', 'Occoneechee Mountain State Natural Area', 'Orange'),
    ('10000000-0000-4000-8000-000000000009', 'Annie Louise Wilkerson, MD Nature Preserve', 'Wake'),
    ('10000000-0000-4000-8000-00000000000a', 'Johnston Mill Nature Preserve', 'Orange');

INSERT INTO trails (
    id, name, park_id, dog_rule, distance_miles, elevation_gain_feet, grade_percent,
    difficulty, geometry, source, confidence, sample
) VALUES
    (
        '20000000-0000-4000-8000-000000000001',
        'Bond Park Lake Trail',
        '10000000-0000-4000-8000-000000000001',
        'allowed_leashed', 2.10, 50, 0.45, 'easy',
        '{"type":"LineString","coordinates":[[-78.818,35.781],[-78.814,35.787],[-78.808,35.785],[-78.811,35.780],[-78.818,35.781]]}'::jsonb,
        'sample-fixture', 0.75, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000002',
        'Black Creek Greenway',
        '10000000-0000-4000-8000-000000000002',
        'allowed_leashed', 3.00, 40, 0.25, 'easy',
        '{"type":"LineString","coordinates":[[-78.832,35.768],[-78.820,35.775],[-78.808,35.784],[-78.798,35.792]]}'::jsonb,
        'sample-fixture', 0.70, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000003',
        'Company Mill Trail',
        '10000000-0000-4000-8000-000000000003',
        'allowed_leashed', 5.80, 420, 1.37, 'moderate',
        '{"type":"LineString","coordinates":[[-78.762,35.832],[-78.748,35.838],[-78.740,35.848],[-78.755,35.852],[-78.762,35.832]]}'::jsonb,
        'sample-fixture', 0.80, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000004',
        'Sycamore Trail',
        '10000000-0000-4000-8000-000000000003',
        'allowed_leashed', 6.80, 520, 1.45, 'moderate',
        '{"type":"LineString","coordinates":[[-78.770,35.860],[-78.758,35.868],[-78.746,35.862],[-78.752,35.850],[-78.770,35.860]]}'::jsonb,
        'sample-fixture', 0.70, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000005',
        'Hemlock Bluffs Loop',
        '10000000-0000-4000-8000-000000000004',
        'prohibited', 1.30, 90, 1.31, 'easy',
        '{"type":"LineString","coordinates":[[-78.785,35.728],[-78.778,35.733],[-78.772,35.729],[-78.780,35.725],[-78.785,35.728]]}'::jsonb,
        'sample-fixture', 0.65, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000006',
        'Shelley Lake Loop',
        '10000000-0000-4000-8000-000000000005',
        'allowed_leashed', 2.10, 30, 0.27, 'easy',
        '{"type":"LineString","coordinates":[[-78.655,35.858],[-78.646,35.866],[-78.638,35.860],[-78.648,35.854],[-78.655,35.858]]}'::jsonb,
        'sample-fixture', 0.70, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000007',
        'American Tobacco Trail — White Oak',
        '10000000-0000-4000-8000-000000000006',
        'allowed_leashed', 4.20, 60, 0.27, 'easy',
        '{"type":"LineString","coordinates":[[-78.845,35.712],[-78.842,35.728],[-78.838,35.744],[-78.834,35.760]]}'::jsonb,
        'sample-fixture', 0.75, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000008',
        'Few''s Ford Loop',
        '10000000-0000-4000-8000-000000000007',
        'allowed_leashed', 3.40, 360, 2.01, 'moderate',
        '{"type":"LineString","coordinates":[[-79.015,36.072],[-79.005,36.078],[-78.996,36.084],[-79.002,36.070],[-79.015,36.072]]}'::jsonb,
        'sample-fixture', 0.70, TRUE
    ),
    (
        '20000000-0000-4000-8000-000000000009',
        'Occoneechee Summit Loop',
        '10000000-0000-4000-8000-000000000008',
        'allowed_leashed', 2.20, 480, 4.13, 'hard',
        '{"type":"LineString","coordinates":[[-79.124,36.055],[-79.116,36.060],[-79.110,36.056],[-79.118,36.052],[-79.124,36.055]]}'::jsonb,
        'sample-fixture', 0.55, TRUE
    ),
    (
        '20000000-0000-4000-8000-00000000000a',
        'Wilkerson Preserve Loop',
        '10000000-0000-4000-8000-000000000009',
        'prohibited', 2.60, 300, 2.19, 'moderate',
        '{"type":"LineString","coordinates":[[-78.570,35.904],[-78.560,35.910],[-78.552,35.906],[-78.562,35.900],[-78.570,35.904]]}'::jsonb,
        'sample-fixture', 0.60, TRUE
    ),
    (
        '20000000-0000-4000-8000-00000000000b',
        'Johnston Mill Creek Trail',
        '10000000-0000-4000-8000-00000000000a',
        'unknown', 2.30, 140, 1.15, 'easy',
        '{"type":"LineString","coordinates":[[-79.058,35.986],[-79.048,35.992],[-79.040,35.988],[-79.050,35.982],[-79.058,35.986]]}'::jsonb,
        'sample-fixture', 0.35, TRUE
    ),
    (
        '20000000-0000-4000-8000-00000000000c',
        'Bond Park Dog Meadow (sample)',
        '10000000-0000-4000-8000-000000000001',
        'allowed_offleash', 0.40, 15, 0.71, 'easy',
        '{"type":"LineString","coordinates":[[-78.806,35.788],[-78.803,35.790],[-78.801,35.788],[-78.804,35.786],[-78.806,35.788]]}'::jsonb,
        'sample-fixture', 0.20, TRUE
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM log_entries
WHERE trail_id IN (SELECT id FROM trails WHERE sample);

DELETE FROM trails WHERE sample;

DELETE FROM parks AS park
WHERE NOT EXISTS (
    SELECT 1 FROM trails WHERE trails.park_id = park.id
);
-- +goose StatementEnd
