package static

import (
	"net/http"
	"strings"
)

func ServerFiles(w http.ResponseWriter, r *http.Request) {

	// this hack allows us to server the routes (e.g. /workflows) with the index file
	if !strings.Contains(r.URL.Path, ".") {
		r.URL.Path = "index.html"
	}
	// in my IDE (IntelliJ) the next line is red for some reason - but this is fine
	ServeHTTP(w, r)
}
