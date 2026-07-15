package e2e

import (
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
		DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(netw, addr)
		},
	},
	CheckRedirect: httpClient.CheckRedirect,
}
