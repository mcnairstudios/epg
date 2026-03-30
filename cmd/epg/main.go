package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gavinmcnair/epg/internal/api"
	"github.com/gavinmcnair/epg/internal/auth"
	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/scheduler"
	"github.com/gavinmcnair/epg/internal/seed"
	"github.com/gavinmcnair/epg/internal/source"
	"github.com/gavinmcnair/epg/internal/store"

	_ "github.com/gavinmcnair/epg/plugins/airtelxstream"
	_ "github.com/gavinmcnair/epg/plugins/abcau"
	_ "github.com/gavinmcnair/epg/plugins/allente"
	_ "github.com/gavinmcnair/epg/plugins/andorradifusio"
	_ "github.com/gavinmcnair/epg/plugins/artonline"
	_ "github.com/gavinmcnair/epg/plugins/astro"
	_ "github.com/gavinmcnair/epg/plugins/beinsports"
	_ "github.com/gavinmcnair/epg/plugins/bluech"
	_ "github.com/gavinmcnair/epg/plugins/canalplus"
	_ "github.com/gavinmcnair/epg/plugins/cgateslt"
	_ "github.com/gavinmcnair/epg/plugins/clarobr"
	_ "github.com/gavinmcnair/epg/plugins/ctcru"
	_ "github.com/gavinmcnair/epg/plugins/clarotvmais"
	_ "github.com/gavinmcnair/epg/plugins/cosmote"
	_ "github.com/gavinmcnair/epg/plugins/cubmu"
	_ "github.com/gavinmcnair/epg/plugins/deltanl"
	_ "github.com/gavinmcnair/epg/plugins/denstv"
	_ "github.com/gavinmcnair/epg/plugins/dtv8"
	_ "github.com/gavinmcnair/epg/plugins/derana"
	_ "github.com/gavinmcnair/epg/plugins/digiturk"
	_ "github.com/gavinmcnair/epg/plugins/directvar"
	_ "github.com/gavinmcnair/epg/plugins/directvus"
	_ "github.com/gavinmcnair/epg/plugins/dishtvIn2"
	_ "github.com/gavinmcnair/epg/plugins/distrotv"
	_ "github.com/gavinmcnair/epg/plugins/dnafi"
	_ "github.com/gavinmcnair/epg/plugins/dsmart"
	_ "github.com/gavinmcnair/epg/plugins/dstv"
	_ "github.com/gavinmcnair/epg/plugins/elcinema"
	_ "github.com/gavinmcnair/epg/plugins/energeek"
	_ "github.com/gavinmcnair/epg/plugins/entertainmentie"
	_ "github.com/gavinmcnair/epg/plugins/epgmaster"
	_ "github.com/gavinmcnair/epg/plugins/elpais"
	_ "github.com/gavinmcnair/epg/plugins/epg112114"
	_ "github.com/gavinmcnair/epg/plugins/firstmedia"
	_ "github.com/gavinmcnair/epg/plugins/foxsportsau"
	_ "github.com/gavinmcnair/epg/plugins/foxtelau"
	_ "github.com/gavinmcnair/epg/plugins/freetv"
	_ "github.com/gavinmcnair/epg/plugins/freeviewuk"
	_ "github.com/gavinmcnair/epg/plugins/getafteritmedia"
	_ "github.com/gavinmcnair/epg/plugins/frikanalen"
	_ "github.com/gavinmcnair/epg/plugins/galamtv"
	_ "github.com/gavinmcnair/epg/plugins/gatotv"
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
	_ "github.com/gavinmcnair/epg/plugins/magentatv"
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
	_ "github.com/gavinmcnair/epg/plugins/playeree"
	_ "github.com/gavinmcnair/epg/plugins/plextv"
	_ "github.com/gavinmcnair/epg/plugins/plutotv"
	_ "github.com/gavinmcnair/epg/plugins/porthu"
	_ "github.com/gavinmcnair/epg/plugins/programetvro"
	_ "github.com/gavinmcnair/epg/plugins/programmetv"
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
	_ "github.com/gavinmcnair/epg/plugins/skyit"
	_ "github.com/gavinmcnair/epg/plugins/skyuk"
	_ "github.com/gavinmcnair/epg/plugins/starhub"
	_ "github.com/gavinmcnair/epg/plugins/stod2"
	_ "github.com/gavinmcnair/epg/plugins/streamingtvguides"
	_ "github.com/gavinmcnair/epg/plugins/superguidatv"
	_ "github.com/gavinmcnair/epg/plugins/taiwanplus"
	_ "github.com/gavinmcnair/epg/plugins/tapdmv"
	_ "github.com/gavinmcnair/epg/plugins/tataplay"
	_ "github.com/gavinmcnair/epg/plugins/telecablesat"
	_ "github.com/gavinmcnair/epg/plugins/teleboy"
	_ "github.com/gavinmcnair/epg/plugins/telemach"
	_ "github.com/gavinmcnair/epg/plugins/teliatv"
	_ "github.com/gavinmcnair/epg/plugins/telenet"
	_ "github.com/gavinmcnair/epg/plugins/toonami"
	_ "github.com/gavinmcnair/epg/plugins/tvireland"
	_ "github.com/gavinmcnair/epg/plugins/tv24se"
	_ "github.com/gavinmcnair/epg/plugins/tv24uk"
	_ "github.com/gavinmcnair/epg/plugins/tvboxbg"
	_ "github.com/gavinmcnair/epg/plugins/tvcctv"
	_ "github.com/gavinmcnair/epg/plugins/tvdirbg"
	_ "github.com/gavinmcnair/epg/plugins/tvguide"
	_ "github.com/gavinmcnair/epg/plugins/tvinsider"
	_ "github.com/gavinmcnair/epg/plugins/tvhebdo"
	_ "github.com/gavinmcnair/epg/plugins/tvheute"
	_ "github.com/gavinmcnair/epg/plugins/tvlv"
	_ "github.com/gavinmcnair/epg/plugins/tvmailru"
	_ "github.com/gavinmcnair/epg/plugins/tvmimt"
	_ "github.com/gavinmcnair/epg/plugins/tvmusorhu"
	_ "github.com/gavinmcnair/epg/plugins/tvnu"
	_ "github.com/gavinmcnair/epg/plugins/tvpostlu"
	_ "github.com/gavinmcnair/epg/plugins/tvpassport"
	_ "github.com/gavinmcnair/epg/plugins/tvtvus"
	_ "github.com/gavinmcnair/epg/plugins/vidio"
	_ "github.com/gavinmcnair/epg/plugins/virginmedia"
	_ "github.com/gavinmcnair/epg/plugins/visionplus"
	_ "github.com/gavinmcnair/epg/plugins/vivoplay"
	_ "github.com/gavinmcnair/epg/plugins/vrtbe"
	_ "github.com/gavinmcnair/epg/plugins/walesi"
	_ "github.com/gavinmcnair/epg/plugins/wavve"
	_ "github.com/gavinmcnair/epg/plugins/whaletvplus"
	_ "github.com/gavinmcnair/epg/plugins/winplay"
	_ "github.com/gavinmcnair/epg/plugins/xumotv"
	_ "github.com/gavinmcnair/epg/plugins/zap2it"
	_ "github.com/gavinmcnair/epg/plugins/ziggogo"
	_ "github.com/gavinmcnair/epg/plugins/znbc"

	_ "github.com/gavinmcnair/epg/plugins/gigatv"
	_ "github.com/gavinmcnair/epg/plugins/kplusvn"
	_ "github.com/gavinmcnair/epg/plugins/monprogrammetv"
	_ "github.com/gavinmcnair/epg/plugins/myafn"
	_ "github.com/gavinmcnair/epg/plugins/rtsch"
	_ "github.com/gavinmcnair/epg/plugins/rthkhk"
	_ "github.com/gavinmcnair/epg/plugins/sibaco"
	_ "github.com/gavinmcnair/epg/plugins/sjonvarp"

	_ "github.com/gavinmcnair/epg/plugins/yesil"
	_ "github.com/gavinmcnair/epg/plugins/tvplustr"
	_ "github.com/gavinmcnair/epg/plugins/turksatkablo"
	_ "github.com/gavinmcnair/epg/plugins/cytacy"
	_ "github.com/gavinmcnair/epg/plugins/digeagr"
	_ "github.com/gavinmcnair/epg/plugins/skyperfect"
	_ "github.com/gavinmcnair/epg/plugins/myjcom"
	_ "github.com/gavinmcnair/epg/plugins/iptvxone"

	_ "github.com/gavinmcnair/epg/plugins/directvuy"
	_ "github.com/gavinmcnair/epg/plugins/musortv"
	_ "github.com/gavinmcnair/epg/plugins/trueid"
	_ "github.com/gavinmcnair/epg/plugins/tvcesoir"
	_ "github.com/gavinmcnair/epg/plugins/tvkaista"
	_ "github.com/gavinmcnair/epg/plugins/tvmustra"
	_ "github.com/gavinmcnair/epg/plugins/tvsmscz"
	_ "github.com/gavinmcnair/epg/plugins/vodafonept"

	_ "github.com/gavinmcnair/epg/plugins/anteltv"
	_ "github.com/gavinmcnair/epg/plugins/chadama"
	_ "github.com/gavinmcnair/epg/plugins/clickthecity"
	_ "github.com/gavinmcnair/epg/plugins/mediaset"
	_ "github.com/gavinmcnair/epg/plugins/mxtvjp"
	_ "github.com/gavinmcnair/epg/plugins/tvgidsnl"

	_ "github.com/gavinmcnair/epg/plugins/cableplus"
	_ "github.com/gavinmcnair/epg/plugins/enaskylife"
	_ "github.com/gavinmcnair/epg/plugins/epgshare"
	_ "github.com/gavinmcnair/epg/plugins/hrvatskitelekom"
	_ "github.com/gavinmcnair/epg/plugins/mojiid"
	_ "github.com/gavinmcnair/epg/plugins/nuevosiglo"
	_ "github.com/gavinmcnair/epg/plugins/reportv"
	_ "github.com/gavinmcnair/epg/plugins/siolnet"
	_ "github.com/gavinmcnair/epg/plugins/sportsnetca"
	_ "github.com/gavinmcnair/epg/plugins/t2net"
	_ "github.com/gavinmcnair/epg/plugins/tccuy"
	_ "github.com/gavinmcnair/epg/plugins/tvprofil"
	_ "github.com/gavinmcnair/epg/plugins/webtvsk"

	_ "github.com/gavinmcnair/epg/plugins/berrymedia"
	_ "github.com/gavinmcnair/epg/plugins/mediagenie"
	_ "github.com/gavinmcnair/epg/plugins/tving"
	_ "github.com/gavinmcnair/epg/plugins/watchyourtv"
	_ "github.com/gavinmcnair/epg/plugins/worldfishing"
	_ "github.com/gavinmcnair/epg/plugins/zapao"

	_ "github.com/gavinmcnair/epg/plugins/arianatv"
	_ "github.com/gavinmcnair/epg/plugins/awilime"
	_ "github.com/gavinmcnair/epg/plugins/telebilbao"
	_ "github.com/gavinmcnair/epg/plugins/tvarenasport"
	_ "github.com/gavinmcnair/epg/plugins/tvarenasporthr"
	_ "github.com/gavinmcnair/epg/plugins/tvcubana"

	_ "github.com/gavinmcnair/epg/plugins/antennaeurope"
	_ "github.com/gavinmcnair/epg/plugins/antennapacific"
	_ "github.com/gavinmcnair/epg/plugins/antennasatellite"
	_ "github.com/gavinmcnair/epg/plugins/thesportplus"
	_ "github.com/gavinmcnair/epg/plugins/tivutv"

	_ "github.com/gavinmcnair/epg/plugins/meuguia"
	_ "github.com/gavinmcnair/epg/plugins/nhkjp"
	_ "github.com/gavinmcnair/epg/plugins/ninetv"
	_ "github.com/gavinmcnair/epg/plugins/optosic"

	_ "github.com/gavinmcnair/epg/plugins/guiadetv"
	_ "github.com/gavinmcnair/epg/plugins/ionplustv"
	_ "github.com/gavinmcnair/epg/plugins/rotana"
	_ "github.com/gavinmcnair/epg/plugins/royatv"
	_ "github.com/gavinmcnair/epg/plugins/snrtma"
	_ "github.com/gavinmcnair/epg/plugins/startimes"
	_ "github.com/gavinmcnair/epg/plugins/zuragtmn"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data-dir", ".", "directory for persistent data files")
	days := flag.Int("days", 3, "number of days to grab")
	maxConn := flag.Int("max-connections", 3, "max concurrent connections per grab")
	schedule := flag.String("schedule", "", "cron schedule for automatic grabbing (e.g. '0 0 * * *' for daily)")
	rolling := flag.Bool("rolling", false, "enable rolling grabber (grabs each source at midnight in its timezone)")
	rollingParallel := flag.Int("rolling-parallel", 2, "max parallel grabs for rolling grabber")
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

	if added := seed.SeedExternalSources(ext); added > 0 {
		log.Printf("seeded %d external sources", added)
	}

	disabled := store.NewDisabledStore(filepath.Join(*dataDir, "disabled.gob"))
	if err := disabled.Load(); err != nil {
		log.Printf("warning: failed to load disabled sources: %v", err)
	}

	users := auth.NewUserStore(filepath.Join(*dataDir, "users.gob"))
	if err := users.Load(); err != nil {
		log.Printf("warning: failed to load users: %v", err)
	}

	if !users.HasAdminWithPassword() {
		if err := users.EnsureAdmin("admin", "admin"); err != nil {
			log.Printf("warning: failed to create admin user: %v", err)
		} else {
			if err := users.Save(); err != nil {
				log.Printf("warning: failed to save admin user: %v", err)
			}
			log.Printf("default admin credentials: admin / admin")
		}
	}

	g := grabber.New(s, *maxConn, *days)
	srv := api.NewServer(s, g, ext, users, disabled)

	var sched *scheduler.Scheduler
	if *schedule != "" {
		sched = scheduler.New(g, s, disabled)
		if err := sched.Start(*schedule); err != nil {
			log.Fatalf("failed to start scheduler: %v", err)
		}
	}

	var rg *scheduler.RollingGrabber
	if *rolling {
		rg = scheduler.NewRollingGrabber(g, s, *rollingParallel)
		rg.Start()
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: srv,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("signal received, shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("starting EPG server on %s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	var drainWg sync.WaitGroup
	if sched != nil {
		drainWg.Add(1)
		go func() { defer drainWg.Done(); sched.Stop() }()
	}
	if rg != nil {
		drainWg.Add(1)
		go func() { defer drainWg.Done(); rg.Stop() }()
	}
	drainWg.Wait()

	s.Close()

	log.Println("saving data...")
	s.Save()
	if err := ext.Save(); err != nil {
		log.Printf("warning: failed to save external sources: %v", err)
	}
	if err := users.Save(); err != nil {
		log.Printf("warning: failed to save users: %v", err)
	}
	if err := disabled.Save(); err != nil {
		log.Printf("warning: failed to save disabled sources: %v", err)
	}
	log.Println("shutdown complete")
}
