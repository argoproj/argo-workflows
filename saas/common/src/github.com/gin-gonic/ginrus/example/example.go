package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	r := gin.New()

	// Add a ginrus middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	//   - RFC3339 with UTC time format.
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))

	// Add similar middleware, but:
	//   - Only logs requests with errors, like an error log.
	//   - Logs to stderr instead of stdout.
	//   - Local time zone instead of UTC.
	logger := logrus.New()
	logger.Level = logrus.ErrorLevel
	logger.Out = os.Stderr
	r.Use(ginrus.Ginrus(logger, time.RFC3339, false))

	// Example ping request.
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
