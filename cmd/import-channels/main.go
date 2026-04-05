package main

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

type xmlChannels struct {
	Channels []xmlChannel `xml:"channel"`
}

type xmlChannel struct {
	Site    string `xml:"site,attr"`
	SiteID  string `xml:"site_id,attr"`
	Lang    string `xml:"lang,attr"`
	XmltvID string `xml:"xmltv_id,attr"`
	Logo    string `xml:"logo,attr"`
	Name    string `xml:",chardata"`
}

type channelEntry struct {
	SiteID  string `json:"site_id"`
	Name    string `json:"name"`
	Lang    string `json:"lang"`
	Region  string `json:"region,omitempty"`
	XmltvID string `json:"xmltv_id,omitempty"`
	Logo    string `json:"logo,omitempty"`
}

var siteToPlugin = map[string]string{
	"andorradifusio.ad":            "andorradifusio",
	"artonline.tv":                 "artonline",
	"airtelxstream.in":             "airtelxstream",
	"abc.net.au":                   "abcau",
	"tv.blue.ch":                   "bluech",
	"tv.sfr.fr":                    "sfrfr",
	"allente.no":                   "allente",
	"content.astro.com.my":         "astro",
	"beinsports.com":               "beinsports",
	"canalplus.com":                "canalplus",
	"cgates.lt":                    "cgateslt",
	"ctc.ru":                       "ctcru",
	"claro.com.br":                 "clarobr",
	"clarotvmais.com.br":           "clarotvmais",
	"cosmotetv.gr":                 "cosmote",
	"cubmu.com":                    "cubmu",
	"dens.tv":                      "denstv",
	"dtv8.net":                     "dtv8",
	"webtv.delta.nl":               "deltanl",
	"derana.lk":                    "derana",
	"digiturk.com.tr":              "digiturk",
	"directv.com":                   "directvus",
	"directv.com.ar":               "directvar",
	"dishtv.in":                    "dishtvIn2",
	"distro.tv":                    "distrotv",
	"dna.fi":                       "dnafi",
	"dsmart.com.tr":                "dsmart",
	"dstv.com":                     "dstv",
	"elcinema.com":                 "elcinema",
	"energeek.cl":                  "energeek",
	"entertainment.ie":             "entertainmentie",
	"epgmaster.com":                "epgmaster",
	"epg.112114.xyz":               "epg112114",
	"firstmedia.com":               "firstmedia",
	"foxsports.com.au":             "foxsportsau",
	"foxtel.com.au":                "foxtelau",
	"freetv.tv":                    "freetv",
	"getafteritmedia.com":          "getafteritmedia",
	"freeview.co.uk":               "freeviewuk",
	"frikanalen.no":                "frikanalen",
	"galamtv.kz":                   "galamtv",
	"gatotv.com":                   "gatotv",
	"guida.tv":                     "guidatv",
	"horizon.tv":                   "horizontv",
	"hoy.tv":                       "hoytv",
	"i24news.tv":                   "i24news",
	"i.mjh.nz":                     "imjhnz",
	"iltalehti.fi":                 "iltalehti",
	"indihometv.com":               "indihome",
	"ipko.tv":                      "ipkotv",
	"jiotv.com":                    "jiotv",
	"kan.org.il":                   "kanil",
	"knr.gl":                       "knrgl",
	"kvf.fo":                       "kvffo",
	"magticom.ge":                  "magticom",
	"tv.magenta.at":                "magentaat",
	"web.magentatv.de":             "magentatv",
	"maxtvgo.mk":                   "maxtvgo",
	"mediaklikk.hu":                "mediaklikk",
	"melita.com":                   "melita",
	"meo.pt":                       "meopt",
	"mewatch.sg":                   "mewatch",
	"mi.tv":                        "mitv",
	"mncvision.id":                 "mncvision",
	"movistarplus.es":              "movistarplus",
	"mytelly.co.uk":                "mytelly",
	"tv.movistar.com.pe":           "movistarpe",
	"mtel.ba":                      "mtelba",
	"mts.rs":                       "mtsrs",
	"mujtvprogram.cz":              "mujtvprogram",
	"mysky.com.ph":                 "myskyph",
	"mytvsuper.com":                "mytvsuper",
	"neo.io":                       "neoio",
	"nhkworldpremium.com":          "nhkworld",
	"nhl.com":                      "nhlcom",
	"nostv.pt":                     "nostv",
	"nzxmltv.com":                  "nzxmltv",
	"novacyprus.com":               "novacyprus",
	"novasports.gr":                "novasports",
	"nowplayer.now.com":            "nowplayer",
	"ontvtonight.com":              "ontvtonight",
	"chaines-tv.orange.fr":         "orangefr2",
	"orangetv.orange.es":           "orangees",
	"osn.com":                      "osn",
	"passie.nl":                    "passienl",
	"pbsguam.org":                  "pbsguam",
	"pickx.be":                     "pickxbe",
	"player.ee.co.uk":              "playeree",
	"plex.tv":                      "plextv",
	"pluto.tv":                     "plutotv",
	"programacion-tv.elpais.com":   "elpais",
	"port.hu":                      "porthu",
	"programetv.ro":                "programetvro",
	"programme-tv.net":              "programmetv",
	"programtv.onet.pl":            "onetpl",
	"raiplay.it":                   "raiplay",
	"rikstv.no":                    "rikstv",
	"rtmklik.rtm.gov.my":           "rtmklik",
	"rtp.pt":                       "rtppt",
	"ruv.is":                       "ruvis",
	"sat.tv":                       "sattv",
	"shahid.mbc.net":               "shahid",
	"singtel.com":                  "singtel",
	"sky.co.nz":                    "skynz",
	"sky.com":                      "skyuk",
	"sky.de":                       "skyde",
	"guidatv.sky.it":               "skyit",
	"skylife.co.kr":                "skylife",
	"starhubtvplus.com":            "starhub",
	"streamingtvguides.com":        "streamingtvguides",
	"superguidatv.it":              "superguidatv",
	"stod2.is":                     "stod2",
	"taiwanplus.com":               "taiwanplus",
	"tapdmv.com":                   "tapdmv",
	"tataplay.com":                 "tataplay",
	"teleboy.ch":                   "teleboy",
	"epg.telemach.ba":              "telemach",
	"teliatv.ee":                   "teliatv",
	"telenet.tv":                   "telenet",
	"toonamiaftermath.com":         "toonami",
	"tv24.co.uk":                   "tv24uk",
	"tv24.se":                      "tv24se",
	"tv.post.lu":                   "tvpostlu",
	"tv.boxbg.net":                 "tvboxbg",
	"tv.cctv.com":                  "tvcctv",
	"tv.dir.bg":                    "tvdirbg",
	"tvguide.com":                  "tvguide",
	"tvinsider.com":                "tvinsider",
	"tvheute.at":                   "tvheute",
	"tv.lv":                        "tvlv",
	"tv.mail.ru":                   "tvmailru",
	"tvmi.mt":                      "tvmimt",
	"tvmusor.hu":                   "tvmusorhu",
	"tv.nu":                        "tvnu",
	"tvpassport.com":               "tvpassport",
	"tvhebdo.com":                  "tvhebdo",
	"tvtv.us":                      "tvtvus",
	"vidio.com":                    "vidio",
	"virgintvgo.virginmedia.com":   "virginmedia",
	"visionplus.id":                "visionplus",
	"vivoplay.com.br":              "vivoplay",
	"tvireland.ie":                  "tvireland",
	"vrt.be":                       "vrtbe",
	"walesi.com.fj":                "walesi",
	"wavve.com":                    "wavve",
	"winplay.co":                   "winplay",
	"xumo.tv":                      "xumotv",
	"watch.whaletvplus.com":         "whaletvplus",
	"zap2it.com":                   "zap2it",
	"ziggogo.tv":                   "ziggogo",
	"znbc.co.zm":                   "znbc",
	"tv-programme.telecablesat.fr": "telecablesat",
	"gigatv.3bbtv.co.th":           "gigatv",
	"xem.kplus.vn":                 "kplusvn",
	"v3.myafn.dodmedia.osd.mil":    "myafn",
	"rts.ch":                       "rtsch",
	"siba.com.co":                  "sibaco",
	"sjonvarp.is":                  "sjonvarp",
	"rthk.hk":                      "rthkhk",
	"mon-programme-tv.be":          "monprogrammetv",
	"yes.co.il":                    "yesil",
	"tvplus.com.tr":                "tvplustr",
	"turksatkablo.com.tr":          "turksatkablo",
	"cyta.com.cy":                  "cytacy",
	"digea.gr":                     "digeagr",
	"skyperfectv.co.jp":            "skyperfect",
	"tvguide.myjcom.jp":            "myjcom",
	"epg.iptvx.one":                "iptvxone",
	"m.tv.sms.cz":                 "tvsmscz",
	"directv.com.uy":              "directvuy",
	"tv.trueid.net":               "trueid",
	"vodafone.pt":                 "vodafonept",
	"musor.tv":                    "musortv",
	"tvcesoir.fr":                 "tvcesoir",
	"tvkaista.org":                "tvkaista",
	"tvmustra.hu":                    "tvmustra",
	"mediasetinfinity.mediaset.it":   "mediaset",
	"tvgids.nl":                      "tvgidsnl",
	"anteltv.com.uy":                 "anteltv",
	"chada.ma":                       "chadama",
	"clickthecity.com":               "clickthecity",
	"s.mxtv.jp":                      "mxtvjp",
	"webtv.sk":                       "webtvsk",
	"nuevosiglo.com.uy":              "nuevosiglo",
	"cableplus.com.uy":               "cableplus",
	"reportv.com.ar":                 "reportv",
	"programacion.tcc.com.uy":        "tccuy",
	"mojmaxtv.hrvatskitelekom.hr":    "hrvatskitelekom",
	"tv2go.t-2.net":                  "t2net",
	"tv-spored.siol.net":             "siolnet",
	"epgshare01.online":              "epgshare",
	"tvprofil.com":                   "tvprofil",
	"ena.skylifetv.co.kr":            "enaskylife",
	"watch.sportsnet.ca":             "sportsnetca",
	"moji.id":                        "mojiid",
	"guiadetv.com":                   "guiadetv",
	"telebilbao.es":                  "telebilbao",
	"roya-tv.com":                    "royatv",
	"rotana.net":                     "rotana",
	"snrt.ma":                        "snrtma",
	"ionplustv.com":                  "ionplustv",
	"zuragt.mn":                      "zuragtmn",
	"startimestv.com":                "startimes",
}

