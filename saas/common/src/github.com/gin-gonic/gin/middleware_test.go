// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"errors"
	"strings"

	"testing"

	"github.com/manucorporat/sse"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareGeneralCase(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *Context) {
		signature += "C"
	})
	router.GET("/", func(c *Context) {
		signature += "D"
	})
	router.NoRoute(func(c *Context) {
		signature += " X "
	})
	router.NoMethod(func(c *Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, signature, "ACDB")
}

func TestMiddlewareNoRoute(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *Context) {
		signature += "C"
		c.Next()
		c.Next()
		c.Next()
		c.Next()
		signature += "D"
	})
	router.NoRoute(func(c *Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *Context) {
		signature += "G"
		c.Next()
		signature += "H"
	})
	router.NoMethod(func(c *Context) {
		signature += " X "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, w.Code, 404)
	assert.Equal(t, signature, "ACEGHFDB")
}

func TestMiddlewareNoMethodEnabled(t *testing.T) {
	signature := ""
	router := New()
	router.HandleMethodNotAllowed = true
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *Context) {
		signature += "C"
		c.Next()
		signature += "D"
	})
	router.NoMethod(func(c *Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *Context) {
		signature += "G"
		c.Next()
		signature += "H"
	})
	router.NoRoute(func(c *Context) {
		signature += " X "
	})
	router.POST("/", func(c *Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, w.Code, 405)
	assert.Equal(t, signature, "ACEGHFDB")
}

func TestMiddlewareNoMethodDisabled(t *testing.T) {
	signature := ""
	router := New()
	router.HandleMethodNotAllowed = false
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *Context) {
		signature += "C"
		c.Next()
		signature += "D"
	})
	router.NoMethod(func(c *Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *Context) {
		signature += "G"
		c.Next()
		signature += "H"
	})
	router.NoRoute(func(c *Context) {
		signature += " X "
	})
	router.POST("/", func(c *Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, w.Code, 404)
	assert.Equal(t, signature, "AC X DB")
}

func TestMiddlewareAbort(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
	})
	router.Use(func(c *Context) {
		signature += "C"
		c.AbortWithStatus(401)
		c.Next()
		signature += "D"
	})
	router.GET("/", func(c *Context) {
		signature += " X "
		c.Next()
		signature += " XX "
	})

	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, w.Code, 401)
	assert.Equal(t, signature, "ACD")
}

func TestMiddlewareAbortHandlersChainAndNext(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		c.AbortWithStatus(410)
		signature += "B"

	})
	router.GET("/", func(c *Context) {
		signature += "C"
		c.Next()
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, w.Code, 410)
	assert.Equal(t, signature, "ACB")
}

// TestFailHandlersChain - ensure that Fail interrupt used middleware in fifo order as
// as well as Abort
func TestMiddlewareFailHandlersChain(t *testing.T) {
	// SETUP
	signature := ""
	router := New()
	router.Use(func(context *Context) {
		signature += "A"
		context.AbortWithError(500, errors.New("foo"))
	})
	router.Use(func(context *Context) {
		signature += "B"
		context.Next()
		signature += "C"
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, w.Code, 500)
	assert.Equal(t, signature, "A")
}

func TestMiddlewareWrite(t *testing.T) {
	router := New()
	router.Use(func(c *Context) {
		c.String(400, "hola\n")
	})
	router.Use(func(c *Context) {
		c.XML(400, H{"foo": "bar"})
	})
	router.Use(func(c *Context) {
		c.JSON(400, H{"foo": "bar"})
	})
	router.GET("/", func(c *Context) {
		c.JSON(400, H{"foo": "bar"})
	}, func(c *Context) {
		c.Render(400, sse.Event{
			Event: "test",
			Data:  "message",
		})
	})

	w := performRequest(router, "GET", "/")

	assert.Equal(t, w.Code, 400)
	assert.Equal(t, strings.Replace(w.Body.String(), " ", "", -1), strings.Replace("hola\n<map><foo>bar</foo></map>{\"foo\":\"bar\"}\n{\"foo\":\"bar\"}\nevent:test\ndata:message\n\n", " ", "", -1))
}
