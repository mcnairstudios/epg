# EPG Service — Feature Backlog

## Implemented
- [x] Domain models (Channel, Program, Source)
- [x] Plugin interface + registry (blank import pattern)
- [x] In-memory store with gob persistence
- [x] Store filtering (source, channel, genre, quality, lang, date)
- [x] XMLTV serializer + parser
- [x] Grabber with concurrency control
- [x] REST API (Chi router) with full CRUD
- [x] XMLTV legacy endpoints (/epg.xml, /epg/{source}.xml)
- [x] OpenAPI 3.0 spec
- [x] External XMLTV source import (add URLs via UI, gzip support)
- [x] User management + token-based authentication
- [x] Embedded web UI (sources, channels, programs, XMLTV, users)
- [x] 14 plugins: pluto.tv, sky.com, foxsports.com.au, singtel, magticom.ge, i.mjh.nz, directv.com.ar, mi.tv, starhubtvplus, plex.tv, distro.tv, vrt.be, dstv.com, tvguide.com
- [x] Full test coverage for all packages and plugins

## Planned
- [ ] tvg-id metadata standardization
- [ ] More plugins from iptv-org/epg (~226 remaining)
- [ ] WebGrab+Plus source research and plugin conversion
- [ ] Scheduled grabbing (built-in cron)
- [ ] Dockerfile

## Feature Ideas / Competitor Inspiration
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
