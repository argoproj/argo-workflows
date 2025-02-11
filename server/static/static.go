package static

import (
	"fmt"
	"net/http"
	"strings"
)

type FilesServer struct {
	baseHRef        string
	hsts            bool
	xframeOpts      string
	corsAllowOrigin string
}

func NewFilesServer(baseHRef string, hsts bool, xframeOpts string, corsAllowOrigin string) *FilesServer {
	return &FilesServer{baseHRef, hsts, xframeOpts, corsAllowOrigin}
}

func (s *FilesServer) ServerFiles(w http.ResponseWriter, r *http.Request) {
	// If there is no stored static file, we'll redirect to the js app
	if Hash(strings.TrimLeft(r.URL.Path, "/")) == "" {
		r.URL.Path = "index.html"
	}

	if r.URL.Path == "index.html" {
		// hack to prevent ServerHTTP from giving us gzipped content which we can do our search-and-replace on
		r.Header.Del("Accept-Encoding")
		w = &responseRewriter{ResponseWriter: w, old: []byte(`<base href="/">`), new: []byte(fmt.Sprintf(`<base href="%s">`, s.baseHRef))}
	}

	if s.xframeOpts != "" {
		w.Header().Set("X-Frame-Options", s.xframeOpts)
	}

	if s.corsAllowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", s.corsAllowOrigin)
		if r.Method == http.MethodOptions { // Set CORS headers for preflight request
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	// `data:` is need for Monaco editors wiggly red lines
	w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline'; img-src 'self' data:")
	if s.hsts {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}

	// in my IDE (IntelliJ) the next line is red for some reason - but this is fine
	ServeHTTP(w, r)
}
