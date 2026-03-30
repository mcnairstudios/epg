# EPG Service

Go-based Electronic Program Guide service with 223+ source plugins, REST API, and legacy XMLTV endpoints.

## Quick Start

```sh
go build -o epg-server ./cmd/epg/
./epg-server
```

Open http://localhost:8080 for the web UI.

## Features

- 223+ source plugins covering 150+ countries
- RESTful JSON API with OpenAPI spec
- Legacy XMLTV endpoints for IPTV player compatibility
- External XMLTV source import
- On-demand logo proxy with local disk cache
- Rolling grabber — grabs each source at midnight in its local timezone
- Built-in cron scheduler
- Token and session-based authentication
- Embedded web UI with grab progress and bandwidth monitor
- In-memory store with gob persistence
- Multi-arch Docker image (amd64 + arm64)

## Docker

```sh
docker run -p 8080:8080 -v /path/to/data:/data gavinmcnair/epg
```

## Usage

```
./epg-server [flags]

Flags:
  -addr                string   Listen address (default ":8080")
  -data-dir            string   Directory for persistent data files (default ".")
  -days                int      Number of days ahead to grab (default 3)
  -max-connections     int      Max concurrent connections per grab (default 3)
  -schedule            string   Cron schedule for automatic grabbing (e.g. "0 0 * * *")
  -rolling             bool     Enable rolling grabber — grabs each source at midnight in its timezone
  -rolling-parallel    int      Max parallel grabs for rolling grabber (default 2)
```

## API

Authentication is required for all API endpoints. Use a Bearer token (`Authorization: Bearer <token>`) or `?token=<token>` query parameter. Admin-only endpoints additionally require a valid session cookie.

### Read-only (Bearer token or session)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/sources` | List all registered sources |
| `GET` | `/api/v1/channels` | List channels — filter by `?source=`, `?genre=`, `?quality=`, `?lang=`, `?region=` |
| `GET` | `/api/v1/channels/{id}` | Get a single channel by ID |
| `GET` | `/api/v1/regions` | List all regions |
| `GET` | `/api/v1/programs` | List programs — filter by `?channel=`, `?source=`, `?genre=`, `?date=` |
| `GET` | `/api/v1/stats` | Source, channel, and program counts |
| `GET` | `/api/v1/grab-stats` | Per-source grab statistics (last run, programs fetched, errors) |
| `GET` | `/api/v1/grab-progress` | Live grab progress |
| `GET` | `/api/v1/bandwidth` | Current bandwidth rate and total bytes transferred |

### Admin-only (session cookie)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/grab` | Trigger a grab — body: `{"source":"plutotv"}` |
| `POST` | `/api/v1/grab-all` | Trigger grab of all sources |
| `GET` | `/api/v1/external-sources` | List external XMLTV sources |
| `POST` | `/api/v1/external-sources` | Add external source — body: `{"name":"uk","url":"..."}` |
| `DELETE` | `/api/v1/external-sources/{name}` | Remove external source |
| `POST` | `/api/v1/external-sources/{name}/grab` | Trigger grab of one external source |
| `GET` | `/api/v1/disabled` | List disabled sources |
| `POST` | `/api/v1/disabled/{name}` | Disable a source |
| `DELETE` | `/api/v1/disabled/{name}` | Enable a source |
| `GET` | `/api/v1/users` | List users |
| `POST` | `/api/v1/users` | Create user — body: `{"username":"viewer","role":"viewer"}` |
| `PUT` | `/api/v1/users/{id}/password` | Change password |
| `DELETE` | `/api/v1/users/{id}` | Delete user |

### XMLTV (Bearer token or session)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/epg.xml` | Full EPG — filter by `?source=`, `?channel=`, `?genre=`, `?quality=` |
| `GET` | `/epg/{source}.xml` | Per-source XMLTV |

### Public

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/logos/{id}` | Channel logo — proxied and cached from source on first request |
| `GET` | `/health` | Health check — returns `{"status":"ok"}` |
| `GET` | `/api/openapi.yaml` | OpenAPI spec |

## Adding a Plugin

Each plugin implements the `plugin.Provider` interface:

```go
type Provider interface {
    Name() string
    Description() string
    Channels() ([]domain.Channel, error)
    Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error)
}
```

Register in `init()`:

```go
func init() {
    plugin.Register(&provider{})
}
```

Add a blank import in `cmd/epg/main.go`:

```go
_ "github.com/gavinmcnair/epg/plugins/yourplugin"
```

## Architecture

| Package | Role |
|---------|------|
| `internal/domain` | Core models — Channel, Program, Source |
| `internal/plugin` | Provider interface and global registry |
| `internal/store` | Thread-safe in-memory store with gob persistence |
| `internal/grabber` | Fetch orchestration with concurrency control |
| `internal/scheduler` | Cron scheduler and rolling midnight grabber |
| `internal/xmltv` | XMLTV serialisation and parsing |
| `internal/api` | REST handlers, XMLTV endpoints, logo proxy |
| `internal/auth` | Token and session-based authentication |
| `internal/source` | External XMLTV source manager |
| `internal/logocache` | On-demand logo proxy with local disk cache |
| `internal/bandwidth` | Rolling bandwidth tracker |
| `internal/ui` | Embedded web interface |
| `plugins/*` | Source adapter plugins |
