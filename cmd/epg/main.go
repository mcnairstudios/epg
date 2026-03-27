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
	"github.com/gavinmcnair/epg/internal/scheduler"
	"github.com/gavinmcnair/epg/internal/source"
	"github.com/gavinmcnair/epg/internal/store"

	_ "github.com/gavinmcnair/epg/plugins/abcau"
	_ "github.com/gavinmcnair/epg/plugins/allente"
	_ "github.com/gavinmcnair/epg/plugins/astro"
	_ "github.com/gavinmcnair/epg/plugins/beinsports"
	_ "github.com/gavinmcnair/epg/plugins/canalplus"
	_ "github.com/gavinmcnair/epg/plugins/cosmote"
	_ "github.com/gavinmcnair/epg/plugins/cubmu"
	_ "github.com/gavinmcnair/epg/plugins/digiturk"
	_ "github.com/gavinmcnair/epg/plugins/directvar"
	_ "github.com/gavinmcnair/epg/plugins/dishtvIn"
	_ "github.com/gavinmcnair/epg/plugins/distrotv"
	_ "github.com/gavinmcnair/epg/plugins/dstv"
	_ "github.com/gavinmcnair/epg/plugins/elcinema"
	_ "github.com/gavinmcnair/epg/plugins/epg112114"
	_ "github.com/gavinmcnair/epg/plugins/firstmedia"
	_ "github.com/gavinmcnair/epg/plugins/foxsportsau"
	_ "github.com/gavinmcnair/epg/plugins/freetv"
	_ "github.com/gavinmcnair/epg/plugins/freeviewuk"
	_ "github.com/gavinmcnair/epg/plugins/guidatv"
	_ "github.com/gavinmcnair/epg/plugins/horizontv"
	_ "github.com/gavinmcnair/epg/plugins/i24news"
	_ "github.com/gavinmcnair/epg/plugins/iltalehti"
	_ "github.com/gavinmcnair/epg/plugins/imjhnz"
	_ "github.com/gavinmcnair/epg/plugins/indihome"
	_ "github.com/gavinmcnair/epg/plugins/magticom"
	_ "github.com/gavinmcnair/epg/plugins/mediaklikk"
	_ "github.com/gavinmcnair/epg/plugins/knrgl"
	_ "github.com/gavinmcnair/epg/plugins/kvffo"
	_ "github.com/gavinmcnair/epg/plugins/melita"
	_ "github.com/gavinmcnair/epg/plugins/meopt"
	_ "github.com/gavinmcnair/epg/plugins/mitv"
	_ "github.com/gavinmcnair/epg/plugins/mncvision"
	_ "github.com/gavinmcnair/epg/plugins/movistarplus"
	_ "github.com/gavinmcnair/epg/plugins/mtelba"
	_ "github.com/gavinmcnair/epg/plugins/mtsrs"
	_ "github.com/gavinmcnair/epg/plugins/mytvsuper"
	_ "github.com/gavinmcnair/epg/plugins/neoio"
	_ "github.com/gavinmcnair/epg/plugins/nhkworld"
	_ "github.com/gavinmcnair/epg/plugins/nowplayer"
	_ "github.com/gavinmcnair/epg/plugins/ontvtonight"
	_ "github.com/gavinmcnair/epg/plugins/onetpl"
	_ "github.com/gavinmcnair/epg/plugins/orangees"
	_ "github.com/gavinmcnair/epg/plugins/osn"
	_ "github.com/gavinmcnair/epg/plugins/pickxbe"
	_ "github.com/gavinmcnair/epg/plugins/plextv"
	_ "github.com/gavinmcnair/epg/plugins/plutotv"
	_ "github.com/gavinmcnair/epg/plugins/porthu"
	_ "github.com/gavinmcnair/epg/plugins/programetvro"
	_ "github.com/gavinmcnair/epg/plugins/raiplay"
	_ "github.com/gavinmcnair/epg/plugins/rtmklik"
	_ "github.com/gavinmcnair/epg/plugins/rtppt"
	_ "github.com/gavinmcnair/epg/plugins/sattv"
	_ "github.com/gavinmcnair/epg/plugins/shahid"
	_ "github.com/gavinmcnair/epg/plugins/singtel"
	_ "github.com/gavinmcnair/epg/plugins/skyde"
	_ "github.com/gavinmcnair/epg/plugins/skynz"
	_ "github.com/gavinmcnair/epg/plugins/skyuk"
	_ "github.com/gavinmcnair/epg/plugins/starhub"
	_ "github.com/gavinmcnair/epg/plugins/tataplay"
	_ "github.com/gavinmcnair/epg/plugins/teleboy"
	_ "github.com/gavinmcnair/epg/plugins/tv24uk"
	_ "github.com/gavinmcnair/epg/plugins/tvguide"
	_ "github.com/gavinmcnair/epg/plugins/tvheute"
	_ "github.com/gavinmcnair/epg/plugins/tvlv"
	_ "github.com/gavinmcnair/epg/plugins/tvmailru"
	_ "github.com/gavinmcnair/epg/plugins/tvmimt"
	_ "github.com/gavinmcnair/epg/plugins/tvnu"
	_ "github.com/gavinmcnair/epg/plugins/tvpassport"
	_ "github.com/gavinmcnair/epg/plugins/tvtvus"
	_ "github.com/gavinmcnair/epg/plugins/vidio"
	_ "github.com/gavinmcnair/epg/plugins/virginmedia"
	_ "github.com/gavinmcnair/epg/plugins/vrtbe"
	_ "github.com/gavinmcnair/epg/plugins/wavve"
	_ "github.com/gavinmcnair/epg/plugins/xumotv"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data-dir", ".", "directory for persistent data files")
	days := flag.Int("days", 3, "number of days to grab")
	maxConn := flag.Int("max-connections", 3, "max concurrent connections per grab")
	schedule := flag.String("schedule", "", "cron schedule for automatic grabbing (e.g. '0 0 * * *' for daily)")
	flag.Parse()

	s := store.New(filepath.Join(*dataDir, "epg.gob"))
	if err := s.Load(); err != nil {
		log.Printf("warning: failed to load data: %v", err)
	}

	ext := source.NewExternalSourceManager(filepath.Join(*dataDir, "sources.gob"))
	if err := ext.Load(); err != nil {
		log.Printf("warning: failed to load external sources: %v", err)
	}

	users := auth.NewUserStore(filepath.Join(*dataDir, "users.gob"))
	if err := users.Load(); err != nil {
		log.Printf("warning: failed to load users: %v", err)
	}

	g := grabber.New(s, *maxConn, *days)
	srv := api.NewServer(s, g, ext, users)

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
