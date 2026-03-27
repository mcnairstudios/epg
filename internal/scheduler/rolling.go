package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/store"
)

var sourceTimezones = map[string]string{
	"freeview.co.uk":              "Europe/London",
	"sky.com":                     "Europe/London",
	"player.ee.co.uk":             "Europe/London",
	"tvireland.ie":                "Europe/Dublin",
	"mytelly.co.uk":               "Europe/London",
	"tvguide.com":                 "America/New_York",
	"tvtvus":                      "America/New_York",
	"tvinsider.com":               "America/New_York",
	"zap2it.com":                  "America/New_York",
	"streamingtvguides.com":       "America/Los_Angeles",
	"directv.com":                 "America/New_York",
	"pluto.tv":                    "America/New_York",
	"distro.tv":                   "America/New_York",
	"xumo.tv":                     "America/New_York",
	"sky.de":                      "Europe/Berlin",
	"web.magentatv.de":            "Europe/Berlin",
	"tvheute.at":                  "Europe/Vienna",
	"tv.magenta.at":               "Europe/Vienna",
	"tv.blue.ch":                  "Europe/Zurich",
	"teleboy.ch":                  "Europe/Zurich",
	"tv.sfr.fr":                   "Europe/Paris",
	"programme-tv.net":            "Europe/Paris",
	"canalplus.com":               "Europe/Paris",
	"raiplay.it":                  "Europe/Rome",
	"superguidatv.it":             "Europe/Rome",
	"guidatv.sky.it":              "Europe/Rome",
	"guida.tv":                    "Europe/Rome",
	"vrt.be":                      "Europe/Brussels",
	"telenet.tv":                  "Europe/Brussels",
	"pickx.be":                    "Europe/Brussels",
	"tv.nu":                       "Europe/Stockholm",
	"tv24.se":                     "Europe/Stockholm",
	"allente.no":                  "Europe/Oslo",
	"rikstv.no":                   "Europe/Oslo",
	"iltalehti.fi":                "Europe/Helsinki",
	"dna.fi":                      "Europe/Helsinki",
	"movistarplus.es":             "Europe/Madrid",
	"orangetv.orange.es":          "Europe/Madrid",
	"gatotv.com":                  "America/New_York",
	"rtp.pt":                      "Europe/Lisbon",
	"nostv.pt":                    "Europe/Lisbon",
	"meo.pt":                      "Europe/Lisbon",
	"programetv.ro":               "Europe/Bucharest",
	"mediaklikk.hu":               "Europe/Budapest",
	"tvmusor.hu":                  "Europe/Budapest",
	"cosmotetv.gr":                "Europe/Athens",
	"novasports.gr":               "Europe/Athens",
	"mujtvprogram.cz":             "Europe/Prague",
	"cgates.lt":                   "Europe/Vilnius",
	"ipko.tv":                     "Europe/Belgrade",
	"mts.rs":                      "Europe/Belgrade",
	"mtel.ba":                     "Europe/Sarajevo",
	"teliatv.ee":                  "Europe/Tallinn",
	"tv.lv":                       "Europe/Riga",
	"abc.net.au":                  "Australia/Sydney",
	"foxtel.com.au":               "Australia/Sydney",
	"nzxmltv.com":                 "Pacific/Auckland",
	"singtel.com":                 "Asia/Singapore",
	"starhubtvplus.com":           "Asia/Singapore",
	"mewatch.sg":                  "Asia/Singapore",
	"content.astro.com.my":        "Asia/Kuala_Lumpur",
	"mytvsuper.com":               "Asia/Hong_Kong",
	"hoy.tv":                      "Asia/Hong_Kong",
	"jiotv.com":                   "Asia/Kolkata",
	"airtelxstream.in":            "Asia/Kolkata",
	"tataplay.com":                "Asia/Kolkata",
	"tv.cctv.com":                 "Asia/Shanghai",
	"shahid.mbc.net":              "Asia/Dubai",
	"osn.com":                     "Asia/Dubai",
	"beinsports.com":              "Asia/Qatar",
	"dstv.com":                    "Africa/Johannesburg",
	"magticom.ge":                 "Asia/Tbilisi",
	"plex.tv":                     "America/New_York",
	"i.mjh.nz":                    "Australia/Sydney",
	"mi.tv":                       "America/Mexico_City",
	"neo.io":                      "Europe/Ljubljana",
	"port.hu":                     "Europe/Budapest",
	"tv.boxbg.net":                "Europe/Sofia",
	"elcinema.com":                "Africa/Cairo",
	"nhkworldpremium.com":         "Asia/Tokyo",
	"dsmart.com.tr":               "Europe/Istanbul",
	"tv.mail.ru":                  "Europe/Moscow",
	"tvhebdo.com":                 "America/Toronto",
	"tv.post.lu":                  "Europe/Luxembourg",
	"watch.whaletvplus.com":       "Asia/Seoul",
}

type RollingGrabber struct {
	grabber      *grabber.Grabber
	store        *store.Store
	mu           sync.Mutex
	running      bool
	stopCh       chan struct{}
	maxParallel  int
	lastGrabDate map[string]string
}

func NewRollingGrabber(g *grabber.Grabber, s *store.Store, maxParallel int) *RollingGrabber {
	if maxParallel < 1 {
		maxParallel = 2
	}
	return &RollingGrabber{
		grabber:      g,
		store:        s,
		maxParallel:  maxParallel,
		lastGrabDate: make(map[string]string),
	}
}

func (r *RollingGrabber) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.stopCh = make(chan struct{})
	r.mu.Unlock()

	go r.loop()
	log.Println("rolling grabber started")
}

func (r *RollingGrabber) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		close(r.stopCh)
		r.running = false
		log.Println("rolling grabber stopped")
	}
}

func (r *RollingGrabber) loop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.checkAndGrab()
		}
	}
}

func (r *RollingGrabber) checkAndGrab() {
	var toGrab []string

	for _, p := range plugin.List() {
		name := p.Name()
		tz := sourceTimezones[name]
		if tz == "" {
			tz = "UTC"
		}

		loc, err := time.LoadLocation(tz)
		if err != nil {
			continue
		}

		now := time.Now().In(loc)
		hour := now.Hour()
		dateKey := now.Format("2006-01-02")

		if hour >= 0 && hour < 1 {
			r.mu.Lock()
			lastDate := r.lastGrabDate[name]
			if lastDate != dateKey {
				r.lastGrabDate[name] = dateKey
				toGrab = append(toGrab, name)
			}
			r.mu.Unlock()
		}
	}

	if len(toGrab) == 0 {
		return
	}

	log.Printf("rolling grab: %d sources due", len(toGrab))

	sem := make(chan struct{}, r.maxParallel)
	var wg sync.WaitGroup

	for _, name := range toGrab {
		wg.Add(1)
		go func(source string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			log.Printf("rolling grab: starting %s", source)
			results, err := r.grabber.GrabSource(source)
			if err != nil {
				log.Printf("rolling grab: %s failed: %v", source, err)
				return
			}

			total := 0
			errors := 0
			for _, result := range results {
				total += result.Count
				if result.Err != nil {
					errors++
				}
			}

			r.store.RecordGrabStats(source, len(results), total, errors, 0)
			r.store.SaveSource(source)
			log.Printf("rolling grab: %s done (%d programs, %d errors)", source, total, errors)
		}(name)
	}

	wg.Wait()
}
