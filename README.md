# Trail Mix

Trail Mix is a small trail log for hikes around Cary, North Carolina. It is a server-rendered Go application with [templ](https://templ.guide/) pages and a PostGIS database. There is no Vercel deployment. Map tiles come from [OpenFreeMap](https://openfreemap.org/).

The screen follows the wireframe: a random-trail button, a map area, the list of hikes you have logged, and the selected entry with its photo and notes. A separate page filters the catalog.

## What is real

- Trail search filters by distance, difficulty, average grade, park, and dog rule (`allowed_leashed`, `allowed_offleash`, `prohibited`, `unknown`).
- An interactive map. [MapLibre GL JS](https://maplibre.org/) draws the filtered trails on the OpenFreeMap Liberty style. Click a line for a popup that links to the trail. Logged trails are highlighted.
- Trail lines stored as PostGIS `LineString` geometries (SRID 4326), with a trailhead point and a GiST index. Miles are the geodesic length of the line (`ST_Length` on geography). Grade is the stored elevation gain over that length.
- `GET /api/trails.geojson` (the same filters as Find trails) and `GET /api/trails/{id}/geojson`.
- A personal log stored in Postgres: visit date, duration, notes, and a photo.
- A random-trail button that picks one trail from the catalog.
- Goose migrations, including a dozen sample trails. Every sample trail has `sample = true` and `source = sample-fixture`.
- `GET /health` and `GET /ready`.

## What is a placeholder

- Distances, elevation, grades, dog rules, and lines are sample fixtures, not a survey. The mile value is the length of the fixture line. Low-confidence rows say so on the trail page. The off-leash row exists so that filter has a fixture; it is not a verified off-leash trail.
- "Random trail near me" draws from that Cary-area catalog. It does not read the device location.
- There is no login. The app is a single-user local log. Do not publish it on the internet as-is.
- Image push and production deploy are not wired. Run it locally with Docker Compose.

## Run locally with Docker

Docker 24+ with Compose v2 is enough. You do not need Go, Templ, or Tailwind on the host. The image builds those assets, then the migrate service applies the schema and sample trails before the app listens.

```bash
make up
```

`docker compose up --build` does the same thing. Open <http://127.0.0.1:4600>.

The database listens on `127.0.0.1:5432`. The local password is the default `trailmix` (user `trailmix`, database `trailmix`). It is a development default, not a secret to reuse anywhere else. Override `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` together if you change it, and keep the password URL-safe.

`make down` stops the containers and keeps the database volume, including log entries you added. `docker compose down -v` deletes that volume. The next `make up` creates an empty database and seeds the sample trails again.

| Setting | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Yes | PostgreSQL connection string. Compose sets this. |
| `PORT` | No | HTTP port. Defaults to `4600`. |
| `LOG_LEVEL` | No | `debug`, `info`, `warn`, or `error`. Defaults to `info`. |
| `SECURE_COOKIES` | No | `false` for local HTTP. Defaults to `true`, which also sends HSTS. |
| `MAP_STYLE_URL` | No | MapLibre style URL. Defaults to the OpenFreeMap Liberty style. |
| `MAP_ATTRIBUTION` | No | Credit line shown on the map. Defaults to the OpenFreeMap / OpenStreetMap credit. |

The database image is `postgis/postgis:17-3.6-alpine` (PostgreSQL 17 and PostGIS 3.6). A volume created by the older plain `postgres:17-alpine` image usually starts, because the major version matches. If it refuses to start, `docker compose down -v` drops that volume and the log entries in it.

`DATABASE_URL_FILE` and `/run/secrets/trailmix_database_url` are also read, for the same file-secret pattern as the other apps. Do not commit a real URL.

## Run from the source tree

Use this when you want `go run` instead of the app container. Postgres still has to be running. `make up` starts both; to run the binary against Compose's database, start only the database (`docker compose up db`) and apply migrations.

Requirements:

- Go 1.26.11 (the `go` directive in `go.mod`)
- The [Tailwind CSS standalone CLI](https://github.com/tailwindlabs/tailwindcss/releases) v4.3.0 on your `PATH`, named `tailwindcss`
- [Goose](https://github.com/pressly/goose) for `make migrate`
- PostgreSQL 17 with PostGIS 3.6 (the Compose database image)

Templ is run with `go run` from the module version. Generated `*_templ.go` files and `static/styles.css` are gitignored.

```bash
# Linux x64 Tailwind CLI. Confirm the checksum before moving it onto PATH.
curl -fsSL https://github.com/tailwindlabs/tailwindcss/releases/download/v4.3.0/tailwindcss-linux-x64 -o /tmp/tailwindcss
echo "73f0e5459054e5cfaa8ab6f3b940f3fbe0f13cc7fd83bc24e7c655033c203400  /tmp/tailwindcss" | sha256sum --check --strict
chmod +x /tmp/tailwindcss
sudo mv /tmp/tailwindcss /usr/local/bin/tailwindcss

cp local.mk.example local.mk
docker compose up -d db
make migrate
make run
```

`make build` writes `bin/trailmix`. `make test` regenerates Templ and CSS, then runs `go test ./...`.

Repository and migration tests skip unless `TRAILMIX_TEST_DATABASE_URL` is set. They create a schema, migrate it, and drop it. Point the variable at the Compose database:

```bash
export TRAILMIX_TEST_DATABASE_URL='postgres://trailmix:trailmix@127.0.0.1:5432/trailmix?sslmode=disable'
make test
```

CI runs the same tests against PostgreSQL 17 with PostGIS, then `gofmt` / golangci-lint, `go vet`, `govulncheck`, a race run, and `docker build`.

## Map provider

The style URL and the attribution line are defined once, as `DefaultMapStyleURL` and `DefaultMapAttribution` in `internal/config`. The environment variables above override them. MapLibre GL JS 6.16.0 is vendored in `static/vendor/maplibre/` under the 3-clause BSD license; the notice is in the file header.

| | Default |
| --- | --- |
| `MAP_STYLE_URL` | `https://tiles.openfreemap.org/styles/liberty` |
| `MAP_ATTRIBUTION` | `OpenFreeMap © OpenMapTiles Data from OpenStreetMap` |

OpenFreeMap serves that style with no API key. The map data is © OpenStreetMap contributors, via OpenMapTiles. MapLibre also shows any attribution embedded in the style's TileJSON, so the default map credits OpenFreeMap, OpenMapTiles, and OpenStreetMap in the corner as well as in the caption.

To point the map at another MapLibre style (a topo style, or another tile host), set `MAP_STYLE_URL` and `MAP_ATTRIBUTION` to that provider's style and required credit, then restart. No code change. Handlers only pass a GeoJSON URL into `mapview.Renderer`.

A provider that is not a MapLibre style, such as the Google Maps JavaScript API, needs a new implementation of that interface. The GeoJSON endpoints stay the same.

## License

Trail Mix is available under the [MIT License](LICENSE).
