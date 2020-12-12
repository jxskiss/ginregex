package ginregex

import (
	"github.com/gin-gonic/gin"
	"regexp"
)

type Matcher struct {
	Method   string
	RE       *regexp.Regexp
	Handlers gin.HandlersChain
}

func (m *Matcher) Handle(c *gin.Context) {
	for _, h := range m.Handlers {
		h(c)
		if c.IsAborted() {
			return
		}
	}
}

// NewMatcher returns a matcher to be used with the `Dispatch` function
// to dispatch requests using regular expressions. Named capturing will be
// filled into `gin.Context.Params`, you can access them like normal
// gin path parameters in your handler or binding functions.
//
// The given re string must be a valid regular expression, else it will panic.
func NewMatcher(method, re string, handlers ...gin.HandlerFunc) *Matcher {
	return &Matcher{
		Method:   method,
		RE:       regexp.MustCompile(re),
		Handlers: handlers,
	}
}

// Dispatch can be used to dispatch requests to multiple different handlers
// under same path, using the given matchers.
// The matchers given order matters, if a path matches multiple matchers,
// only the first one (in the order they present) takes effect.
//
// Compared to the `RegexRouter` in this package, `Dispatch` does not use
// unsafe tricks to hack the framework at runtime.
//
// Usage example:
//
// 	r := gin.Default()
// 	r.Any("/users/*any", ginregex.Dispatch(
// 		ginregex.NewMatcher("GET", `^/users/settings/$`, myMiddleware1, GetUserSettings),
// 		ginregex.NewMatcher("POST", `^/users/settings/$`, myMiddleware1, ModifyUserSettings),
// 		ginregex.NewMatcher("GET", `^/users/(?P<user_id>\d+)/$`, myMiddleware2, GetUserDetail),
// 	))
func Dispatch(matchers ...*Matcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path
		for _, m := range matchers {
			if method == m.Method {
				match := m.RE.FindStringSubmatch(path)
				if len(match) == 0 {
					continue
				}
				var params gin.Params
				for i, name := range m.RE.SubexpNames() {
					if i != 0 && name != "" {
						params = append(params, gin.Param{
							Key:   name,
							Value: match[i],
						})
					}
				}
				if len(params) > 0 {
					c.Params = params
				}
				m.Handle(c)
				return
			}
		}
		c.Status(404)
	}
}
