package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBadConfig(t *testing.T) {
	assert.Panics(t, func() { New(Options{}) })
	assert.Panics(t, func() {
		New(Options{
			AllowAllOrigins: true,
			AllowedOrigins:  []string{"http://google.com"},
		})
	})
	assert.Panics(t, func() {
		New(Options{
			AllowAllOrigins: true,
			AllowOriginFunc: func(origin string) bool { return false },
		})
	})
	assert.Panics(t, func() {
		New(Options{
			AllowedOrigins:  []string{"http://google.com"},
			AllowOriginFunc: func(origin string) bool { return false },
		})
	})
	assert.Panics(t, func() {
		New(Options{
			AllowedOrigins: []string{"google.com"},
		})
	})
}

func TestDeny0(t *testing.T) {
	called := false

	router := gin.Default()
	router.Use(New(Options{
		AllowedOrigins: []string{"http://example.com"},
	}))
	router.GET("/", func(c *gin.Context) {
		called = true
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	router.ServeHTTP(w, req)

	assert.True(t, called)
	assert.NotContains(t, w.Header(), "Access-Control")
}

func TestDenyAbortOnError(t *testing.T) {
	called := false

	router := gin.Default()
	router.Use(New(Options{
		AbortOnError:   true,
		AllowedOrigins: []string{"http://example.com"},
	}))
	router.GET("/", func(c *gin.Context) {
		called = true
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	router.ServeHTTP(w, req)

	assert.False(t, called)
	assert.NotContains(t, w.Header(), "Access-Control")
}

func TestDeny2(t *testing.T) {

}
func TestDeny3(t *testing.T) {

}

func TestPasses0(t *testing.T) {

}

func TestPasses1(t *testing.T) {

}

func TestPasses2(t *testing.T) {

}
