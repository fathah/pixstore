# PixStore

Image storage and search service built with Go and PostgreSQL. Uploads are checked for duplicates using SHA256, originals are kept private, and public product images and thumbnails are converted to WebP. Search uses PostgreSQL `pg_trgm` and `unaccent`.

## Build and publish

Every push to the `release` branch runs `.github/workflows/deploy.yaml`. The workflow tests the Go code, builds the Docker image for Linux AMD64 and ARM64, and publishes:

```text
ghcr.io/fathah/pixstore:latest
```

The image name is derived from the GitHub repository, in lowercase. The workflow uses `GITHUB_TOKEN` with `packages: write`; no server SSH credentials or runtime secrets are needed in GitHub Actions.

## Deploy on a server

Install Docker Engine with the Compose plugin. After the first successful image publication, copy **only `compose.yaml` and `.env.example`** into a directory on the server. A source checkout, Dockerfile, Caddyfile, and migration files are not needed there.

```bash
cp .env.example .env
```

Edit `.env` and replace `POSTGRES_PASSWORD`, `ADMIN_SECRET`, and `API_KEY` with three different secrets. Generate each value with `openssl rand -hex 32`. Keep the database password URL-safe because Compose uses it in the database connection URL.

If the GHCR package is private, log in on the server with a personal access token that has `read:packages`:

```bash
docker login ghcr.io -u fathah
```

Paste the token at the password prompt. Public packages do not need a registry login.

```bash
docker compose pull
docker compose up -d --wait
curl --fail http://localhost:8081/health
```

Compose runs the published image and PostgreSQL; it has no `build:` configuration. PixStore waits for PostgreSQL, applies its embedded initial schema in a transaction, and serves the API and public images on port 8080 inside the container. `PORT` in `.env` controls the server's exposed port (8081 by default). The database has no exposed server port.

Two named volumes persist the database and uploaded images across container replacements. `docker compose down` preserves them. Keep the same deployment directory or Compose project name when updating, and back up both volumes. Changing `POSTGRES_PASSWORD` in `.env` does not update the password of an existing database; rotate it in PostgreSQL first.

For HTTPS, point an existing reverse proxy at the configured server port. Set `BIND_ADDRESS=127.0.0.1` when that proxy runs directly on the same host.

## Update after another release push

Wait for GitHub Actions to publish the new `latest` image, then run in the same server directory:

```bash
docker compose pull
docker compose up -d --wait
```

The workflow publishes the image; these server commands deploy it. Check status and logs with:

```bash
docker compose ps
docker compose logs --tail=100 pixstore
```

## API

### Upload

`POST /api/images/upload` requires `X-Admin-Secret` and accepts JPEG, PNG, and WebP. The multipart request limit is 10 MB. `product_name` is required; `category` and comma-separated `tags` are optional.

```bash
curl http://localhost:8081/api/images/upload \
  -H "X-Admin-Secret: YOUR_ADMIN_SECRET" \
  -F "image=@/path/to/image.jpg" \
  -F "product_name=Cool T-Shirt" \
  -F "category=apparel" \
  -F "tags=shirt,cotton,summer"
```

```json
{
  "id": "uuid...",
  "thumbnail_url": "/thumb/hash.webp",
  "image_url": "/600/hash.webp"
}
```

Product images fit within 600×600 pixels and thumbnails within 200×200, retaining their aspect ratio. Public files are served at `/600/` and `/thumb/`; `/img/` is an alias for product images. Original uploads and directory listings are not exposed.

### Search

`GET /api/images/search` requires `x-api-key`. `q` is required; `limit` defaults to 20 and can be set between 1 and 100.

```bash
curl -G http://localhost:8081/api/images/search \
  -H "x-api-key: YOUR_API_KEY" \
  --data-urlencode "q=tshirt"
```

```json
[
  {
    "id": "uuid...",
    "thumbnail_url": "/thumb/hash.webp",
    "image_url": "/600/hash.webp"
  }
]
```

### Health

`GET /health` returns `200 OK` when PostgreSQL is reachable and `503` when it is unavailable. Docker uses this endpoint for the container health check.

## Local development

Use Go 1.24.5 or newer, a C compiler (the WebP encoder uses CGO), and PostgreSQL 14 or newer. Set these environment variables before starting the server:

```bash
export PORT=8080
export DATABASE_URL='postgres://USER:PASSWORD@localhost:5432/pixstore?sslmode=disable'
export ADMIN_SECRET='YOUR_ADMIN_SECRET'
export API_KEY='YOUR_API_KEY'
export STORAGE_PATH='./storage'
go test ./...
go run ./cmd/server
```

The initial schema is embedded into the binary and applied at startup. `.env.example` is intended for Compose; its database password and host port variables are translated into application settings by Compose. The old Caddy and PM2 files are retained only for reference and are not used by Docker deployment.
