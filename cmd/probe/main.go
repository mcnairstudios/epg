package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"

	_ "github.com/gavinmcnair/epg/plugins/directvus"
	_ "github.com/gavinmcnair/epg/plugins/directvar"
	_ "github.com/gavinmcnair/epg/plugins/directvuy"
	_ "github.com/gavinmcnair/epg/plugins/epg112114"
	_ "github.com/gavinmcnair/epg/plugins/mtsrs"
	_ "github.com/gavinmcnair/epg/plugins/maxtvgo"
	_ "github.com/gavinmcnair/epg/plugins/galamtv"
	_ "github.com/gavinmcnair/epg/plugins/enaskylife"
	_ "github.com/gavinmcnair/epg/plugins/berrymedia"
	_ "github.com/gavinmcnair/epg/plugins/znbc"
	_ "github.com/gavinmcnair/epg/plugins/kanil"
	_ "github.com/gavinmcnair/epg/plugins/thesportplus"
	_ "github.com/gavinmcnair/epg/plugins/andorradifusio"
	_ "github.com/gavinmcnair/epg/plugins/clarobr"
	_ "github.com/gavinmcnair/epg/plugins/turksatkablo"
	_ "github.com/gavinmcnair/epg/plugins/myskyph"
	_ "github.com/gavinmcnair/epg/plugins/zapao"
	_ "github.com/gavinmcnair/epg/plugins/novacyprus"
)

var targets = []string{
	"directv.com", "directv.com.ar", "directv.com.uy",
	"epg.112114.xyz", "mts.rs", "maxtvgo.mk",
	"galamtv.kz", "ena.skylifetv.co.kr", "berrymedia.co.kr",
	"znbc.co.zm", "kan.org.il", "thesportplus.com",
	"andorradifusio.ad", "claro.com.br", "turksatkablo.com.tr",
	"mysky.com.ph", "zap.co.ao", "novacyprus.com",
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_ = ctx

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for _, name := range targets {
		p, ok := plugin.Get(name)
		if !ok {
			fmt.Printf("[%s] NOT FOUND\n", name)
			continue
		}

		channels, err := p.Channels()
		if err != nil {
			fmt.Printf("[%s] Channels() error: %v\n", name, err)
			continue
		}
		if len(channels) == 0 {
			fmt.Printf("[%s] no channels\n", name)
			continue
		}

		// Test first channel only
		ch := channels[0]
		fmt.Printf("[%s] testing ch=%q siteID=%q...\n", name, ch.Name, ch.SiteID)

		programs, err := testFetch(p, ch, today)
		if err != nil {
			fmt.Printf("[%s] FETCH ERROR: %v\n", name, err)
		} else {
			fmt.Printf("[%s] OK: %d programs\n", name, len(programs))
		}
	}
}

type fetcher interface {
	Fetch(ch domain.Channel, date time.Time) ([]domain.Program, error)
}

func testFetch(p fetcher, ch domain.Channel, date time.Time) ([]domain.Program, error) {
	done := make(chan struct{})
	var progs []domain.Program
	var ferr error
	go func() {
		defer close(done)
		progs, ferr = p.Fetch(ch, date)
	}()
	select {
	case <-done:
		return progs, ferr
	case <-time.After(30 * time.Second):
		fmt.Fprintf(os.Stderr, "TIMEOUT\n")
		return nil, fmt.Errorf("timeout")
	}
}
