package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/contrib/cache"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	store := cache.NewInMemoryStore(time.Second)
	// Cached Page
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	r.GET("/cache_ping", cache.CachePage(store, time.Minute, func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	}))

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
