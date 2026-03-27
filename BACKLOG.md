# EPG Service — Feature Backlog

## Implemented
- [x] Domain models (Channel, Program, Source)
- [x] Plugin interface + registry (blank import pattern)
- [x] In-memory store with gob persistence
- [x] Store filtering (source, channel, genre, quality, lang, date)
- [x] XMLTV serializer
- [x] Pluto.tv plugin with tests
- [x] Grabber with concurrency control

## In Progress
- [ ] REST API (Chi router)
- [ ] XMLTV legacy endpoints
- [ ] OpenAPI spec

## Planned
- [ ] External XMLTV source import (add URLs via UI as first-class sources)
- [ ] User management + authentication
- [ ] Embedded web UI (channel browser, grab trigger, status dashboard)
- [ ] tvg-id metadata standardization
- [ ] More plugins (arirang, movistarplus, directv, etc.)

## Feature Ideas / Competitor Inspiration
- [ ] Scheduled grabbing (built-in cron)
- [ ] WebGrab+Plus: SiteIni configuration files (declarative scraping)
- [ ] WebGrab+Plus: TimeShift support for timezone handling
- [ ] WebGrab+Plus: Show detail enrichment (sub-page fetching)
- [ ] WebGrab+Plus: MDB (movie database) integration for metadata
- [ ] EPG caching / deduplication across sources
- [ ] Channel mapping / merge (same channel from multiple sources)
- [ ] Programme genre normalization (map site-specific genres to standard set)
- [ ] Gzip XMLTV endpoint support
- [ ] M3U playlist generation with tvg-id tags
- [ ] Health check / status endpoint
- [ ] Grab progress tracking (SSE/WebSocket)
- [ ] Rate limiting per source
- [ ] Retry logic with exponential backoff
- [ ] Proxy support (SOCKS5, HTTP)
- [ ] Docker image
- [ ] Prometheus metrics
- [ ] Channel logo proxy/cache
- [ ] Programme image proxy/cache
- [ ] Multi-language programme title support
- [ ] Parental rating normalization

## Optimisations Missing
- [ ] Incremental grab (only fetch dates not already cached)
- [ ] Parallel source grabbing (grab multiple sources simultaneously)
- [ ] Response caching at HTTP client level
- [ ] Connection pooling per source
- [ ] Batch XMLTV generation (avoid regenerating unchanged data)
