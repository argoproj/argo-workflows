// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

//TODO
// func (engine *Engine) LoadHTMLGlob(pattern string) {
// func (engine *Engine) LoadHTMLFiles(files ...string) {
// func (engine *Engine) RunTLS(addr string, cert string, key string) error {

func init() {
	SetMode(TestMode)
}

func TestCreateEngine(t *testing.T) {
	router := New()
	assert.Equal(t, "/", router.basePath)
	assert.Equal(t, router.engine, router)
	assert.Empty(t, router.Handlers)
}

// func TestLoadHTMLDebugMode(t *testing.T) {
// 	router := New()
// 	SetMode(DebugMode)
// 	router.LoadHTMLGlob("*.testtmpl")
// 	r := router.HTMLRender.(render.HTMLDebug)
// 	assert.Empty(t, r.Files)
// 	assert.Equal(t, r.Glob, "*.testtmpl")
//
// 	router.LoadHTMLFiles("index.html.testtmpl", "login.html.testtmpl")
// 	r = router.HTMLRender.(render.HTMLDebug)
// 	assert.Empty(t, r.Glob)
// 	assert.Equal(t, r.Files, []string{"index.html", "login.html"})
// 	SetMode(TestMode)
// }

func TestLoadHTMLReleaseMode(t *testing.T) {

}

func TestAddRoute(t *testing.T) {
	router := New()
	router.addRoute("GET", "/", HandlersChain{func(_ *Context) {}})

	assert.Len(t, router.trees, 1)
	assert.NotNil(t, router.trees.get("GET"))
	assert.Nil(t, router.trees.get("POST"))

	router.addRoute("POST", "/", HandlersChain{func(_ *Context) {}})

	assert.Len(t, router.trees, 2)
	assert.NotNil(t, router.trees.get("GET"))
	assert.NotNil(t, router.trees.get("POST"))

	router.addRoute("POST", "/post", HandlersChain{func(_ *Context) {}})
	assert.Len(t, router.trees, 2)
}

func TestAddRouteFails(t *testing.T) {
	router := New()
	assert.Panics(t, func() { router.addRoute("", "/", HandlersChain{func(_ *Context) {}}) })
	assert.Panics(t, func() { router.addRoute("GET", "a", HandlersChain{func(_ *Context) {}}) })
	assert.Panics(t, func() { router.addRoute("GET", "/", HandlersChain{}) })

	router.addRoute("POST", "/post", HandlersChain{func(_ *Context) {}})
	assert.Panics(t, func() {
		router.addRoute("POST", "/post", HandlersChain{func(_ *Context) {}})
	})
}

func TestCreateDefaultRouter(t *testing.T) {
	router := Default()
	assert.Len(t, router.Handlers, 2)
}

func TestNoRouteWithoutGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}

	router := New()

	router.NoRoute(middleware0)
	assert.Nil(t, router.Handlers)
	assert.Len(t, router.noRoute, 1)
	assert.Len(t, router.allNoRoute, 1)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware0)

	router.NoRoute(middleware1, middleware0)
	assert.Len(t, router.noRoute, 2)
	assert.Len(t, router.allNoRoute, 2)
	compareFunc(t, router.noRoute[0], middleware1)
	compareFunc(t, router.allNoRoute[0], middleware1)
	compareFunc(t, router.noRoute[1], middleware0)
	compareFunc(t, router.allNoRoute[1], middleware0)
}

func TestNoRouteWithGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}
	var middleware2 HandlerFunc = func(c *Context) {}

	router := New()
	router.Use(middleware2)

	router.NoRoute(middleware0)
	assert.Len(t, router.allNoRoute, 2)
	assert.Len(t, router.Handlers, 1)
	assert.Len(t, router.noRoute, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware2)
	compareFunc(t, router.allNoRoute[1], middleware0)

	router.Use(middleware1)
	assert.Len(t, router.allNoRoute, 3)
	assert.Len(t, router.Handlers, 2)
	assert.Len(t, router.noRoute, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.Handlers[1], middleware1)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware2)
	compareFunc(t, router.allNoRoute[1], middleware1)
	compareFunc(t, router.allNoRoute[2], middleware0)
}

func TestNoMethodWithoutGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}

	router := New()

	router.NoMethod(middleware0)
	assert.Empty(t, router.Handlers)
	assert.Len(t, router.noMethod, 1)
	assert.Len(t, router.allNoMethod, 1)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware0)

	router.NoMethod(middleware1, middleware0)
	assert.Len(t, router.noMethod, 2)
	assert.Len(t, router.allNoMethod, 2)
	compareFunc(t, router.noMethod[0], middleware1)
	compareFunc(t, router.allNoMethod[0], middleware1)
	compareFunc(t, router.noMethod[1], middleware0)
	compareFunc(t, router.allNoMethod[1], middleware0)
}

func TestRebuild404Handlers(t *testing.T) {

}

func TestNoMethodWithGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}
	var middleware2 HandlerFunc = func(c *Context) {}

	router := New()
	router.Use(middleware2)

	router.NoMethod(middleware0)
	assert.Len(t, router.allNoMethod, 2)
	assert.Len(t, router.Handlers, 1)
	assert.Len(t, router.noMethod, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware2)
	compareFunc(t, router.allNoMethod[1], middleware0)

	router.Use(middleware1)
	assert.Len(t, router.allNoMethod, 3)
	assert.Len(t, router.Handlers, 2)
	assert.Len(t, router.noMethod, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.Handlers[1], middleware1)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware2)
	compareFunc(t, router.allNoMethod[1], middleware1)
	compareFunc(t, router.allNoMethod[2], middleware0)
}

func compareFunc(t *testing.T, a, b interface{}) {
	sf1 := reflect.ValueOf(a)
	sf2 := reflect.ValueOf(b)
	if sf1.Pointer() != sf2.Pointer() {
		t.Error("different functions")
	}
}

func TestListOfRoutes(t *testing.T) {
	router := New()
	router.GET("/favicon.ico", handlerTest1)
	router.GET("/", handlerTest1)
	group := router.Group("/users")
	{
		group.GET("/", handlerTest2)
		group.GET("/:id", handlerTest1)
		group.POST("/:id", handlerTest2)
	}
	router.Static("/static", ".")

	list := router.Routes()

	assert.Len(t, list, 7)
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/favicon.ico",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/users/",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest2$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/users/:id",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "POST",
		Path:    "/users/:id",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest2$",
	})
}

func assertRoutePresent(t *testing.T, gotRoutes RoutesInfo, wantRoute RouteInfo) {
	for _, gotRoute := range gotRoutes {
		if gotRoute.Path == wantRoute.Path && gotRoute.Method == wantRoute.Method {
			assert.Regexp(t, wantRoute.Handler, gotRoute.Handler)
			return
		}
	}
	t.Errorf("route not found: %v", wantRoute)
}

func handlerTest1(c *Context) {}
func handlerTest2(c *Context) {}