func main() {
	sitesDir := flag.String("sites", "", "path to iptv-org/epg sites directory")
	pluginsDir := flag.String("plugins", "", "path to epg plugins directory")
	dbCSV := flag.String("db", "", "path or URL to iptv-org/database logos.csv for logo enrichment (optional)")
	channelsCSV := flag.String("channels", "", "path or URL to iptv-org/database channels.csv for name-based logo fallback (optional)")
	tvLogosDir := flag.String("tv-logos", "", "path to cloned tv-logo/tv-logos repository for additional logo coverage (optional)")
	tvLogosURL := flag.String("tv-logos-url", "https://raw.githubusercontent.com/gavinmcnair/tv-logos/main", "base raw URL for tv-logos repository")
	flag.Parse()

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	if *sitesDir == "" || *pluginsDir == "" {
		log.Fatal().Msg("usage: import-channels --sites /path/to/sites --plugins /path/to/plugins [--db logos.csv] [--channels channels.csv] [--tv-logos /path/to/tv-logos]")
	}

	logosByID := make(map[string]string)
	if *dbCSV != "" {
		var err error
		logosByID, err = loadLogosFromDB(*dbCSV)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load logos database")
		}
		log.Info().Int("logos", len(logosByID)).Msg("loaded logos from database")
	}

	logosByName := make(map[string]string)
	xmltvByName := make(map[string]string)
	if *channelsCSV != "" {
		var err error
		logosByName, xmltvByName, err = buildNameIndexes(*channelsCSV, logosByID)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load channels database")
		}
		log.Info().Int("name_entries", len(logosByName)).Int("xmltv_entries", len(xmltvByName)).Msg("built name-based indexes")
	}

	tvLogos := make(map[string]string)
	channelMapOverrides := make(map[string]string) // "\x00cc:norm" or "norm" → xmltv_id
	if *tvLogosDir != "" {
		var err error
		tvLogos, err = buildTVLogosIndex(*tvLogosDir, *tvLogosURL)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to build tv-logos index")
		}
		log.Info().Int("tv_logos", len(tvLogos)).Msg("built tv-logos index")

		channelMapOverrides, err = loadChannelMaps(*tvLogosDir)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load channel maps")
		}
		log.Info().Int("channel_map_entries", len(channelMapOverrides)).Msg("loaded channel map overrides")
	}

	files, err := filepath.Glob(filepath.Join(*sitesDir, "*", "*.channels.xml"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to glob sites")
	}

	merged := make(map[string]map[string]channelEntry)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		region := extractRegion(filepath.Base(file))

		var xmlCh xmlChannels
		if err := xml.Unmarshal(data, &xmlCh); err != nil {
			continue
		}

		for _, ch := range xmlCh.Channels {
			name := strings.TrimSpace(ch.Name)
			if name == "" || ch.SiteID == "" {
				continue
			}

			pluginName, ok := siteToPlugin[ch.Site]
			if !ok {
				continue
			}

			if merged[pluginName] == nil {
				merged[pluginName] = make(map[string]channelEntry)
			}

			xmltvID := ch.XmltvID
			if name != "" {
				siteCC := countryCodeFromDomain(ch.Site)
				// Channel-map overrides take highest priority
				if override := lookupXmltvID(channelMapOverrides, name, siteCC); override != "" {
					xmltvID = override
				} else if xmltvID == "" {
					xmltvID = lookupXmltvID(xmltvByName, name, siteCC)
				}
			}

			logo := ch.Logo
			if logo == "" && xmltvID != "" {
				logo = lookupLogo(logosByID, xmltvID)
			}
			if logo == "" && name != "" {
				logo = logosByName[normalizeName(name)]
			}
			if logo == "" && len(tvLogos) > 0 {
				cc := countryCodeFromXmltvID(xmltvID)
				logo = lookupTVLogo(tvLogos, name, cc)
			}

			key := ch.SiteID + "|" + ch.Lang + "|" + region
			merged[pluginName][key] = channelEntry{
				SiteID:  ch.SiteID,
				Name:    name,
				Lang:    ch.Lang,
				Region:  region,
				XmltvID: xmltvID,
				Logo:    logo,
			}
		}
	}

	total := 0
	pluginCount := 0

	for pluginName, channelMap := range merged {
		pluginDir := filepath.Join(*pluginsDir, pluginName)
		if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
			continue
		}

		existingLogos := loadExistingLogos(filepath.Join(pluginDir, "channels.json"))

		entries := make([]channelEntry, 0, len(channelMap))
		for _, e := range channelMap {
			if e.Logo == "" && e.SiteID != "" {
				e.Logo = existingLogos[e.SiteID]
			}
			entries = append(entries, e)
		}

		outPath := filepath.Join(pluginDir, "channels.json")
		outData, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			log.Error().Err(err).Str("plugin", pluginName).Msg("failed to marshal channels")
			continue
		}

		if err := os.WriteFile(outPath, outData, 0644); err != nil {
			log.Error().Err(err).Str("path", outPath).Msg("failed to write channels")
			continue
		}

		total += len(entries)
		pluginCount++
		fmt.Printf("%s: %d channels\n", pluginName, len(entries))
	}

	fmt.Printf("\nDone: %d unique channels across %d plugins\n", total, pluginCount)
}

