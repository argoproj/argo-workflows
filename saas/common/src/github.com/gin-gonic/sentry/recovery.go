package sentry

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery(client *raven.Client, onlyCrashes bool) gin.HandlerFunc {

	return func(c *gin.Context) {
		defer func() {
			flags := map[string]string{
				"endpoint": c.Request.RequestURI,
			}
			if rval := recover(); rval != nil {
				debug.PrintStack()
				rvalStr := fmt.Sprint(rval)
				packet := raven.NewPacket(rvalStr,
					raven.NewException(errors.New(rvalStr), raven.NewStacktrace(2, 3, nil)),
					raven.NewHttp(c.Request))
				client.Capture(packet, flags)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
			if !onlyCrashes {
				for _, item := range c.Errors {
					packet := raven.NewPacket(item.Error(),
						&raven.Message{
							Message: item.Error(),
							Params:  []interface{}{item.Meta},
						},
						raven.NewHttp(c.Request))
					client.Capture(packet, flags)
				}
			}
		}()

		c.Next()
	}
}
