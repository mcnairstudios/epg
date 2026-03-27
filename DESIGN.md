# EPG Service — Design Specification

## Purpose

Go-based EPG (Electronic Program Guide) service that scrapes TV schedule data from broadcaster websites via a plugin architecture and serves it through a RESTful JSON API and legacy XMLTV endpoints.

## Architecture

### Plugin System

Blank import registration pattern. Each plugin is an independent Go package that:
- Implements the `plugin.Provider` interface
- Registers itself via `init()` calling `plugin.Register()`
- Is added/removed by adding/removing a blank import in `cmd/epg/main.go`

```go
type Provider interface {
    Name() string
    Description() string
    Channels() ([]domain.Channel, error)
    Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error)
}
```

### Storage

In-memory store with gob file serialization for persistence across restarts. Thread-safe via `sync.RWMutex`. Supports filtering by source, channel, genre, quality, language, and date.

### API

#### REST (JSON) — `/api/v1/`
- `GET /api/v1/sources` — list registered plugins and external sources
- `GET /api/v1/channels?source=&genre=&quality=&lang=` — list/filter channels
- `GET /api/v1/channels/{id}` — single channel detail
- `GET /api/v1/programs?channel=&source=&date=&genre=` — list/filter programs
- `POST /api/v1/grab` — trigger a grab `{"source": "pluto.tv"}`
- `GET /api/v1/users` — list users (admin)
- `POST /api/v1/users` — create user

#### XMLTV (legacy)
- `GET /epg.xml?source=&channel=&genre=&quality=` — filtered XMLTV output
- `GET /epg/{source}.xml` — per-source XMLTV

#### External Sources (via UI)
- Users can add external XMLTV URLs as first-class sources
- These are imported alongside plugin-sourced data
- Treated identically in the API/filtering layer

### Domain Models

#### Channel
- ID, Name, Source, SiteID, Lang, Logo, Genre, Quality, XmltvID

#### Program
- Channel, Source, Title, SubTitle, Description, Category, Season, Episode, Image, Lang, Start, Stop

### Metadata / Tagging

Channels carry structured metadata for tvg-id compatibility:
- `XmltvID` — unique identifier in format `ChannelName.source@feed`
- `Quality` — SD, HD, FHD, 4K
- `Genre` — standardized genre tag
- `Lang` — ISO 639-1 language code

### Package Structure

| Package | Responsibility |
|---------|---------------|
| `internal/domain` | Core data models, no dependencies |
| `internal/plugin` | Provider interface + global registry |
| `internal/store` | In-memory data store + gob persistence |
| `internal/grabber` | Orchestration: fetch EPG data via plugins |
| `internal/xmltv` | XMLTV XML serialization |
| `internal/api` | HTTP handlers (REST + XMLTV endpoints) |
| `internal/auth` | User management + authentication |
| `internal/ui` | Embedded web interface |
| `plugins/*` | Site adapter implementations |