func openCSV(path string) (io.Reader, func(), error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path) //nolint:noctx
		if err != nil {
			return nil, nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, path)
		}
		return resp.Body, func() { resp.Body.Close() }, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func loadLogosFromDB(path string) (map[string]string, error) {
	r, closer, err := openCSV(path)
	if err != nil {
		return nil, err
	}
	defer closer()

	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	idCol, logoCol := -1, -1
	for i, h := range header {
		switch h {
		case "id", "channel":
			idCol = i
		case "logo", "url":
			logoCol = i
		}
	}
	if idCol < 0 || logoCol < 0 {
		return nil, fmt.Errorf("CSV missing channel ID column ('id'/'channel') or logo column ('logo'/'url')")
	}

	logos := make(map[string]string)
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}
		if idCol >= len(row) || logoCol >= len(row) {
			continue
		}
		id := row[idCol]
		logo := row[logoCol]
		if id != "" && logo != "" {
			logos[id] = logo
		}
	}
	return logos, nil
}

func normalizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// buildNameIndexes reads channels.csv once and returns:
//   - logosByName: normalized name → logo URL
//   - xmltvByCountryName: "country:normalized_name" → xmltv_id (country-specific)
//   - xmltvByName: normalized name → first-seen xmltv_id (fallback)
func buildNameIndexes(path string, logosByID map[string]string) (logosByName, xmltvByName map[string]string, err error) {
	r, closer, openErr := openCSV(path)
	if openErr != nil {
		return nil, nil, openErr
	}
	defer closer()

	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("reading CSV header: %w", err)
	}

	idCol, nameCol, altCol, countryCol := -1, -1, -1, -1
	for i, h := range header {
		switch h {
		case "id":
			idCol = i
		case "name":
			nameCol = i
		case "alt_names":
			altCol = i
		case "country":
			countryCol = i
		}
	}
	if idCol < 0 || nameCol < 0 {
		return nil, nil, fmt.Errorf("channels.csv missing 'id' or 'name' column")
	}

	logosByName = make(map[string]string)
	xmltvByName = make(map[string]string)
	// country-specific: "cc:norm" → xmltv_id
	xmltvByCountryName := make(map[string]string)

	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading CSV row: %w", err)
		}
		if idCol >= len(row) || nameCol >= len(row) {
			continue
		}
		id := row[idCol]
		logo := logosByID[id]

		// Extract country code from channel id (e.g. "BBCRadio2.uk" → "uk")
		cc := countryCodeFromXmltvID(id)
		if cc == "" && countryCol >= 0 && countryCol < len(row) {
			cc = strings.ToLower(strings.TrimSpace(row[countryCol]))
		}

		names := []string{row[nameCol]}
		if altCol >= 0 && altCol < len(row) && row[altCol] != "" {
			for _, alt := range strings.Split(row[altCol], ";") {
				if alt = strings.TrimSpace(alt); alt != "" {
					names = append(names, alt)
				}
			}
		}

		for _, n := range names {
			norm := normalizeName(n)
			if norm == "" {
				continue
			}
			if logo != "" {
				if _, exists := logosByName[norm]; !exists {
					logosByName[norm] = logo
				}
			}
			if id != "" {
				// Country-specific index
				if cc != "" {
					ccKey := cc + ":" + norm
					if _, exists := xmltvByCountryName[ccKey]; !exists {
						xmltvByCountryName[ccKey] = id
					}
				}
				// Global fallback index
				if _, exists := xmltvByName[norm]; !exists {
					xmltvByName[norm] = id
				}
			}
		}
	}

	// Merge country-specific entries into xmltvByName under a sentinel key
	// so lookupXmltvID can access them. Encode as "cc\x00norm" → id.
	for k, v := range xmltvByCountryName {
		xmltvByName["\x00"+k] = v
	}

	return logosByName, xmltvByName, nil
}

