package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/epg/internal/logocache"

	_ "github.com/gavinmcnair/epg/plugins/abcau"
	_ "github.com/gavinmcnair/epg/plugins/airtelxstream"
	_ "github.com/gavinmcnair/epg/plugins/allente"
	_ "github.com/gavinmcnair/epg/plugins/andorradifusio"
	_ "github.com/gavinmcnair/epg/plugins/antennaeurope"
	_ "github.com/gavinmcnair/epg/plugins/antennapacific"
	_ "github.com/gavinmcnair/epg/plugins/antennasatellite"
	_ "github.com/gavinmcnair/epg/plugins/anteltv"
	_ "github.com/gavinmcnair/epg/plugins/arianatv"
	_ "github.com/gavinmcnair/epg/plugins/artonline"
	_ "github.com/gavinmcnair/epg/plugins/astro"
	_ "github.com/gavinmcnair/epg/plugins/awilime"
	_ "github.com/gavinmcnair/epg/plugins/beinsports"
	_ "github.com/gavinmcnair/epg/plugins/berrymedia"
	_ "github.com/gavinmcnair/epg/plugins/bluech"
	_ "github.com/gavinmcnair/epg/plugins/cableplus"
	_ "github.com/gavinmcnair/epg/plugins/canalplus"
	_ "github.com/gavinmcnair/epg/plugins/chadama"
	_ "github.com/gavinmcnair/epg/plugins/cgateslt"
	_ "github.com/gavinmcnair/epg/plugins/clarobr"
	_ "github.com/gavinmcnair/epg/plugins/clarotvmais"
	_ "github.com/gavinmcnair/epg/plugins/clickthecity"
	_ "github.com/gavinmcnair/epg/plugins/cosmote"
	_ "github.com/gavinmcnair/epg/plugins/ctcru"
	_ "github.com/gavinmcnair/epg/plugins/cubmu"
	_ "github.com/gavinmcnair/epg/plugins/cytacy"
	_ "github.com/gavinmcnair/epg/plugins/deltanl"
	_ "github.com/gavinmcnair/epg/plugins/denstv"
	_ "github.com/gavinmcnair/epg/plugins/derana"
	_ "github.com/gavinmcnair/epg/plugins/digeagr"
	_ "github.com/gavinmcnair/epg/plugins/digiturk"
	_ "github.com/gavinmcnair/epg/plugins/directvar"
	_ "github.com/gavinmcnair/epg/plugins/directvus"
	_ "github.com/gavinmcnair/epg/plugins/directvuy"
	_ "github.com/gavinmcnair/epg/plugins/dishtvIn2"
	_ "github.com/gavinmcnair/epg/plugins/distrotv"
	_ "github.com/gavinmcnair/epg/plugins/dnafi"
	_ "github.com/gavinmcnair/epg/plugins/dsmart"
	_ "github.com/gavinmcnair/epg/plugins/dstv"
	_ "github.com/gavinmcnair/epg/plugins/dtv8"
	_ "github.com/gavinmcnair/epg/plugins/elcinema"
	_ "github.com/gavinmcnair/epg/plugins/elpais"
	_ "github.com/gavinmcnair/epg/plugins/enaskylife"
	_ "github.com/gavinmcnair/epg/plugins/energeek"
	_ "github.com/gavinmcnair/epg/plugins/entertainmentie"
	_ "github.com/gavinmcnair/epg/plugins/epg112114"
	_ "github.com/gavinmcnair/epg/plugins/epgmaster"
	_ "github.com/gavinmcnair/epg/plugins/epgshare"
	_ "github.com/gavinmcnair/epg/plugins/firstmedia"
	_ "github.com/gavinmcnair/epg/plugins/foxsportsau"
	_ "github.com/gavinmcnair/epg/plugins/foxtelau"
	_ "github.com/gavinmcnair/epg/plugins/freetv"
	_ "github.com/gavinmcnair/epg/plugins/freeviewuk"
	_ "github.com/gavinmcnair/epg/plugins/frikanalen"
	_ "github.com/gavinmcnair/epg/plugins/galamtv"
	_ "github.com/gavinmcnair/epg/plugins/gatotv"
	_ "github.com/gavinmcnair/epg/plugins/getafteritmedia"
	_ "github.com/gavinmcnair/epg/plugins/gigatv"
	_ "github.com/gavinmcnair/epg/plugins/guiadetv"
	_ "github.com/gavinmcnair/epg/plugins/guidatv"
	_ "github.com/gavinmcnair/epg/plugins/horizontv"
	_ "github.com/gavinmcnair/epg/plugins/hoytv"
	_ "github.com/gavinmcnair/epg/plugins/hrvatskitelekom"
	_ "github.com/gavinmcnair/epg/plugins/i24news"
	_ "github.com/gavinmcnair/epg/plugins/iltalehti"
	_ "github.com/gavinmcnair/epg/plugins/imjhnz"
	_ "github.com/gavinmcnair/epg/plugins/indihome"
	_ "github.com/gavinmcnair/epg/plugins/ionplustv"
	_ "github.com/gavinmcnair/epg/plugins/ipkotv"
	_ "github.com/gavinmcnair/epg/plugins/iptvxone"
	_ "github.com/gavinmcnair/epg/plugins/jiotv"
	_ "github.com/gavinmcnair/epg/plugins/kanil"
	_ "github.com/gavinmcnair/epg/plugins/knrgl"
	_ "github.com/gavinmcnair/epg/plugins/kplusvn"
	_ "github.com/gavinmcnair/epg/plugins/kvffo"
	_ "github.com/gavinmcnair/epg/plugins/magentaat"
	_ "github.com/gavinmcnair/epg/plugins/magentatv"
	_ "github.com/gavinmcnair/epg/plugins/magticom"
	_ "github.com/gavinmcnair/epg/plugins/maxtvgo"
	_ "github.com/gavinmcnair/epg/plugins/mediaset"
	_ "github.com/gavinmcnair/epg/plugins/mediagenie"
	_ "github.com/gavinmcnair/epg/plugins/mediaklikk"
	_ "github.com/gavinmcnair/epg/plugins/melita"
	_ "github.com/gavinmcnair/epg/plugins/meopt"
	_ "github.com/gavinmcnair/epg/plugins/mewatch"
	_ "github.com/gavinmcnair/epg/plugins/meuguia"
	_ "github.com/gavinmcnair/epg/plugins/mitv"
	_ "github.com/gavinmcnair/epg/plugins/mncvision"
	_ "github.com/gavinmcnair/epg/plugins/mojiid"
	_ "github.com/gavinmcnair/epg/plugins/monprogrammetv"
	_ "github.com/gavinmcnair/epg/plugins/movistarpe"
	_ "github.com/gavinmcnair/epg/plugins/movistarplus"
	_ "github.com/gavinmcnair/epg/plugins/mtelba"
	_ "github.com/gavinmcnair/epg/plugins/mtsrs"
	_ "github.com/gavinmcnair/epg/plugins/mujtvprogram"
	_ "github.com/gavinmcnair/epg/plugins/musortv"
	_ "github.com/gavinmcnair/epg/plugins/myskyph"
	_ "github.com/gavinmcnair/epg/plugins/mytelly"
	_ "github.com/gavinmcnair/epg/plugins/mytvsuper"
	_ "github.com/gavinmcnair/epg/plugins/mxtvjp"
	_ "github.com/gavinmcnair/epg/plugins/myjcom"
	_ "github.com/gavinmcnair/epg/plugins/neoio"
	_ "github.com/gavinmcnair/epg/plugins/nhkjp"
	_ "github.com/gavinmcnair/epg/plugins/nhkworld"
	_ "github.com/gavinmcnair/epg/plugins/nhlcom"
	_ "github.com/gavinmcnair/epg/plugins/ninetv"
	_ "github.com/gavinmcnair/epg/plugins/nostv"
	_ "github.com/gavinmcnair/epg/plugins/novacyprus"
	_ "github.com/gavinmcnair/epg/plugins/novasports"
	_ "github.com/gavinmcnair/epg/plugins/nowplayer"
	_ "github.com/gavinmcnair/epg/plugins/nuevosiglo"
	_ "github.com/gavinmcnair/epg/plugins/nzxmltv"
	_ "github.com/gavinmcnair/epg/plugins/onetpl"
	_ "github.com/gavinmcnair/epg/plugins/ontvtonight"
	_ "github.com/gavinmcnair/epg/plugins/optosic"
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
	_ "github.com/gavinmcnair/epg/plugins/reportv"
	_ "github.com/gavinmcnair/epg/plugins/rikstv"
	_ "github.com/gavinmcnair/epg/plugins/rotana"
	_ "github.com/gavinmcnair/epg/plugins/royatv"
	_ "github.com/gavinmcnair/epg/plugins/rtmklik"
	_ "github.com/gavinmcnair/epg/plugins/rtppt"
	_ "github.com/gavinmcnair/epg/plugins/rtsch"
	_ "github.com/gavinmcnair/epg/plugins/rthkhk"
	_ "github.com/gavinmcnair/epg/plugins/ruvis"
	_ "github.com/gavinmcnair/epg/plugins/sattv"
	_ "github.com/gavinmcnair/epg/plugins/sfrfr"
	_ "github.com/gavinmcnair/epg/plugins/shahid"
	_ "github.com/gavinmcnair/epg/plugins/sibaco"
	_ "github.com/gavinmcnair/epg/plugins/singtel"
	_ "github.com/gavinmcnair/epg/plugins/siolnet"
	_ "github.com/gavinmcnair/epg/plugins/sjonvarp"
	_ "github.com/gavinmcnair/epg/plugins/skyde"
	_ "github.com/gavinmcnair/epg/plugins/skyit"
	_ "github.com/gavinmcnair/epg/plugins/skynz"
	_ "github.com/gavinmcnair/epg/plugins/skyuk"
	_ "github.com/gavinmcnair/epg/plugins/skyperfect"
	_ "github.com/gavinmcnair/epg/plugins/snrtma"
	_ "github.com/gavinmcnair/epg/plugins/sportsnetca"
	_ "github.com/gavinmcnair/epg/plugins/startimes"
	_ "github.com/gavinmcnair/epg/plugins/starhub"
	_ "github.com/gavinmcnair/epg/plugins/stod2"
	_ "github.com/gavinmcnair/epg/plugins/streamingtvguides"
	_ "github.com/gavinmcnair/epg/plugins/superguidatv"
	_ "github.com/gavinmcnair/epg/plugins/t2net"
	_ "github.com/gavinmcnair/epg/plugins/taiwanplus"
	_ "github.com/gavinmcnair/epg/plugins/tapdmv"
	_ "github.com/gavinmcnair/epg/plugins/tataplay"
	_ "github.com/gavinmcnair/epg/plugins/tccuy"
	_ "github.com/gavinmcnair/epg/plugins/telecablesat"
	_ "github.com/gavinmcnair/epg/plugins/teleboy"
	_ "github.com/gavinmcnair/epg/plugins/telebilbao"
	_ "github.com/gavinmcnair/epg/plugins/telemach"
	_ "github.com/gavinmcnair/epg/plugins/teliatv"
	_ "github.com/gavinmcnair/epg/plugins/telenet"
	_ "github.com/gavinmcnair/epg/plugins/thesportplus"
	_ "github.com/gavinmcnair/epg/plugins/tivutv"
	_ "github.com/gavinmcnair/epg/plugins/toonami"
	_ "github.com/gavinmcnair/epg/plugins/trueid"
	_ "github.com/gavinmcnair/epg/plugins/tvarenasport"
	_ "github.com/gavinmcnair/epg/plugins/tvarenasporthr"
	_ "github.com/gavinmcnair/epg/plugins/tvcesoir"
	_ "github.com/gavinmcnair/epg/plugins/tvcctv"
	_ "github.com/gavinmcnair/epg/plugins/tvcubana"
	_ "github.com/gavinmcnair/epg/plugins/tvdirbg"
	_ "github.com/gavinmcnair/epg/plugins/tvboxbg"
	_ "github.com/gavinmcnair/epg/plugins/tvgidsnl"
	_ "github.com/gavinmcnair/epg/plugins/tvguide"
	_ "github.com/gavinmcnair/epg/plugins/tvhebdo"
	_ "github.com/gavinmcnair/epg/plugins/tvheute"
	_ "github.com/gavinmcnair/epg/plugins/tvinsider"
	_ "github.com/gavinmcnair/epg/plugins/tvireland"
	_ "github.com/gavinmcnair/epg/plugins/tvkaista"
	_ "github.com/gavinmcnair/epg/plugins/tvlv"
	_ "github.com/gavinmcnair/epg/plugins/tvmailru"
	_ "github.com/gavinmcnair/epg/plugins/tvmimt"
	_ "github.com/gavinmcnair/epg/plugins/tvmusorhu"
	_ "github.com/gavinmcnair/epg/plugins/tvmustra"
	_ "github.com/gavinmcnair/epg/plugins/tvnu"
	_ "github.com/gavinmcnair/epg/plugins/tvpassport"
	_ "github.com/gavinmcnair/epg/plugins/tvpostlu"
	_ "github.com/gavinmcnair/epg/plugins/tvplustr"
	_ "github.com/gavinmcnair/epg/plugins/tvprofil"
	_ "github.com/gavinmcnair/epg/plugins/tvsmscz"
	_ "github.com/gavinmcnair/epg/plugins/tvtvus"
	_ "github.com/gavinmcnair/epg/plugins/tv24se"
	_ "github.com/gavinmcnair/epg/plugins/tv24uk"
	_ "github.com/gavinmcnair/epg/plugins/turksatkablo"
	_ "github.com/gavinmcnair/epg/plugins/vidio"
	_ "github.com/gavinmcnair/epg/plugins/virginmedia"
	_ "github.com/gavinmcnair/epg/plugins/visionplus"
	_ "github.com/gavinmcnair/epg/plugins/vivoplay"
	_ "github.com/gavinmcnair/epg/plugins/vodafonept"
	_ "github.com/gavinmcnair/epg/plugins/vrtbe"
	_ "github.com/gavinmcnair/epg/plugins/walesi"
	_ "github.com/gavinmcnair/epg/plugins/watchyourtv"
	_ "github.com/gavinmcnair/epg/plugins/wavve"
	_ "github.com/gavinmcnair/epg/plugins/webtvsk"
	_ "github.com/gavinmcnair/epg/plugins/whaletvplus"
	_ "github.com/gavinmcnair/epg/plugins/winplay"
	_ "github.com/gavinmcnair/epg/plugins/worldfishing"
	_ "github.com/gavinmcnair/epg/plugins/xumotv"
	_ "github.com/gavinmcnair/epg/plugins/yesil"
	_ "github.com/gavinmcnair/epg/plugins/zap2it"
	_ "github.com/gavinmcnair/epg/plugins/zapao"
	_ "github.com/gavinmcnair/epg/plugins/ziggogo"
	_ "github.com/gavinmcnair/epg/plugins/znbc"
	_ "github.com/gavinmcnair/epg/plugins/zuragtmn"

	"github.com/gavinmcnair/epg/internal/plugin"
)

