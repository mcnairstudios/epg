package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gavinmcnair/epg/internal/api"
	"github.com/gavinmcnair/epg/internal/grabber"
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
	dataPath := flag.String("data", "epg.gob", "path to gob data file")
	days := flag.Int("days", 3, "number of days to grab")
	maxConn := flag.Int("max-connections", 3, "max concurrent connections per grab")
	flag.Parse()

	s := store.New(*dataPath)
	if err := s.Load(); err != nil {
		log.Printf("warning: failed to load data: %v", err)
	}

	g := grabber.New(s, *maxConn, *days)
	srv := api.NewServer(s, g)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down, saving data...")
		if err := s.Save(); err != nil {
			log.Printf("failed to save data: %v", err)
		}
		os.Exit(0)
	}()

	log.Printf("starting EPG server on %s", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatal(err)
	}
}