// lookupXmltvID returns the correct xmltv_id for a channel name.
// If the source has a country code, only country-specific matches are used —
// no fallback to a wrong-country entry. For sources without a country code,
// the global index is used.
func lookupXmltvID(xmltvByName map[string]string, name, countryCode string) string {
	norm := normalizeName(name)
	if countryCode != "" {
		return xmltvByName["\x00"+countryCode+":"+norm]
	}
	return xmltvByName[norm]
}

func lookupLogo(logos map[string]string, xmltvID string) string {
	if l := logos[xmltvID]; l != "" {
		return l
	}
	if idx := strings.Index(xmltvID, "@"); idx >= 0 {
		return logos[xmltvID[:idx]]
	}
	return ""
}

func loadExistingLogos(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entries []channelEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.Logo != "" {
			m[e.SiteID] = e.Logo
		}
	}
	return m
}

// loadChannelMaps reads channel-map.json from each countries/* directory in
// the tv-logos repo. Each file maps normalized channel name → xmltv_id for
// that country. Returns the same format as xmltvByName (sentinel-keyed).
func loadChannelMaps(dir string) (map[string]string, error) {
	result := make(map[string]string)
	countriesDir := filepath.Join(dir, "countries")

	entries, err := os.ReadDir(countriesDir)
	if err != nil {
		return nil, fmt.Errorf("reading tv-logos countries dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mapFile := filepath.Join(countriesDir, entry.Name(), "channel-map.json")
		data, err := os.ReadFile(mapFile)
		if err != nil {
			continue // no channel-map.json in this directory — skip
		}

		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", mapFile, err)
		}

		cc := dirToCountryCode[entry.Name()]
		for name, xmltvID := range m {
			norm := normalizeName(name)
			if norm == "" || xmltvID == "" {
				continue
			}
			if cc != "" {
				result["\x00"+cc+":"+norm] = xmltvID
			}
			// Also add to global fallback if not already set
			if _, exists := result[norm]; !exists {
				result[norm] = xmltvID
			}
		}
	}

	return result, nil
}

