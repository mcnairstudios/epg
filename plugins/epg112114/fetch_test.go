package epg112114

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

type countingClient struct {
	calls atomic.Int32
	body  []byte
	delay time.Duration
}

func (c *countingClient) Do(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)
	time.Sleep(c.delay)
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(c.body))}, nil
}

func TestFetchDownloadsOncePerDocument(t *testing.T) {
	raw, err := os.ReadFile("testdata/content.xml")
	if err != nil {
		t.Fatal(err)
	}
	body := raw
	if false {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write(raw)
		zw.Close()
		body = buf.Bytes()
	}

	ch := domain.Channel{XmltvID: "X.id", SiteID: "BTV文艺", Lang: "zh"}
	date := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
	want, err := Parse(raw, ch, date)
	if err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 {
		t.Fatal("fixture should produce programmes for this channel and date")
	}

	cc := &countingClient{body: body, delay: 30 * time.Millisecond}
	p := &provider{client: cc, docs: newDocs()}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := p.Fetch(ch, date)
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Fetch result differs from Parse: %d vs %d programmes", len(got), len(want))
			}
		}()
	}
	wg.Wait()

	if n := cc.calls.Load(); n != 1 {
		t.Fatalf("expected 1 download, got %d", n)
	}
}
