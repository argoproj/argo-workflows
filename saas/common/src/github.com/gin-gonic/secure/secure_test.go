package secure

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	testResponse = "bar"
)

func newServer(options Options) *gin.Engine {
	r := gin.Default()
	r.Use(Secure(options))
	r.GET("/foo", func(c *gin.Context) {
		c.String(200, testResponse)
	})
	return r
}

func TestNoConfig(t *testing.T) {
	s := newServer(Options{
	// Intentionally left blank.
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), "bar")
}

func TestNoAllowHosts(t *testing.T) {
	s := newServer(Options{
		AllowedHosts: []string{},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `bar`)
}

func TestGoodSingleAllowHosts(t *testing.T) {
	s := newServer(Options{
		AllowedHosts: []string{"www.example.com"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `bar`)
}

func TestBadSingleAllowHosts(t *testing.T) {
	s := newServer(Options{
		AllowedHosts: []string{"sub.example.com"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusInternalServerError)
}

func TestGoodMultipleAllowHosts(t *testing.T) {
	s := newServer(Options{
		AllowedHosts: []string{"www.example.com", "sub.example.com"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "sub.example.com"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), `bar`)
}

func TestBadMultipleAllowHosts(t *testing.T) {
	s := newServer(Options{
		AllowedHosts: []string{"www.example.com", "sub.example.com"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www3.example.com"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusInternalServerError)
}

func TestAllowHostsInDevMode(t *testing.T) {
	s := newServer(Options{
		AllowedHosts:  []string{"www.example.com", "sub.example.com"},
		IsDevelopment: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www3.example.com"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
}

func TestBadHostHandler(t *testing.T) {

	badHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "BadHost", http.StatusInternalServerError)
	})

	s := newServer(Options{
		AllowedHosts:   []string{"www.example.com", "sub.example.com"},
		BadHostHandler: badHandler,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www3.example.com"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusInternalServerError)

	// http.Error outputs a new line character with the response.
	expect(t, res.Body.String(), "BadHost\n")
}

func TestSSL(t *testing.T) {
	s := newServer(Options{
		SSLRedirect: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "https"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
}

func TestSSLInDevMode(t *testing.T) {
	s := newServer(Options{
		SSLRedirect:   true,
		IsDevelopment: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
}

func TestBasicSSL(t *testing.T) {
	s := newServer(Options{
		SSLRedirect: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusMovedPermanently)
	expect(t, res.Header().Get("Location"), "https://www.example.com/foo")
}

func TestBasicSSLWithHost(t *testing.T) {
	s := newServer(Options{
		SSLRedirect: true,
		SSLHost:     "secure.example.com",
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusMovedPermanently)
	expect(t, res.Header().Get("Location"), "https://secure.example.com/foo")
}

func TestBadProxySSL(t *testing.T) {
	s := newServer(Options{
		SSLRedirect: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"
	req.Header.Add("X-Forwarded-Proto", "https")

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusMovedPermanently)
	expect(t, res.Header().Get("Location"), "https://www.example.com/foo")
}

func TestCustomProxySSL(t *testing.T) {
	s := newServer(Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"
	req.Header.Add("X-Forwarded-Proto", "https")

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
}

func TestCustomProxySSLInDevMode(t *testing.T) {
	s := newServer(Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"
	req.Header.Add("X-Forwarded-Proto", "http")

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
}

func TestCustomProxyAndHostSSL(t *testing.T) {
	s := newServer(Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		SSLHost:         "secure.example.com",
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"
	req.Header.Add("X-Forwarded-Proto", "https")

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
}

func TestCustomBadProxyAndHostSSL(t *testing.T) {
	s := newServer(Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "superman"},
		SSLHost:         "secure.example.com",
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"
	req.Header.Add("X-Forwarded-Proto", "https")

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusMovedPermanently)
	expect(t, res.Header().Get("Location"), "https://secure.example.com/foo")
}

func TestCustomBadProxyAndHostSSLWithTempRedirect(t *testing.T) {
	s := newServer(Options{
		SSLRedirect:          true,
		SSLProxyHeaders:      map[string]string{"X-Forwarded-Proto": "superman"},
		SSLHost:              "secure.example.com",
		SSLTemporaryRedirect: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.Host = "www.example.com"
	req.URL.Scheme = "http"
	req.Header.Add("X-Forwarded-Proto", "https")

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusTemporaryRedirect)
	expect(t, res.Header().Get("Location"), "https://secure.example.com/foo")
}

func TestStsHeader(t *testing.T) {
	s := newServer(Options{
		STSSeconds: 315360000,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("Strict-Transport-Security"), "max-age=315360000")
}

func TestStsHeaderInDevMode(t *testing.T) {
	s := newServer(Options{
		STSSeconds:    315360000,
		IsDevelopment: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("Strict-Transport-Security"), "")
}

func TestStsHeaderWithSubdomain(t *testing.T) {
	s := newServer(Options{
		STSSeconds:           315360000,
		STSIncludeSubdomains: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("Strict-Transport-Security"), "max-age=315360000; includeSubdomains")
}

func TestFrameDeny(t *testing.T) {
	s := newServer(Options{
		FrameDeny: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("X-Frame-Options"), "DENY")
}

func TestCustomFrameValue(t *testing.T) {
	s := newServer(Options{
		CustomFrameOptionsValue: "SAMEORIGIN",
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("X-Frame-Options"), "SAMEORIGIN")
}

func TestCustomFrameValueWithDeny(t *testing.T) {
	s := newServer(Options{
		FrameDeny:               true,
		CustomFrameOptionsValue: "SAMEORIGIN",
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("X-Frame-Options"), "SAMEORIGIN")
}

func TestContentNosniff(t *testing.T) {
	s := newServer(Options{
		ContentTypeNosniff: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("X-Content-Type-Options"), "nosniff")
}

func TestXSSProtection(t *testing.T) {
	s := newServer(Options{
		BrowserXssFilter: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("X-XSS-Protection"), "1; mode=block")
}

func TestCsp(t *testing.T) {
	s := newServer(Options{
		ContentSecurityPolicy: "default-src 'self'",
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("Content-Security-Policy"), "default-src 'self'")
}

func TestInlineSecure(t *testing.T) {
	s := newServer(Options{
		FrameDeny: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)

	s.ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Header().Get("X-Frame-Options"), "DENY")
}

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected [%v] (type %v) - Got [%v] (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
