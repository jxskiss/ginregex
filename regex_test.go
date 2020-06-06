package ginregex

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

func TestRegexpHandler(t *testing.T) {
	tests := []map[string]interface{}{
		{
			"pattern": `^/myapp/users/(?P<user_id>\d+)/$`,
			"path":    "/myapp/not_match/1234/",
			"match":   false,
			"params":  (gin.Params)(nil),
		},
		{
			"pattern": `^/myapp/users/(?P<user_id>\d+)/$`,
			"path":    "/myapp/users/123456/",
			"match":   true,
			"params":  gin.Params{{Key: "user_id", Value: "123456"}},
		},
		{
			"pattern": `^/myapp/users\d+/$`,
			"path":    "/myapp/users123456/",
			"match":   true,
			"params":  (gin.Params)(nil),
		},
	}
	for _, test := range tests {
		c := &gin.Context{}
		c.Request, _ = http.NewRequest("GET", test["path"].(string), nil)
		handler := newRegexHandler(test["pattern"].(string), nil)
		params, ok := handler.match(c.Request.URL.Path, c.Params)
		assert.Equal(t, test["match"].(bool), ok)
		assert.Equal(t, test["params"].(gin.Params), params)
	}
}

var notMatchPath = "/myapp/not_match/1234/"
var usersHandler = newRegexHandler(`^/myapp/users/(?P<user_id>\d+)/$`, nil)

func BenchmarkRegexpNotMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = usersHandler.regex.FindStringSubmatch(notMatchPath)
	}
}

func BenchmarkPrefixNotMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = strings.HasPrefix(notMatchPath, usersHandler.prefix)
	}
}
