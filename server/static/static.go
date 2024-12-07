package static

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/ui"
)

type FilesServer struct {
	baseHRef        string
	hsts            bool
	xframeOpts      string
	corsAllowOrigin string
	staticAssets    embed.FS
}

var baseHRefRegex = regexp.MustCompile(`<base href="(.*?)">`)

func NewFilesServer(baseHRef string, hsts bool, xframeOpts string, corsAllowOrigin string, staticAssets embed.FS) *FilesServer {
	return &FilesServer{baseHRef, hsts, xframeOpts, corsAllowOrigin, staticAssets}
}

func (s *FilesServer) ServerFiles(w http.ResponseWriter, r *http.Request) {

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

	if r.URL.Path == "/" || !s.uiAssetExists(r.URL.Path) {
		data, err := s.getIndexData()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		modTime, err := time.Parse(argo.GetVersion().BuildDate, time.RFC3339)
		if err != nil {
			modTime = time.Now()
		}
		http.ServeContent(w, r, "index.html", modTime, bytes.NewReader(data))
	} else {
		staticFS, _ := fs.Sub(s.staticAssets, ui.EMBED_PATH)
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
	}
}

func (s *FilesServer) getIndexData() ([]byte, error) {
	data, err := s.staticAssets.ReadFile(ui.EMBED_PATH + "/index.html")
	if err != nil {
		return data, err
	}
	if s.baseHRef != "/" && s.baseHRef != "" {
		data = []byte(replaceBaseHRef(string(data), fmt.Sprintf(`<base href="/%s/">`, strings.Trim(s.baseHRef, "/"))))
	}

	return data, nil
}

func (s *FilesServer) uiAssetExists(filename string) bool {
	f, err := s.staticAssets.Open(ui.EMBED_PATH + filename)
	if err != nil {
		return false
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func replaceBaseHRef(data string, replaceWith string) string {
	return baseHRefRegex.ReplaceAllString(data, replaceWith)
}
