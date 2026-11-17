package epgshare

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/fetchcache"
)

type fakeClient struct {
	calls  atomic.Int32
	status int
	body   []byte
	delay  time.Duration
}

func (f *fakeClient) Do(req *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	time.Sleep(f.delay)
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(bytes.NewReader(f.body)),
	}, nil
}

func gzipped(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(raw)
	zw.Close()
	return buf.Bytes()
}

var testChannel = domain.Channel{
	ID:      "epgshare01.online-ALJAZEERA1#AlJazeera.English.net",
	XmltvID: "AlJazeera.English.net",
	SiteID:  "ALJAZEERA1#AlJazeera.English.net",
	Lang:    "en",
}

func TestFetchDownloadsOncePerTagConcurrently(t *testing.T) {
	fc := &fakeClient{status: 200, body: gzipped(t, "testdata/content.xml"), delay: 50 * time.Millisecond}
	p := &provider{client: fc, cache: newCache()}

	var wg sync.WaitGroup
	counts := make([]int, 30)
	for i := range counts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			progs, err := p.Fetch(testChannel, time.Date(2025, 2, 9, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Error(err)
				return
			}
			counts[i] = len(progs)
		}(i)
	}
	wg.Wait()

	if n := fc.calls.Load(); n != 1 {
		t.Fatalf("expected 1 download, got %d", n)
	}
	for i, c := range counts {
		if c != 40 {
			t.Fatalf("goroutine %d got %d programs, want 40", i, c)
		}
	}
}

func TestFetchMatchesParse(t *testing.T) {
	raw, err := os.ReadFile("testdata/content.xml")
	if err != nil {
		t.Fatal(err)
	}
	fc := &fakeClient{status: 200, body: gzipped(t, "testdata/content.xml")}
	p := &provider{client: fc, cache: newCache()}

	for d := 8; d <= 11; d++ {
		date := time.Date(2025, 2, d, 0, 0, 0, 0, time.UTC)
		want, err := Parse(raw, testChannel, date)
		if err != nil {
			t.Fatal(err)
		}
		got, err := p.Fetch(testChannel, date)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("day %d: got %d want %d", d, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("day %d program %d differs", d, i)
			}
		}
	}
}

func TestFetch404CachedPerTag(t *testing.T) {
	fc := &fakeClient{status: 404}
	p := &provider{client: fc, cache: newCache()}

	for i := 0; i < 10; i++ {
		_, err := p.Fetch(testChannel, time.Now())
		var se *fetchcache.StatusError
		if !errors.As(err, &se) || se.Code != 404 {
			t.Fatalf("expected 404 status error, got %v", err)
		}
		if err.Error() != "HTTP 404 from epgshare" {
			t.Fatalf("unexpected message %q", err.Error())
		}
	}
	if n := fc.calls.Load(); n != 1 {
		t.Fatalf("expected 1 request, got %d", n)
	}
}

func TestChannelsGroupedByTag(t *testing.T) {
	p := &provider{}
	chs, err := p.Channels()
	if err != nil {
		t.Fatal(err)
	}
	if len(chs) != 20706 {
		t.Fatalf("expected 20706 channels, got %d", len(chs))
	}
	seen := map[string]bool{}
	prev := ""
	for _, ch := range chs {
		tag := tagOf(ch)
		if tag != prev {
			if seen[tag] {
				t.Fatalf("tag %s appears in more than one run", tag)
			}
			seen[tag] = true
			prev = tag
		}
	}
}
