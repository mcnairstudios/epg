package httpclient

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"

	"github.com/gavinmcnair/epg/internal/bandwidth"
)

var (
	once         sync.Once
	instance     *http.Client
	insecureOnce sync.Once
	insecureInst *http.Client
)

func Browser() *http.Client {
	once.Do(func() {
		instance = &http.Client{
			Transport: &browserTransport{},
			Timeout:   30 * time.Second,
		}
	})
	return instance
}

func InsecureBrowser() *http.Client {
	insecureOnce.Do(func() {
		insecureInst = &http.Client{
			Transport: &browserTransport{insecureSkipVerify: true},
			Timeout:   30 * time.Second,
		}
	})
	return insecureInst
}

type browserTransport struct {
	insecureSkipVerify bool
}

func (t *browserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	port := req.URL.Port()
	if port == "" {
		port = "443"
	}

	conn, err := net.Dial("tcp", net.JoinHostPort(req.URL.Hostname(), port))
	if err != nil {
		return nil, err
	}

	tlsConn := utls.UClient(conn, &utls.Config{
		ServerName:         req.URL.Hostname(),
		InsecureSkipVerify: t.insecureSkipVerify,
	}, utls.HelloChrome_Auto)
	if err := tlsConn.HandshakeContext(req.Context()); err != nil {
		conn.Close()
		return nil, err
	}

	var resp *http.Response
	if tlsConn.ConnectionState().NegotiatedProtocol == "h2" {
		h2Conn, err := (&http2.Transport{}).NewClientConn(tlsConn)
		if err != nil {
			conn.Close()
			return nil, err
		}
		resp, err = h2Conn.RoundTrip(req)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		resp, err = (&http.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return tlsConn, nil
			},
		}).RoundTrip(req)
		if err != nil {
			return nil, err
		}
	}

	resp.Body = &countingReadCloser{ReadCloser: resp.Body}
	return resp, nil
}

type countingReadCloser struct {
	io.ReadCloser
}

func (c *countingReadCloser) Read(p []byte) (n int, err error) {
	n, err = c.ReadCloser.Read(p)
	if n > 0 {
		bandwidth.Global.Add(int64(n))
	}
	return
}
