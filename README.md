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
- Legacy XMLTV XML endpoints for IPTV player compatibility
- External XMLTV source import (add any XMLTV URL)
- Token-based authentication
- Embedded web UI
- Built-in cron scheduler
- In-memory store with gob persistence
- Dockerfile included

## Usage

```sh
./epg-server \
  -addr :8080 \
  -data-dir /data \
  -days 3 \
  -max-connections 3 \
  -schedule "0 0 * * *"
```

## API

### REST (JSON)
- `GET /api/v1/sources` - list all sources
- `GET /api/v1/channels?source=&genre=&quality=&lang=` - list channels
- `GET /api/v1/programs?channel=&source=&date=&genre=` - list programs
- `GET /api/v1/stats` - source/channel/program counts
- `POST /api/v1/grab` - trigger grab `{"source":"pluto.tv"}`
- `POST /api/v1/grab-all` - grab all sources
- `GET /api/v1/external-sources` - list external XMLTV sources
- `POST /api/v1/external-sources` - add external source `{"name":"uk","url":"..."}`
- `GET /api/v1/users` - list users
- `POST /api/v1/users` - create user `{"username":"admin","role":"admin"}`

### XMLTV
- `GET /epg.xml` - full EPG in XMLTV format
- `GET /epg/{source}.xml` - per-source XMLTV

### Health
- `GET /health` - health check

## Docker

```sh
docker build -t epg .
docker run -p 8080:8080 -v /path/to/data:/data epg
```

## Adding a Plugin

Each plugin is an independent Go package that implements the `plugin.Provider` interface:

```go
type Provider interface {
    Name() string
    Description() string
    Channels() ([]domain.Channel, error)
    Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error)
}
```

Register via `init()`:

```go
func init() {
    plugin.Register(&provider{client: http.DefaultClient})
}
```

Add a blank import in `cmd/epg/main.go`:

```go
_ "github.com/gavinmcnair/epg/plugins/yourplugin"
```

## Architecture

- `internal/domain` - Core models (Channel, Program, Source)
- `internal/plugin` - Provider interface + global registry
- `internal/store` - In-memory store + gob persistence
- `internal/grabber` - Fetch orchestration with concurrency
- `internal/xmltv` - XMLTV serialization + parsing
- `internal/api` - REST handlers + XMLTV endpoints
- `internal/auth` - Token-based auth
- `internal/source` - External XMLTV source manager
- `internal/scheduler` - Built-in cron scheduler
- `internal/ui` - Embedded web interface
- `plugins/*` - Source adapter plugins
