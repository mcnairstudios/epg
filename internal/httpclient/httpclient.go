package httpclient

import (
	"context"
	"net"
	"net/http"
	"sync"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
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
		}
	})
	return instance
}

func InsecureBrowser() *http.Client {
	insecureOnce.Do(func() {
		insecureInst = &http.Client{
			Transport: &browserTransport{insecureSkipVerify: true},
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

	if tlsConn.ConnectionState().NegotiatedProtocol == "h2" {
		h2Conn, err := (&http2.Transport{}).NewClientConn(tlsConn)
		if err != nil {
			conn.Close()
			return nil, err
		}
		return h2Conn.RoundTrip(req)
	}

	return (&http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return tlsConn, nil
		},
	}).RoundTrip(req)
}
