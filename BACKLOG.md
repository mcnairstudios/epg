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
- [x] OpenAPI 3.0 spec
- [x] External XMLTV source import (add URLs via UI, gzip support)
- [x] User management + token-based authentication
- [x] Embedded web UI (sources, channels, programs, XMLTV, users)
- [x] Per-source XMLTV URL display in UI
- [x] Grab-all endpoint and UI button
- [x] Stats endpoint (source/channel/program counts)
- [x] Health check endpoint
- [x] Dockerfile (multi-stage alpine build)
- [x] 49+ plugins with full test coverage
- [x] Plugins cover: AU, NZ, UK, US, Europe (BE, NL, FR, DE, IT, ES, PT, RO, HU, NO, SE, GR, TR), Asia (SG, MY, KR, JP, ID, IN, GE, HK), Middle East (OSN, beIN), Africa (DStv), South America (AR, BR, CL, CO, MX, PE)

## Planned
- [ ] tvg-id metadata standardization
- [ ] More plugins from iptv-org/epg (~190 remaining)
- [ ] WebGrab+Plus source research and plugin conversion
- [ ] Scheduled grabbing (built-in cron)

## Feature Ideas / Competitor Inspiration
- [ ] WebGrab+Plus: SiteIni configuration files (declarative scraping)
- [ ] WebGrab+Plus: MDB (movie database) integration for metadata
- [ ] EPG caching / deduplication across sources
- [ ] Channel mapping / merge (same channel from multiple sources)
- [ ] Programme genre normalization (map site-specific genres to standard set)
- [ ] Gzip XMLTV endpoint support (Accept-Encoding)
- [ ] M3U playlist generation with tvg-id tags
- [ ] Grab progress tracking (SSE/WebSocket)
- [ ] Rate limiting per source
- [ ] Retry logic with exponential backoff
- [ ] Proxy support (SOCKS5, HTTP)
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
