package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	"abc.net.au":                   "abcau",
	"allente.no":                   "allente",
	"content.astro.com.my":         "astro",
	"beinsports.com":               "beinsports",
	"canalplus.com":                "canalplus",
	"cgates.lt":                    "cgateslt",
	"claro.com.br":                 "clarobr",
	"clarotvmais.com.br":           "clarotvmais",
	"cosmotetv.gr":                 "cosmote",
	"cubmu.com":                    "cubmu",
	"dens.tv":                      "denstv",
	"derana.lk":                    "derana",
	"digiturk.com.tr":              "digiturk",
	"directv.com.ar":               "directvar",
	"dishtv.in":                    "dishtvIn",
	"distro.tv":                    "distrotv",
	"dna.fi":                       "dnafi",
	"dsmart.com.tr":                "dsmart",
	"dstv.com":                     "dstv",
	"elcinema.com":                 "elcinema",
	"epg.112114.xyz":               "epg112114",
	"firstmedia.com":               "firstmedia",
	"foxsports.com.au":             "foxsportsau",
	"foxtel.com.au":                "foxtelau",
	"freetv.tv":                    "freetv",
	"freeview.co.uk":               "freeviewuk",
	"frikanalen.no":                "frikanalen",
	"galamtv.kz":                   "galamtv",
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
	"maxtvgo.mk":                   "maxtvgo",
	"mediaklikk.hu":                "mediaklikk",
	"melita.com":                   "melita",
	"meo.pt":                       "meopt",
	"mewatch.sg":                   "mewatch",
	"mi.tv":                        "mitv",
	"mncvision.id":                 "mncvision",
	"movistarplus.es":              "movistarplus",
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
	"novacyprus.com":               "novacyprus",
	"novasports.gr":                "novasports",
	"nowplayer.now.com":            "nowplayer",
	"ontvtonight.com":              "ontvtonight",
	"orangetv.orange.es":           "orangees",
	"osn.com":                      "osn",
	"passie.nl":                    "passienl",
	"pbsguam.org":                  "pbsguam",
	"pickx.be":                     "pickxbe",
	"plex.tv":                      "plextv",
	"pluto.tv":                     "plutotv",
	"port.hu":                      "porthu",
	"programetv.ro":                "programetvro",
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
	"skylife.co.kr":                "skylife",
	"starhubtvplus.com":            "starhub",
	"stod2.is":                     "stod2",
	"taiwanplus.com":               "taiwanplus",
	"tapdmv.com":                   "tapdmv",
	"tataplay.com":                 "tataplay",
	"teleboy.ch":                   "teleboy",
	"epg.telemach.ba":              "telemach",
	"teliatv.ee":                   "teliatv",
	"toonamiaftermath.com":         "toonami",
	"tv24.co.uk":                   "tv24uk",
	"tv.boxbg.net":                 "tvboxbg",
	"tv.cctv.com":                  "tvcctv",
	"tv.dir.bg":                    "tvdirbg",
	"tvguide.com":                  "tvguide",
	"tvheute.at":                   "tvheute",
	"tv.lv":                        "tvlv",
	"tv.mail.ru":                   "tvmailru",
	"tvmi.mt":                      "tvmimt",
	"tvmusor.hu":                   "tvmusorhu",
	"tv.nu":                        "tvnu",
	"tvpassport.com":               "tvpassport",
	"tvtv.us":                      "tvtvus",
	"vidio.com":                    "vidio",
	"virgintvgo.virginmedia.com":   "virginmedia",
	"visionplus.id":                "visionplus",
	"vivoplay.com.br":              "vivoplay",
	"vrt.be":                       "vrtbe",
	"walesi.com.fj":                "walesi",
	"wavve.com":                    "wavve",
	"winplay.co":                   "winplay",
	"xumo.tv":                      "xumotv",
	"znbc.co.zm":                   "znbc",
}

func main() {
	sitesDir := flag.String("sites", "", "path to iptv-org/epg sites directory")
	pluginsDir := flag.String("plugins", "", "path to epg plugins directory")
	flag.Parse()

	if *sitesDir == "" || *pluginsDir == "" {
		log.Fatal("usage: import-channels --sites /path/to/sites --plugins /path/to/plugins")
	}

	files, err := filepath.Glob(filepath.Join(*sitesDir, "*", "*.channels.xml"))
	if err != nil {
		log.Fatal(err)
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

			key := ch.SiteID + "|" + ch.Lang + "|" + region
			merged[pluginName][key] = channelEntry{
				SiteID:  ch.SiteID,
				Name:    name,
				Lang:    ch.Lang,
				Region:  region,
				XmltvID: ch.XmltvID,
				Logo:    ch.Logo,
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

		entries := make([]channelEntry, 0, len(channelMap))
		for _, e := range channelMap {
			entries = append(entries, e)
		}

		outPath := filepath.Join(pluginDir, "channels.json")
		outData, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			log.Printf("failed to marshal %s: %v", pluginName, err)
			continue
		}

		if err := os.WriteFile(outPath, outData, 0644); err != nil {
			log.Printf("failed to write %s: %v", outPath, err)
			continue
		}

		total += len(entries)
		pluginCount++
		fmt.Printf("%s: %d channels\n", pluginName, len(entries))
	}

	fmt.Printf("\nDone: %d unique channels across %d plugins\n", total, pluginCount)
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