var dirToCountryCode = map[string]string{
	"albania":              "al",
	"argentina":            "ar",
	"australia":            "au",
	"austria":              "at",
	"azerbaijan":           "az",
	"belgium":              "be",
	"brazil":               "br",
	"bulgaria":             "bg",
	"canada":               "ca",
	"caribbean":            "car",
	"chile":                "cl",
	"costa-rica":           "cr",
	"croatia":              "hr",
	"czech-republic":       "cz",
	"france":               "fr",
	"germany":              "de",
	"greece":               "gr",
	"hong-kong":            "hk",
	"hungary":              "hu",
	"india":                "in",
	"indonesia":            "id",
	"international":        "int",
	"ireland":              "ie",
	"israel":               "il",
	"italy":                "it",
	"lebanon":              "lb",
	"lithuania":            "lt",
	"luxembourg":           "lu",
	"malaysia":             "my",
	"malta":                "mt",
	"mexico":               "mx",
	"netherlands":          "nl",
	"new-zealand":          "nz",
	"nordic":               "nordic",
	"philippines":          "ph",
	"poland":               "pl",
	"portugal":             "pt",
	"romania":              "ro",
	"russia":               "ru",
	"serbia":               "rs",
	"singapore":            "sg",
	"slovakia":             "sk",
	"slovenia":             "si",
	"south-africa":         "za",
	"spain":                "es",
	"switzerland":          "ch",
	"turkey":               "tr",
	"ukraine":              "ua",
	"united-arab-emirates": "ae",
	"united-kingdom":       "uk",
	"united-states":        "us",
	"world-africa":         "afr",
	"world-asia":           "asi",
	"world-europe":         "eu",
	"world-latin-america":  "lam",
	"world-middle-east":    "mea",
}

