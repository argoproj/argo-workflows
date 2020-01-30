package static

import (
	"fmt"
	"net/http"
	"strings"
)

type FilesServer struct {
	baseHRef string
}

func NewFilesServer(baseHRef string) *FilesServer {
	return &FilesServer{baseHRef}
}

func (s *FilesServer) ServerFiles(w http.ResponseWriter, r *http.Request) {

	// this hack allows us to server the routes (e.g. /workflows) with the index file
	if !strings.Contains(r.URL.Path, ".") {
		r.URL.Path = "index.html"
	}

	if r.URL.Path == "index.html" {
		// hack to prevent ServerHTTP from giving us gzipped content which we can do our search-and-replace on
		r.Header.Del("Accept-Encoding")
		w = &responseRewriter{ResponseWriter: w, old: []byte(`<base href="/">`), new: []byte(fmt.Sprintf(`<base href="%s">`, s.baseHRef))}
	}

	// in my IDE (IntelliJ) the next line is red for some reason - but this is fine
	ServeHTTP(w, r)
}
