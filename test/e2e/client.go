package e2e

import (
	"crypto/tls"
	"net/http"
)

var httpClient = &http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// http2Client speaks HTTP/2 over cleartext (h2c, prior knowledge) to the plain-HTTP argo-server.
var http2Client = &http.Client{
	Transport:     &http.Transport{Protocols: unencryptedHTTP2Only()},
	CheckRedirect: httpClient.CheckRedirect,
}

func unencryptedHTTP2Only() *http.Protocols {
	p := &http.Protocols{}
	p.SetUnencryptedHTTP2(true)
	return p
}
