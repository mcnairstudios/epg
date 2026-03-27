# EPG Service — Feature Backlog

## Implemented
- [x] Domain models (Channel, Program, Source)
- [x] Plugin interface + registry (blank import pattern)
- [x] In-memory store with gob persistence, sorted output
- [x] Store filtering (source, channel, genre, quality, lang, date)
- [x] XMLTV serializer + parser
- [x] Grabber with concurrency control
- [x] REST API (Chi router) with full CRUD
- [x] XMLTV legacy endpoints (/epg.xml, /epg/{source}.xml with dotted names)
- [x] Gzip XMLTV endpoint support (Accept-Encoding)
- [x] OpenAPI 3.0 spec
- [x] External XMLTV source import (add URLs via UI, gzip support)
- [x] User management + token-based authentication
- [x] Embedded web UI (sources, channels, programs, XMLTV URLs, users, grab-all)
- [x] Stats endpoint (source/channel/program counts)
- [x] Health check endpoint
- [x] Built-in cron scheduler (--schedule flag)
- [x] Dockerfile (multi-stage alpine build)
- [x] README with documentation
- [x] 101 plugins with full test coverage across 40+ countries

## Planned
- [ ] tvg-id metadata standardization
- [ ] More plugins from iptv-org/epg (~140 remaining)
- [ ] WebGrab+Plus source research
- [ ] Channel mapping / merge (same channel from multiple sources)
- [ ] Programme genre normalization

## Feature Ideas
- [ ] M3U playlist generation with tvg-id tags
- [ ] Grab progress tracking (SSE/WebSocket)
- [ ] Rate limiting per source
- [ ] Retry logic with exponential backoff
- [ ] Proxy support (SOCKS5, HTTP)
- [ ] Prometheus metrics
- [ ] Channel logo proxy/cache

## Optimisations
- [ ] Incremental grab (only fetch dates not already cached)
- [ ] Parallel source grabbing (grab multiple sources simultaneously)
- [ ] Response caching at HTTP client level
