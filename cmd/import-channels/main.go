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
	flag.Parse()

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	if *sitesDir == "" || *pluginsDir == "" {
		log.Fatal().Msg("usage: import-channels --sites /path/to/sites --plugins /path/to/plugins [--db logos.csv] [--channels channels.csv]")
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
	if *channelsCSV != "" && len(logosByID) > 0 {
		var err error
		logosByName, err = buildNameLogoIndex(*channelsCSV, logosByID)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load channels database")
		}
		log.Info().Int("name_entries", len(logosByName)).Msg("built name-based logo index")
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

			logo := ch.Logo
			if logo == "" && ch.XmltvID != "" {
				logo = lookupLogo(logosByID, ch.XmltvID)
			}
			if logo == "" && name != "" {
				logo = logosByName[normalizeName(name)]
			}

			key := ch.SiteID + "|" + ch.Lang + "|" + region
			merged[pluginName][key] = channelEntry{
				SiteID:  ch.SiteID,
				Name:    name,
				Lang:    ch.Lang,
				Region:  region,
				XmltvID: ch.XmltvID,
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

func buildNameLogoIndex(path string, logosByID map[string]string) (map[string]string, error) {
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

	idCol, nameCol, altCol := -1, -1, -1
	for i, h := range header {
		switch h {
		case "id":
			idCol = i
		case "name":
			nameCol = i
		case "alt_names":
			altCol = i
		}
	}
	if idCol < 0 || nameCol < 0 {
		return nil, fmt.Errorf("channels.csv missing 'id' or 'name' column")
	}

	index := make(map[string]string)
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}
		if idCol >= len(row) || nameCol >= len(row) {
			continue
		}
		id := row[idCol]
		logo := logosByID[id]
		if logo == "" {
			continue
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
			if norm := normalizeName(n); norm != "" {
				if _, exists := index[norm]; !exists {
					index[norm] = logo
				}
			}
		}
	}
	return index, nil
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
