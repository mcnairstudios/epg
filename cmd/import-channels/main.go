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
