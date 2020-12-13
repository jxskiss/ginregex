package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jxskiss/ginregex"
)

var db = make(map[string]string)

func setupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// Get user value
	r.GET("/user/:name", func(c *gin.Context) {
		user := c.Params.ByName("name")
		value, ok := db[user]
		if ok {
			c.JSON(http.StatusOK, gin.H{"user": user, "value": value})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": user, "status": "no value"})
		}
	})

	// Authorized group (uses gin.BasicAuth() middleware)
	// Same than:
	// authorized := r.Group("/")
	// authorized.Use(gin.BasicAuth(gin.Credentials{
	//	  "foo":  "bar",
	//	  "manu": "123",
	//}))
	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		"foo":  "bar", // user:foo password:bar
		"manu": "123", // user:manu password:123
	}))

	authorized.POST("admin", func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(string)

		// Parse JSON
		var json struct {
			Value string `json:"value" binding:"required"`
		}

		if c.Bind(&json) == nil {
			db[user] = json.Value
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	// Dispatch
	r.Any("/users/*any", ginregex.Dispatch(
		ginregex.NewMatcher("GET", `^/users/settings/$`, echoHandler),
		ginregex.NewMatcher("POST", `^/users/settings/$`, echoHandler),
		ginregex.NewMatcher("GET", `^/users/(?P<user_id>\d+)/$`, paramHandler),
	))

	// Regular expression endpoints
	regexhook := func(c *gin.Context, pattern string) {
		// do whatever you need to prepare/hack the request
	}
	regexRouter := ginregex.New(r, regexhook)
	regexRouter.GET("^/.*$", func(c *gin.Context) {
		c.String(http.StatusOK, c.Request.URL.String())
	})

	return r
}

func echoHandler(c *gin.Context) {
	c.String(http.StatusOK, fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path))
}

func paramHandler(c *gin.Context) {
	var params []string
	for _, param := range c.Params {
		params = append(params, fmt.Sprintf("%s:%s", param.Key, param.Value))
	}
	c.String(http.StatusOK, strings.Join(params, " "))
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
