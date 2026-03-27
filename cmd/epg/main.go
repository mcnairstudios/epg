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
	"github.com/gavinmcnair/epg/internal/source"
	"github.com/gavinmcnair/epg/internal/store"

	_ "github.com/gavinmcnair/epg/plugins/foxsportsau"
	_ "github.com/gavinmcnair/epg/plugins/imjhnz"
	_ "github.com/gavinmcnair/epg/plugins/magticom"
	_ "github.com/gavinmcnair/epg/plugins/plutotv"
	_ "github.com/gavinmcnair/epg/plugins/singtel"
	_ "github.com/gavinmcnair/epg/plugins/skyuk"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data-dir", ".", "directory for persistent data files")
	days := flag.Int("days", 3, "number of days to grab")
	maxConn := flag.Int("max-connections", 3, "max concurrent connections per grab")
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
