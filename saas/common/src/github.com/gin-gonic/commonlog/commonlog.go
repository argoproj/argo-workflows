package commonlog

import (
	"bytes"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Instances a Logger middleware that will write the logs to gin.DefaultWriter
// By default gin.DefaultWriter = os.Stdout
func New() gin.HandlerFunc {
	return NewWithWriter(gin.DefaultWriter)
}

// Instance a Logger middleware with the specified writter buffer.
// Example: os.Stdout, a file opened in write mode, a socket...
func NewWithWriter(out io.Writer) gin.HandlerFunc {
	pool := &sync.Pool{
		New: func() interface{} {
			buf := new(bytes.Buffer)
			return buf
		},
	}
	return func(c *gin.Context) {
		// Process request
		c.Next()

		//127.0.0.1 user-identifier frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326
		w := pool.Get().(*bytes.Buffer)
		w.Reset()
		w.WriteString(c.ClientIP())
		w.WriteString(" - - ")
		w.WriteString(time.Now().Format("[02/Jan/2006:15:04:05 -0700] "))
		w.WriteString("\"")
		w.WriteString(c.Request.Method)
		w.WriteString(" ")
		w.WriteString(c.Request.URL.Path)
		w.WriteString(" ")
		w.WriteString(c.Request.Proto)
		w.WriteString("\" ")
		w.WriteString(strconv.Itoa(c.Writer.Status()))
		w.WriteString(" ")
		w.WriteString(strconv.Itoa(c.Writer.Size()))
		w.WriteString("\n")

		w.WriteTo(out)
		pool.Put(w)
	}
}
