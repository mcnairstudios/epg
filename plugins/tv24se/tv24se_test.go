package tv24se_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tv24se"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 8, 26, 0, 0, 0, 0, time.UTC)
	items := tv24se.Parse(data, domain.Channel{XmltvID: "Test", Lang: "xx", SiteID: "test"}, date)

	if len(items) == 0 {
		t.Fatal("expected at least one program")
	}

	if len(items) != 34 {
		t.Fatalf("expected 34 programs, got %d", len(items))
	}

	if items[0].Title != "Morgonstudion" {
		t.Fatalf("expected title 'Morgonstudion', got %q", items[0].Title)
	}

	expectedStart := time.Date(2022, 8, 26, 4, 0, 0, 0, time.UTC)
	if !items[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, items[0].Start)
	}

	expectedStop := time.Date(2022, 8, 26, 7, 10, 0, 0, time.UTC)
	if !items[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, items[0].Stop)
	}

	last := items[33]
	if last.Title != "Uppdrag granskning" {
		t.Fatalf("expected last title 'Uppdrag granskning', got %q", last.Title)
	}

	expectedLastStart := time.Date(2022, 8, 27, 5, 20, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2022, 8, 27, 5, 50, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseDetails(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	item := tv24se.ParseDetails(data)
	if item == nil {
		t.Fatal("expected non-nil result")
	}

	if item.Image != "https://jrsy.tmsimg.com/assets/p14436175_i_h9_ad.jpg" {
		t.Fatalf("expected image URL, got %q", item.Image)
	}

	if item.Description != "Dagens viktigaste nyheter och analyser med ständiga uppdateringar. Vi sänder direkt inrikes- och utrikesnyheter inklusive sport, kultur och nöje. Dessutom intervjuer med aktuella gäster. Nyhetssammanfattningar varje kvart med start kl 06.00." {
		t.Fatalf("unexpected description: %q", item.Description)
	}

	expectedActors := []string{"Carolina Neurath", "Karin Magnusson", "Pelle Nilsson", "Ted Wigren"}
	if len(item.Actors) != len(expectedActors) {
		t.Fatalf("expected %d actors, got %d", len(expectedActors), len(item.Actors))
	}
	for i, a := range expectedActors {
		if item.Actors[i] != a {
			t.Fatalf("expected actor %q, got %q", a, item.Actors[i])
		}
	}
}

func TestParseDetailsWithMetadata(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program2.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	item := tv24se.ParseDetails(data)
	if item == nil {
		t.Fatal("expected non-nil result")
	}

	if item.Image != "https://jrsy.tmsimg.com/assets/p22818697_e_h9_aa.jpg" {
		t.Fatalf("expected image URL, got %q", item.Image)
	}

	if item.Category != "Dokumentär / Samhällsfrågor" {
		t.Fatalf("expected category 'Dokumentär / Samhällsfrågor', got %q", item.Category)
	}

	if item.Season != 23 {
		t.Fatalf("expected season 23, got %d", item.Season)
	}

	if item.Episode != 5 {
		t.Fatalf("expected episode 5, got %d", item.Episode)
	}

	if item.SubTitle != "Där mördare går fria" {
		t.Fatalf("expected subtitle 'Där mördare går fria', got %q", item.SubTitle)
	}

	expectedActors := []string{"Karin Mattisson", "Ali Fegan"}
	if len(item.Actors) != len(expectedActors) {
		t.Fatalf("expected %d actors, got %d", len(expectedActors), len(item.Actors))
	}
	for i, a := range expectedActors {
		if item.Actors[i] != a {
			t.Fatalf("expected actor %q, got %q", a, item.Actors[i])
		}
	}
}

func TestParseEmptyContent(t *testing.T) {
	date := time.Date(2022, 8, 26, 0, 0, 0, 0, time.UTC)
	items := tv24se.Parse([]byte(""), domain.Channel{XmltvID: "Test", Lang: "xx", SiteID: "test"}, date)
	if len(items) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(items))
	}
}

func TestParseDetailsInvalidJSON(t *testing.T) {
	item := tv24se.ParseDetails([]byte("not json"))
	if item != nil {
		t.Fatal("expected nil on invalid JSON")
	}
}

func TestChannelsLoad(t *testing.T) {
	p := &tv24se.TestableProvider{}
	channels, err := p.Channels()
	if err != nil {
		t.Fatalf("channels load failed: %v", err)
	}
	if len(channels) == 0 {
		t.Fatal("expected at least one channel")
	}
	found := false
	for _, ch := range channels {
		if ch.Source != "tv24.se" {
			t.Fatalf("expected source 'tv24.se', got %q", ch.Source)
		}
		if ch.SiteID == "svt1" {
			found = true
		}
	}
	_ = found
}

type fakeTransport struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f.handler(req)
}

func TestFetchIntegration(t *testing.T) {
	contentData, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read content: %v", err)
	}

	program1Data, err := os.ReadFile(testdataPath("program1.json"))
	if err != nil {
		t.Fatalf("failed to read program1: %v", err)
	}

	program2Data, err := os.ReadFile(testdataPath("program2.json"))
	if err != nil {
		t.Fatalf("failed to read program2: %v", err)
	}

	transport := &fakeTransport{
		handler: func(req *http.Request) (*http.Response, error) {
			var body []byte
			switch {
			case req.URL.Path == "/x/channel/svt1/0/2022-08-26":
				body = contentData
			case req.URL.Path == "/x/b/rh7f40-1hkm/0/0":
				body = program1Data
			case req.URL.Path == "/x/b/rh9dhc-1hkm/0/0":
				body = program2Data
			default:
				body = []byte(`{}`)
			}

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(string(body))),
			}, nil
		},
	}

	p := tv24se.NewWithClient(&http.Client{Transport: transport})
	channel := domain.Channel{
		SiteID:  "svt1",
		XmltvID: "SVT1.se",
		Lang:    "sv",
	}

	date := time.Date(2022, 8, 26, 0, 0, 0, 0, time.UTC)
	programs, err := p.Fetch(channel, date)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	first := programs[0]
	if first.Title != "Morgonstudion" {
		t.Fatalf("expected title 'Morgonstudion', got %q", first.Title)
	}

	if first.Image != "https://jrsy.tmsimg.com/assets/p14436175_i_h9_ad.jpg" {
		t.Fatalf("expected image, got %q", first.Image)
	}

	if first.Channel != "SVT1.se" {
		t.Fatalf("expected channel 'SVT1.se', got %q", first.Channel)
	}

	if first.Source != "tv24.se" {
		t.Fatalf("expected source 'tv24.se', got %q", first.Source)
	}

	last := programs[33]
	if last.Title != "Uppdrag granskning" {
		t.Fatalf("expected last title 'Uppdrag granskning', got %q", last.Title)
	}

	if last.Category != "Dokumentär / Samhällsfrågor" {
		t.Fatalf("expected category, got %q", last.Category)
	}

	if last.Season != 23 {
		t.Fatalf("expected season 23, got %d", last.Season)
	}

	if last.Episode != 5 {
		t.Fatalf("expected episode 5, got %d", last.Episode)
	}

	if last.SubTitle != "Där mördare går fria" {
		t.Fatalf("expected subtitle, got %q", last.SubTitle)
	}
}
