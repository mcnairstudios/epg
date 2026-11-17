package httpcache

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fake struct {
	calls  atomic.Int32
	status int
	body   string
	delay  time.Duration
}

func (f *fake) Do(req *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	time.Sleep(f.delay)
	return &http.Response{
		StatusCode: f.status,
		Status:     http.StatusText(f.status),
		Header:     http.Header{"X-Test": {"1"}},
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

func get(t *testing.T, d Doer, url string, hdr ...string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	resp, err := d.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func newDoer(f *fake, opts ...Option) Doer {
	c := &client{next: f, cache: newCache(1 << 20)}
	for _, o := range opts {
		o(c)
	}
	return c
}

func TestConcurrentIdenticalGetsDownloadOnce(t *testing.T) {
	f := &fake{status: 200, body: "doc", delay: 40 * time.Millisecond}
	d := newDoer(f)
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if code, b := get(t, d, "http://x/a"); code != 200 || b != "doc" {
				t.Errorf("got %d %q", code, b)
			}
		}()
	}
	wg.Wait()
	if n := f.calls.Load(); n != 1 {
		t.Fatalf("expected 1 call, got %d", n)
	}
}

func TestDifferentURLOrHeaderNotShared(t *testing.T) {
	f := &fake{status: 200, body: "doc"}
	d := newDoer(f)
	get(t, d, "http://x/a")
	get(t, d, "http://x/b")
	get(t, d, "http://x/a", "Cookie", "c=1")
	get(t, d, "http://x/a", "Cookie", "c=2")
	if n := f.calls.Load(); n != 4 {
		t.Fatalf("expected 4 calls, got %d", n)
	}
	get(t, d, "http://x/a", "Cookie", "c=2")
	if n := f.calls.Load(); n != 4 {
		t.Fatalf("expected repeat to be cached, got %d calls", n)
	}
}

func TestResponseReadableRepeatedlyAndHeadersKept(t *testing.T) {
	f := &fake{status: 200, body: "payload"}
	d := newDoer(f)
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://x/a", nil)
		resp, err := d.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		if string(b) != "payload" || resp.Header.Get("X-Test") != "1" || resp.ContentLength != 7 {
			t.Fatalf("bad response: %q %v %d", b, resp.Header, resp.ContentLength)
		}
	}
}

func TestNotFoundCachedServerErrorNot(t *testing.T) {
	nf := &fake{status: 404}
	d := newDoer(nf)
	for i := 0; i < 5; i++ {
		if code, _ := get(t, d, "http://x/missing"); code != 404 {
			t.Fatalf("code %d", code)
		}
	}
	if n := nf.calls.Load(); n != 1 {
		t.Fatalf("404 should be cached, got %d calls", n)
	}

	se := &fake{status: 503}
	d = newDoer(se)
	for i := 0; i < 3; i++ {
		get(t, d, "http://x/boom")
	}
	if n := se.calls.Load(); n != 3 {
		t.Fatalf("503 must not be retained, got %d calls", n)
	}
}

func TestPOSTOptInKeyedByBody(t *testing.T) {
	f := &fake{status: 200, body: "r"}
	post := func(d Doer, body string) {
		req, _ := http.NewRequest(http.MethodPost, "http://x/q", strings.NewReader(body))
		resp, err := d.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	d := newDoer(f)
	post(d, "a")
	post(d, "a")
	if n := f.calls.Load(); n != 2 {
		t.Fatalf("POST must not be cached by default, got %d calls", n)
	}

	f2 := &fake{status: 200, body: "r"}
	d = newDoer(f2, WithPOST())
	post(d, "a")
	post(d, "a")
	post(d, "b")
	if n := f2.calls.Load(); n != 2 {
		t.Fatalf("expected 2 calls (a, b), got %d", n)
	}
}

func TestPOSTBodyReachesUpstream(t *testing.T) {
	var got string
	next := doerFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		got = string(b)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	d := &client{next: next, cache: newCache(1 << 20), post: true}
	req, _ := http.NewRequest(http.MethodPost, "http://x/q", strings.NewReader("form=1"))
	if _, err := d.Do(req); err != nil {
		t.Fatal(err)
	}
	if got != "form=1" {
		t.Fatalf("upstream saw body %q", got)
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }
