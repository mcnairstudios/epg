# EPG Service

Plugin-based EPG (Electronic Programme Guide) service written in Go.

## Project Structure

- `cmd/epg/` - Main server binary
- `cmd/import-channels/` - Channel import tool
- `cmd/logo-downloader/` - Logo pre-download tool
- `cmd/probe/` - Site probing tool
- `internal/` - Core packages (api, auth, store, grabber, plugin, logocache, settings, etc.)
- `internal/ui/static/index.html` - Single-page admin UI (vanilla JS)
- `plugins/` - EPG source plugins (~180), each with `channels.json` (go:embed)
- `images/` - Self-hosted channel logos (freeview/, bbc/)

## Build & Deploy

```
make build          # Local binary
make test           # Run all tests
make docker-build   # Multi-arch Docker build + push (uses buildx, pushes to gavinmcnair/epg:latest)
```

Docker image: `gavinmcnair/epg`
Builder: `mybuilder` (docker buildx)
Platforms: `linux/amd64,linux/arm64`

## Key Technical Details

- Channel logos are embedded via `go:embed channels.json` in each plugin
- Logo proxy at `/logos/{id}` is public (no auth) - fetches from source URL on first request, caches to disk
- XMLTV feeds at `/epg/*.xml` require token auth
- Admin UI at `/` requires session cookie
- Settings stored in `settings.json` (data volume), includes TMDB API key
- TMDB integration: `/api/v1/tmdb/search` and `/api/v1/tmdb/details` with 24h in-memory cache

## Logo Sources (priority order)

1. `images/freeview/` - Official Freeview logos (webp), served from GitHub raw
2. `images/bbc/` - BBC Radio SVGs from Wikimedia Commons
3. `gavinmcnair/tv-logos` repo - Large collection of international channel logos (png)
4. External URLs (wikimedia, etc.) - Used where no self-hosted alternative exists

## Code Style

- No comments unless logic is non-obvious
- Tests required for new packages
- Clean interfaces, minimal abstractions
- `go build ./...` and `go test ./...` must pass before committing

## Deployment

- Live instance: `epg.41rpa.uk` (behind Cloudflare)
- Auth: session cookie for UI, token for XMLTV feeds
- Data persisted in `/data` volume (logos cache, settings, grab data)