type job struct {
	key      string
	logoURL  string
}

func main() {
	outDir := flag.String("out", "logos", "output directory for downloaded logos")
	workers := flag.Int("workers", 20, "number of concurrent download workers")
	flag.Parse()

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatal().Err(err).Msg("failed to create output dir")
	}

	seen := make(map[string]string)
	for _, p := range plugin.List() {
		channels, err := p.Channels()
		if err != nil {
			log.Warn().Err(err).Str("plugin", p.Name()).Msg("failed to load channels")
			continue
		}
		for _, ch := range channels {
			if ch.Logo == "" {
				continue
			}
			key := logocache.SafeBase(ch.XmltvID)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; !ok {
				seen[key] = ch.Logo
			}
		}
	}
	log.Info().Int("count", len(seen)).Msg("found unique logo keys")

	jobs := make(chan job, len(seen))
	for k, u := range seen {
		jobs <- job{key: k, logoURL: u}
	}
	close(jobs)

	type result struct {
		key      string
		filename string
	}

	results := make(chan result, len(seen))
	client := &http.Client{Timeout: 30 * time.Second}

	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				filename, err := downloadLogo(client, *outDir, j.key, j.logoURL)
				if err != nil {
					log.Warn().Err(err).Str("key", j.key).Msg("skipping logo")
					continue
				}
				results <- result{key: j.key, filename: filename}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	manifest := make(map[string]string)
	for r := range results {
		manifest[r.key] = r.filename
	}

	manifestPath := filepath.Join(*outDir, "manifest.json")
	f, err := os.Create(manifestPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create manifest")
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		f.Close()
		log.Fatal().Err(err).Msg("failed to write manifest")
	}
	f.Close()

	log.Info().Int("downloaded", len(manifest)).Str("manifest", manifestPath).Msg("done")
}

func downloadLogo(client *http.Client, outDir, key, logoURL string) (string, error) {
	ext := urlExt(logoURL)
	if ext == "" {
		ext = ".png"
	}
	filename := key + ext
	destPath := filepath.Join(outDir, filename)

	if _, err := os.Stat(destPath); err == nil {
		return filename, nil
	}

	resp, err := client.Get(logoURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if ext == ".png" && contentType != "" {
		ext = extFromContentType(contentType)
		filename = key + ext
		destPath = filepath.Join(outDir, filename)
	}

	tmp := destPath + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmp)
		return "", err
	}
	out.Close()
	if err := os.Rename(tmp, destPath); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return filename, nil
}

func urlExt(u string) string {
	if idx := strings.LastIndex(u, "?"); idx != -1 {
		u = u[:idx]
	}
	if idx := strings.LastIndex(u, "."); idx != -1 {
		ext := u[idx:]
		switch strings.ToLower(ext) {
		case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
			return ext
		}
	}
	return ""
}

func extFromContentType(ct string) string {
	ct = strings.Split(ct, ";")[0]
	ct = strings.TrimSpace(ct)
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	}
	return ".png"
}

