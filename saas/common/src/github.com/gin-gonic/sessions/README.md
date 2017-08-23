# sessions
Gin middleware for session management with multi-backend support (currently cookie, Redis). 

## Examples

#### cookie-based

```go
package main

import (
  "github.com/gin-gonic/contrib/sessions"
  "github.com/gin-gonic/gin"
)

func main() {
  r := gin.Default()
  store := sessions.NewCookieStore([]byte("secret"))
  r.Use(sessions.Sessions("mysession", store))

  r.GET("/incr", func(c *gin.Context) {
    session := sessions.Default(c)  
    var count int
    v := session.Get("count")
    if v == nil {
      count = 0
    } else {
      count = v.(int)
      count += 1
    }
    session.Set("count", count)
    session.Save()
    c.JSON(200, gin.H{"count": count})
  })
  r.Run(":8000")
}
```

#### Redis

```go
package main

import (
  "github.com/gin-gonic/contrib/sessions"
  "github.com/gin-gonic/gin"
)

func main() {
  r := gin.Default()
  store, _ := sessions.NewRedisStore(10, "tcp", "localhost:6379", "", []byte("secret"))
  r.Use(sessions.Sessions("session", store))

  r.GET("/incr", func(c *gin.Context) {
    session := sessions.Default(c)
    var count int
    v := session.Get("count")
    if v == nil {
      count = 0
    } else {
      count = v.(int)
      count += 1
    }
    session.Set("count", count)
    session.Save()
    c.JSON(200, gin.H{"count": count})
  })
  r.Run(":8000")
}
```