package e2e

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

var httpClient = &http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

var http2Client = &http.Client{
	Transport: &http2.Transport{
		AllowHTTP: true,
		// Skip TLS dial
		DialTLSContext: func(ctx context.Context, netw, addr string, cfg *tls.Config) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, netw, addr)
		},
	},
	CheckRedirect: httpClient.CheckRedirect,
}
