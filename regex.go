package ginregex

import (
	"github.com/gin-gonic/gin"
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
	"unsafe"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func longestCommonPrefix(a, b string) string {
	i := 0
	max := min(len(a), len(b))
	for i < max && a[i] == b[i] && a[i] != '\\' {
		i++
	}
	return a[:i]
}

func newRegexHandler(pattern string, handlers gin.HandlersChain) *regexHandler {
	normalized := pattern
	if !strings.HasPrefix(normalized, "^") {
		normalized = "^" + pattern
	}
	regex := regexp.MustCompile(normalized)
	trimmed := strings.TrimPrefix(normalized, "^")
	quoted := regexp.QuoteMeta(trimmed)
	prefix := longestCommonPrefix(trimmed, quoted)
	return &regexHandler{
		pattern:  pattern,
		prefix:   prefix,
		regex:    regex,
		handlers: handlers,
	}
}

type regexHandler struct {
	pattern  string
	prefix   string
	regex    *regexp.Regexp
	handlers gin.HandlersChain

	combinedHandlers unsafe.Pointer
}

func (r *regexHandler) match(path string, params gin.Params) (gin.Params, bool) {
	if !strings.HasPrefix(path, r.prefix) {
		return nil, false
	}
	match := r.regex.FindStringSubmatch(path)
	if match == nil {
		return nil, false
	}
	for i, name := range r.regex.SubexpNames() {
		if i != 0 && name != "" {
			params = append(params, gin.Param{
				Key:   name,
				Value: match[i],
			})
		}
	}
	return params, true
}

// patchEngine inserts the regular expression router handler into the beginning
// of `gin.Engine.Handlers` and `gin.Engine.allNoRoute`.
//
// It covers all three cases of `gin.Engine.NoRoute`.
//
// 1. NoRoute called before setup RegexRouter, then `gin.Engine.allNoRoute` has
//    already been set, we insert the regex router into it here.
//
// 2. NoRoute called after setup RegexRouter, `gin.Engine.NoRoute` will rebuild
//    allNoRoute handlers using `gin.Engine.Handlers`, we insert the regex router
//    into `gin.Engine.Handlers`, then it will be the first handler of the final
//    allNoRoute handlers.
//
// 3. NoRoute never been called, then `gin.Engine.allNoRoute` is empty,
//    we fill it with the regex router here.
func (r *RegexRouter) patchEngine() {
	routeFunc := r.route()
	r.engine.Handlers = append(gin.HandlersChain{routeFunc}, r.engine.Handlers...)
	engPtr := unsafe.Pointer(r.engine)
	noRoutePtr := (*gin.HandlersChain)(unsafe.Pointer(uintptr(engPtr) + engineAllNoRouteOffset))
	*noRoutePtr = append(gin.HandlersChain{routeFunc}, *noRoutePtr...)
}

func (r *RegexRouter) route() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.isNoRouteRequest(c) {
			return
		}
		methodHandlers, ok := r.table[c.Request.Method]
		if !ok {
			return
		}
		path := c.Request.URL.Path
		for _, h := range methodHandlers {
			params, match := h.match(path, c.Params)
			if !match {
				continue
			}
			c.Params = params
			r.patchHandlers(c, h)
			if r.hook != nil {
				r.hook(c, h.pattern)
			}
			c.Next()
			return
		}
	}
}

func (r *RegexRouter) isNoRouteRequest(c *gin.Context) bool {
	ctxPtr := unsafe.Pointer(c)
	handlersPtr := (*gin.HandlersChain)(unsafe.Pointer(uintptr(ctxPtr) + contextHandlersOffset))
	if handlersPtr == nil || len(*handlersPtr) == 0 {
		return true
	}
	engPtr := unsafe.Pointer(r.engine)
	noRoutePtr := (*gin.HandlersChain)(unsafe.Pointer(uintptr(engPtr) + engineAllNoRouteOffset))

	isSameHandlers := func(a, b *gin.HandlersChain) bool {
		aptr := (*reflect.SliceHeader)(unsafe.Pointer(a)).Data
		bptr := (*reflect.SliceHeader)(unsafe.Pointer(b)).Data
		return aptr == bptr
	}
	return isSameHandlers(handlersPtr, noRoutePtr)
}

// patchHandlers replaces gin.Context.handlers to handlers of the matched regular expression.
//
// When it comes here, the gin.Context's old handlers are:
//     [RegexRouter.route(), gin.Engine.allNoRoute...]
//     and gin.Context.index is 0;
//
// after patching, it will be replace to:
//     [RegexRouter.route(), gin.Engine.Handlers..., RegexRouter.handlers..., regexHandler.handlers...]
//     and gin.Context.index won't be changed (remains 0);
//
// After this function returns, gin.Context.Next() will be called, then it goes to the
// registered business middleware and endpoint handlers.
func (r *RegexRouter) patchHandlers(c *gin.Context, h *regexHandler) {
	combinedHandlers := (*gin.HandlersChain)(atomic.LoadPointer(&h.combinedHandlers))
	if combinedHandlers == nil {
		combinedHandlers = r.combineHandlers(h)
		atomic.StorePointer(&h.combinedHandlers, unsafe.Pointer(combinedHandlers))
	}
	ctxPtr := unsafe.Pointer(c)
	handlersPtr := (*gin.HandlersChain)(unsafe.Pointer(uintptr(ctxPtr) + contextHandlersOffset))
	*handlersPtr = *combinedHandlers
}

func (r *RegexRouter) combineHandlers(h *regexHandler) *gin.HandlersChain {
	combinedHandlers := make(gin.HandlersChain, 0, len(r.engine.Handlers)+len(r.handlers)+len(h.handlers))
	combinedHandlers = append(combinedHandlers, r.engine.Handlers...)
	combinedHandlers = append(combinedHandlers, r.handlers...)
	combinedHandlers = append(combinedHandlers, h.handlers...)
	return &combinedHandlers
}

var (
	engineAllNoRouteOffset uintptr
	contextHandlersOffset  uintptr
)

func initOffset() {
	engTyp := reflect.TypeOf(gin.Engine{})
	allNoRouteField, ok := engTyp.FieldByName("allNoRoute")
	if !ok {
		panic("unsupported gin version without Engine.allNoRoute field")
	}
	ctxTyp := reflect.TypeOf(gin.Context{})
	handlersField, ok := ctxTyp.FieldByName("handlers")
	if !ok {
		panic("unsupported gin version without Context.handlers field")
	}
	engineAllNoRouteOffset = allNoRouteField.Offset
	contextHandlersOffset = handlersField.Offset
}
