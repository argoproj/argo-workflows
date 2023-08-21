package static

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeFile(t *testing.T) {
	t.Run("should handle base href for static file", func(t *testing.T) {
		filesServer := NewFilesServer("/argotest/", false, "", "")
		filesServer.Hash = func(s string) string {
			resources := map[string]string{
				"main.js": "main.js",
			}
			return resources[s]
		}

		req := httptest.NewRequest(http.MethodGet, "/argotest/main.js", nil)
		w := httptest.NewRecorder()

		filesServer.ServerFiles(w, req)

		if req.URL.Path != "main.js" {
			t.Errorf("should handle base href, expect %v, got %v", "main.js", req.URL.Path)
		}
	})
}
