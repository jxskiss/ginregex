package ginregex

import (
	"github.com/gin-gonic/gin"
	"reflect"
	"regexp"
	"sync/atomic"
	"unsafe"
)

type regexHandler struct {
	pattern  string
	regex    *regexp.Regexp
	handlers gin.HandlersChain

	combinedHandlers unsafe.Pointer
}

func (r *regexHandler) match(c *gin.Context, path string) (gin.Params, bool) {
	match := r.regex.FindStringSubmatch(path)
	if match == nil {
		return nil, false
	}
	params := c.Params[:0]
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
			params, match := h.match(c, path)
			if !match {
				continue
			}

			combinedHandlers := (*gin.HandlersChain)(atomic.LoadPointer(&h.combinedHandlers))
			if combinedHandlers == nil {
				combinedHandlers = r.combineHandlers(h)
				atomic.StorePointer(&h.combinedHandlers, unsafe.Pointer(combinedHandlers))
			}
			r.patchContext(c, params, *combinedHandlers)
			if r.hook != nil {
				r.hook(c, params, c.Request.Method, h.pattern)
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

func (r *RegexRouter) combineHandlers(h *regexHandler) *gin.HandlersChain {
	combinedHandlers := make(gin.HandlersChain, 0, len(r.engine.Handlers)+len(r.handlers)+len(h.handlers))
	combinedHandlers = append(combinedHandlers, r.engine.Handlers...)
	combinedHandlers = append(combinedHandlers, r.handlers...)
	combinedHandlers = append(combinedHandlers, h.handlers...)
	return &combinedHandlers
}

func (r *RegexRouter) patchContext(c *gin.Context, params gin.Params, handlers gin.HandlersChain) {
	ctxPtr := unsafe.Pointer(c)
	handlersPtr := (*gin.HandlersChain)(unsafe.Pointer(uintptr(ctxPtr) + contextHandlersOffset))
	*handlersPtr = handlers
	if params != nil {
		c.Params = params
	}
}

var (
	engineAllNoRouteOffset uintptr
	contextHandlersOffset  uintptr
	contextIndexOffset     uintptr
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
	indexField, ok := ctxTyp.FieldByName("index")
	if !ok {
		panic("unsupported gin version without Context.index field")
	}
	engineAllNoRouteOffset = allNoRouteField.Offset
	contextHandlersOffset = handlersField.Offset
	contextIndexOffset = indexField.Offset
}
