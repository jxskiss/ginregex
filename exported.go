package ginregex

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"regexp"
)

var regexRouters = make(map[*gin.Engine]*RegexRouter)

// New creates a regular expression router with the given gin.Engine.
//
// To function properly, only one RegexRouter can be created for each
// gin.Engine pointer, if called multiple times using same gin.Engine
// pointer, it returns the RegexRouter previously created.
func New(engine *gin.Engine, hook Hook) *RegexRouter {
	router, ok := regexRouters[engine]
	if !ok {
		initOffset()
		router = &RegexRouter{
			engine: engine,
			hook:   hook,
			table:  make(map[string][]*regexHandler),
		}
		engine.Handlers = append(gin.HandlersChain{router.route()}, engine.Handlers...)
		regexRouters[engine] = router
	}
	return router
}

type Hook func(c *gin.Context, params gin.Params, httpMethod, regexPattern string)

// RegexRouter is a regular expression router to be used with gin http framework.
// It uses unsafe magic to patch gin.Engine and gin.Context dynamically.
//
// If named capturing with the (?P<name>...) syntax presents in registered
// routes, the captured values will be filled into gin.Context.Params, so
// you can access them like normal gin path parameters in your handlers.
type RegexRouter struct {
	engine   *gin.Engine
	hook     Hook
	handlers gin.HandlersChain
	table    map[string][]*regexHandler
}

// Use adds middleware to the router.
func (r *RegexRouter) Use(middleware ...gin.HandlerFunc) *RegexRouter {
	r.handlers = append(r.handlers, middleware...)
	return r
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware
// that can and should be shared among different routes.
// See the example code in GitHub.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *RegexRouter) Handle(httpMethod, regexPattern string, handlers ...gin.HandlerFunc) *RegexRouter {
	// normalize regexp pattern
	methodHandlers := r.table[httpMethod]
	for _, h := range methodHandlers {
		if h.pattern == regexPattern {
			panic(fmt.Sprintf("register duplicate regex route: %s %s", httpMethod, regexPattern))
		}
	}
	re := regexp.MustCompile(regexPattern)
	r.table[httpMethod] = append(methodHandlers, &regexHandler{
		pattern:  regexPattern,
		regex:    re,
		handlers: handlers,
	})
	return r
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE.
func (r *RegexRouter) Any(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "HEAD", "OPTIONS", "DELETE", "CONNECT", "TRACE"} {
		r.Handle(method, regexPath, handlers...)
	}
	return r
}

// GET is a shortcut for router.Handle("GET", path, handle).
func (r *RegexRouter) GET(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	return r.Handle("GET", regexPath, handlers...)
}

// POST is a shortcut for router.Handle("POST", path, handle).
func (r *RegexRouter) POST(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	return r.Handle("POST", regexPath, handlers...)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle).
func (r *RegexRouter) DELETE(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	return r.Handle("DELETE", regexPath, handlers...)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle).
func (r *RegexRouter) PATCH(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	return r.Handle("PATCH", regexPath, handlers...)
}

// PUT is a shortcut for router.Handle("PUT", path, handle).
func (r *RegexRouter) PUT(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	return r.Handle("PUT", regexPath, handlers...)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle).
func (r *RegexRouter) OPTIONS(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	return r.Handle("OPTIONS", regexPath, handlers...)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle).
func (r *RegexRouter) HEAD(regexPath string, handlers ...gin.HandlerFunc) *RegexRouter {
	return r.Handle("HEAD", regexPath, handlers...)
}
