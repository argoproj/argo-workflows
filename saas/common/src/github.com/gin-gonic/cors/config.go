package cors

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type settings struct {
	allowAllOrigins   bool
	allowedOriginFunc func(string) bool
	allowedOrigins    []string
	allowedMethods    []string
	allowedHeaders    []string
	exposedHeaders    []string
	normalHeaders     http.Header
	preflightHeaders  http.Header
}

func newSettings(c Config) *settings {
	if err := c.Validate(); err != nil {
		panic(err.Error())
	}
	return &settings{
		allowedOriginFunc: c.AllowOriginFunc,
		allowAllOrigins:   c.AllowAllOrigins,
		allowedOrigins:    c.AllowedOrigins,
		allowedMethods:    distinct(c.AllowedMethods),
		allowedHeaders:    distinct(c.AllowedHeaders),
		normalHeaders:     generateNormalHeaders(c),
		preflightHeaders:  generatePreflightHeaders(c),
	}
}

func (c *settings) validateOrigin(origin string) (string, bool) {
	if c.allowAllOrigins {
		return "*", true
	}
	if c.allowedOriginFunc != nil {
		return origin, c.allowedOriginFunc(origin)
	}
	for _, value := range c.allowedOrigins {
		if value == origin {
			return origin, true
		}
	}
	return "", false
}

func (c *settings) validateMethod(method string) bool {
	// TODO!!!
	return true
}

func (c *settings) validateHeader(header string) bool {
	// TODO!!!
	return true
}

func generateNormalHeaders(c Config) http.Header {
	headers := make(http.Header)
	if c.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if len(c.ExposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(c.ExposedHeaders, ", "))
	}
	return headers
}

func generatePreflightHeaders(c Config) http.Header {
	headers := make(http.Header)
	if c.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if len(c.AllowedMethods) > 0 {
		headers.Set("Access-Control-Allow-Methods", strings.Join(c.AllowedMethods, ", "))
	}
	if len(c.AllowedHeaders) > 0 {
		headers.Set("Access-Control-Allow-Headers", strings.Join(c.AllowedHeaders, ", "))
	}
	if c.MaxAge > time.Duration(0) {
		headers.Set("Access-Control-Max-Age", strconv.FormatInt(int64(c.MaxAge/time.Second), 10))
	}
	return headers
}

func distinct(s []string) []string {
	m := map[string]bool{}
	for _, v := range s {
		if _, seen := m[v]; !seen {
			s[len(m)] = v
			m[v] = true
		}
	}
	return s[:len(m)]
}

func parse(content string) []string {
	if len(content) == 0 {
		return nil
	}
	parts := strings.Split(content, ",")
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