func buildTVLogosIndex(dir, baseURL string) (map[string]string, error) {
	countriesDir := filepath.Join(dir, "countries")
	entries, err := os.ReadDir(countriesDir)
	if err != nil {
		return nil, fmt.Errorf("reading tv-logos countries dir: %w", err)
	}

	index := make(map[string]string)
	baseURL = strings.TrimSuffix(baseURL, "/")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		countryDir := entry.Name()
		cc := dirToCountryCode[countryDir]

		files, err := os.ReadDir(filepath.Join(countriesDir, countryDir))
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".png") {
				continue
			}
			slug := strings.TrimSuffix(f.Name(), ".png")
			rawURL := baseURL + "/countries/" + countryDir + "/" + f.Name()

			// Index by full slug (e.g. "bbc-one-uk")
			if _, exists := index[slug]; !exists {
				index[slug] = rawURL
			}

			// Index by name without country suffix (e.g. "bbc-one")
			if cc != "" && strings.HasSuffix(slug, "-"+cc) {
				nameSlug := strings.TrimSuffix(slug, "-"+cc)
				if _, exists := index[nameSlug]; !exists {
					index[nameSlug] = rawURL
				}
			}
		}
	}

	return index, nil
}

func normalizeTVLogosSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	prev := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prev = false
		} else if !prev {
			b.WriteRune('-')
			prev = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func lookupTVLogo(index map[string]string, name, countryCode string) string {
	slug := normalizeTVLogosSlug(name)
	if countryCode != "" {
		if url := index[slug+"-"+countryCode]; url != "" {
			return url
		}
	}
	return index[slug]
}

// countryCodeFromDomain extracts 2-letter country code from site domain TLD.
// "freeview.co.uk" → "uk", "raiplay.it" → "it", "sky.com" → ""
func countryCodeFromDomain(site string) string {
	parts := strings.Split(site, ".")
	if len(parts) == 0 {
		return ""
	}
	tld := parts[len(parts)-1]
	if len(tld) == 2 {
		return strings.ToLower(tld)
	}
	return ""
}

func countryCodeFromXmltvID(xmltvID string) string {
	if xmltvID == "" {
		return ""
	}
	// strip @feed suffix
	if i := strings.Index(xmltvID, "@"); i >= 0 {
		xmltvID = xmltvID[:i]
	}
	if i := strings.LastIndex(xmltvID, "."); i >= 0 {
		tld := strings.ToLower(xmltvID[i+1:])
		if len(tld) == 2 {
			return tld
		}
	}
	return ""
}

func extractRegion(filename string) string {
	name := strings.TrimSuffix(filename, ".channels.xml")
	idx := strings.LastIndex(name, "_")
	if idx < 0 {
		return ""
	}
	region := name[idx+1:]
	if dashIdx := strings.Index(region, "-"); dashIdx >= 0 {
		region = region[:dashIdx]
	}
	return strings.ToUpper(region)
}
