package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gavinmcnair/epg/internal/api"
	"github.com/gavinmcnair/epg/internal/auth"
	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/scheduler"
	"github.com/gavinmcnair/epg/internal/source"
	"github.com/gavinmcnair/epg/internal/store"

	_ "github.com/gavinmcnair/epg/plugins/abcau"
	_ "github.com/gavinmcnair/epg/plugins/allente"
	_ "github.com/gavinmcnair/epg/plugins/astro"
	_ "github.com/gavinmcnair/epg/plugins/beinsports"
	_ "github.com/gavinmcnair/epg/plugins/bluech"
	_ "github.com/gavinmcnair/epg/plugins/canalplus"
	_ "github.com/gavinmcnair/epg/plugins/cgateslt"
	_ "github.com/gavinmcnair/epg/plugins/clarobr"
	_ "github.com/gavinmcnair/epg/plugins/clarotvmais"
	_ "github.com/gavinmcnair/epg/plugins/cosmote"
	_ "github.com/gavinmcnair/epg/plugins/cubmu"
	_ "github.com/gavinmcnair/epg/plugins/denstv"
	_ "github.com/gavinmcnair/epg/plugins/derana"
	_ "github.com/gavinmcnair/epg/plugins/digiturk"
	_ "github.com/gavinmcnair/epg/plugins/directvar"
	_ "github.com/gavinmcnair/epg/plugins/directvus"
	_ "github.com/gavinmcnair/epg/plugins/distrotv"
	_ "github.com/gavinmcnair/epg/plugins/dnafi"
	_ "github.com/gavinmcnair/epg/plugins/dsmart"
	_ "github.com/gavinmcnair/epg/plugins/dstv"
	_ "github.com/gavinmcnair/epg/plugins/elcinema"
	_ "github.com/gavinmcnair/epg/plugins/epg112114"
	_ "github.com/gavinmcnair/epg/plugins/firstmedia"
	_ "github.com/gavinmcnair/epg/plugins/foxsportsau"
	_ "github.com/gavinmcnair/epg/plugins/foxtelau"
	_ "github.com/gavinmcnair/epg/plugins/freetv"
	_ "github.com/gavinmcnair/epg/plugins/freeviewuk"
	_ "github.com/gavinmcnair/epg/plugins/frikanalen"
	_ "github.com/gavinmcnair/epg/plugins/galamtv"
	_ "github.com/gavinmcnair/epg/plugins/guidatv"
	_ "github.com/gavinmcnair/epg/plugins/horizontv"
	_ "github.com/gavinmcnair/epg/plugins/hoytv"
	_ "github.com/gavinmcnair/epg/plugins/i24news"
	_ "github.com/gavinmcnair/epg/plugins/iltalehti"
	_ "github.com/gavinmcnair/epg/plugins/imjhnz"
	_ "github.com/gavinmcnair/epg/plugins/indihome"
	_ "github.com/gavinmcnair/epg/plugins/ipkotv"
	_ "github.com/gavinmcnair/epg/plugins/jiotv"
	_ "github.com/gavinmcnair/epg/plugins/kanil"
	_ "github.com/gavinmcnair/epg/plugins/knrgl"
	_ "github.com/gavinmcnair/epg/plugins/kvffo"
	_ "github.com/gavinmcnair/epg/plugins/magentaat"
	_ "github.com/gavinmcnair/epg/plugins/magticom"
	_ "github.com/gavinmcnair/epg/plugins/maxtvgo"
	_ "github.com/gavinmcnair/epg/plugins/mediaklikk"
	_ "github.com/gavinmcnair/epg/plugins/melita"
	_ "github.com/gavinmcnair/epg/plugins/meopt"
	_ "github.com/gavinmcnair/epg/plugins/mewatch"
	_ "github.com/gavinmcnair/epg/plugins/mitv"
	_ "github.com/gavinmcnair/epg/plugins/mncvision"
	_ "github.com/gavinmcnair/epg/plugins/movistarpe"
	_ "github.com/gavinmcnair/epg/plugins/movistarplus"
	_ "github.com/gavinmcnair/epg/plugins/mtelba"
	_ "github.com/gavinmcnair/epg/plugins/mtsrs"
	_ "github.com/gavinmcnair/epg/plugins/mujtvprogram"
	_ "github.com/gavinmcnair/epg/plugins/myskyph"
	_ "github.com/gavinmcnair/epg/plugins/mytelly"
	_ "github.com/gavinmcnair/epg/plugins/mytvsuper"
	_ "github.com/gavinmcnair/epg/plugins/neoio"
	_ "github.com/gavinmcnair/epg/plugins/nhkworld"
	_ "github.com/gavinmcnair/epg/plugins/nhlcom"
	_ "github.com/gavinmcnair/epg/plugins/nostv"
	_ "github.com/gavinmcnair/epg/plugins/novacyprus"
	_ "github.com/gavinmcnair/epg/plugins/novasports"
	_ "github.com/gavinmcnair/epg/plugins/nowplayer"
	_ "github.com/gavinmcnair/epg/plugins/nzxmltv"
	_ "github.com/gavinmcnair/epg/plugins/onetpl"
	_ "github.com/gavinmcnair/epg/plugins/ontvtonight"
	_ "github.com/gavinmcnair/epg/plugins/orangees"
	_ "github.com/gavinmcnair/epg/plugins/orangefr2"
	_ "github.com/gavinmcnair/epg/plugins/osn"
	_ "github.com/gavinmcnair/epg/plugins/passienl"
	_ "github.com/gavinmcnair/epg/plugins/pbsguam"
	_ "github.com/gavinmcnair/epg/plugins/pickxbe"
	_ "github.com/gavinmcnair/epg/plugins/plextv"
	_ "github.com/gavinmcnair/epg/plugins/plutotv"
	_ "github.com/gavinmcnair/epg/plugins/porthu"
	_ "github.com/gavinmcnair/epg/plugins/programetvro"
	_ "github.com/gavinmcnair/epg/plugins/raiplay"
	_ "github.com/gavinmcnair/epg/plugins/rikstv"
	_ "github.com/gavinmcnair/epg/plugins/rtmklik"
	_ "github.com/gavinmcnair/epg/plugins/rtppt"
	_ "github.com/gavinmcnair/epg/plugins/ruvis"
	_ "github.com/gavinmcnair/epg/plugins/sattv"
	_ "github.com/gavinmcnair/epg/plugins/sfrfr"
	_ "github.com/gavinmcnair/epg/plugins/shahid"
	_ "github.com/gavinmcnair/epg/plugins/singtel"
	_ "github.com/gavinmcnair/epg/plugins/skyde"
	_ "github.com/gavinmcnair/epg/plugins/skynz"
	_ "github.com/gavinmcnair/epg/plugins/skyuk"
	_ "github.com/gavinmcnair/epg/plugins/starhub"
	_ "github.com/gavinmcnair/epg/plugins/stod2"
	_ "github.com/gavinmcnair/epg/plugins/streamingtvguides"
	_ "github.com/gavinmcnair/epg/plugins/taiwanplus"
	_ "github.com/gavinmcnair/epg/plugins/tapdmv"
	_ "github.com/gavinmcnair/epg/plugins/tataplay"
	_ "github.com/gavinmcnair/epg/plugins/teleboy"
	_ "github.com/gavinmcnair/epg/plugins/telemach"
	_ "github.com/gavinmcnair/epg/plugins/teliatv"
	_ "github.com/gavinmcnair/epg/plugins/toonami"
	_ "github.com/gavinmcnair/epg/plugins/tvireland"
	_ "github.com/gavinmcnair/epg/plugins/tv24uk"
	_ "github.com/gavinmcnair/epg/plugins/tvboxbg"
	_ "github.com/gavinmcnair/epg/plugins/tvcctv"
	_ "github.com/gavinmcnair/epg/plugins/tvdirbg"
	_ "github.com/gavinmcnair/epg/plugins/tvguide"
	_ "github.com/gavinmcnair/epg/plugins/tvinsider"
	_ "github.com/gavinmcnair/epg/plugins/tvheute"
	_ "github.com/gavinmcnair/epg/plugins/tvlv"
	_ "github.com/gavinmcnair/epg/plugins/tvmailru"
	_ "github.com/gavinmcnair/epg/plugins/tvmimt"
	_ "github.com/gavinmcnair/epg/plugins/tvmusorhu"
	_ "github.com/gavinmcnair/epg/plugins/tvnu"
	_ "github.com/gavinmcnair/epg/plugins/tvpassport"
	_ "github.com/gavinmcnair/epg/plugins/tvtvus"
	_ "github.com/gavinmcnair/epg/plugins/vidio"
	_ "github.com/gavinmcnair/epg/plugins/virginmedia"
	_ "github.com/gavinmcnair/epg/plugins/visionplus"
	_ "github.com/gavinmcnair/epg/plugins/vivoplay"
	_ "github.com/gavinmcnair/epg/plugins/vrtbe"
	_ "github.com/gavinmcnair/epg/plugins/walesi"
	_ "github.com/gavinmcnair/epg/plugins/wavve"
	_ "github.com/gavinmcnair/epg/plugins/winplay"
	_ "github.com/gavinmcnair/epg/plugins/xumotv"
	_ "github.com/gavinmcnair/epg/plugins/znbc"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data-dir", ".", "directory for persistent data files")
	days := flag.Int("days", 3, "number of days to grab")
	maxConn := flag.Int("max-connections", 3, "max concurrent connections per grab")
	schedule := flag.String("schedule", "", "cron schedule for automatic grabbing (e.g. '0 0 * * *' for daily)")
	flag.Parse()

	s := store.New(filepath.Join(*dataDir, "providers"))
	if err := s.Load(); err != nil {
		log.Printf("warning: failed to load data: %v", err)
	}

	for _, p := range plugin.List() {
		channels, err := p.Channels()
		if err != nil {
			log.Printf("warning: failed to load channels for %s: %v", p.Name(), err)
			continue
		}
		s.SetChannels(channels)
	}
	log.Printf("loaded channels from %d plugins", len(plugin.List()))

	ext := source.NewExternalSourceManager(filepath.Join(*dataDir, "sources.gob"))
	if err := ext.Load(); err != nil {
		log.Printf("warning: failed to load external sources: %v", err)
	}

	sleeping := store.NewSleepingStore(filepath.Join(*dataDir, "sleeping.gob"))
	if err := sleeping.Load(); err != nil {
		log.Printf("warning: failed to load sleeping sources: %v", err)
	}

	users := auth.NewUserStore(filepath.Join(*dataDir, "users.gob"))
	if err := users.Load(); err != nil {
		log.Printf("warning: failed to load users: %v", err)
	}

	g := grabber.New(s, *maxConn, *days)
	srv := api.NewServer(s, g, ext, users, sleeping)

	if *schedule != "" {
		sched := scheduler.New(g, s)
		if err := sched.Start(*schedule); err != nil {
			log.Fatalf("failed to start scheduler: %v", err)
		}
		defer sched.Stop()
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down, saving data...")
		s.Save()
		ext.Save()
		users.Save()
		os.Exit(0)
	}()

	log.Printf("starting EPG server on %s", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatal(err)
	}
}
