# ginregex

This package implements an regular expression router for the gin http framework.

## Why

`gin` is an HTTP framework that claims high performance. Its main performance advantage comes from the `httprouter` that utilize trie and does not support regular expression. So why bother to build a regular expression router?

It starts with a story.

In the process of maintaining a legacy Python service that has passed several teams in succession, after many optimizations to the old service, the performance of the program and the maintainability of the project are still a headache. Well, everyone knows the pain of maintaining a long-lived and full of history debt project written in a dynamically typed language.

And over time, the company's technology stack has gradually shifted to the Go language. In order to solve the annoying historical problems, it is an obvious solution to rebuild the project in Go language gradually. But reconstruction is not a work we can accomplish at one stroke. Instead we must keep the service runs smoothly and be compatible with many old version clients that have been released to end-users.

First we build a gateway service using Go to take over all ingress requests, then we refactor the interfaces step by step. For the interfaces that has not been migrated, we reverse proxy the requests to the old service, here come the problems:

1. There are hundreds of path routes, configure them one by one?

2. Python (Flask) supports very flexible routing configuration, for example:

    - `/users/settings/` to query and modify the currently logged in user settings
    - `/users/some_biz_data/` to query some business data of the currently logged in user
    - `/users/<int:user_id>/` to query the information of the specified user by identifier

   This routing configuration is not supported in `gin` (`httprouter`).

Therefore, we have no choice other than regular expression routing. I did not find a suitable open source implementation, so I decided to build one by myself.

## Usage

Install:

`go get github.com/jxskiss/ginregex`

Example:

```go
r := gin.Default()

// Configure your normal non regular expression routes.
r.GET("/path_1", handler1)
r.POST("/path_2", handler2)

// Optionally you may configure your NoRoute handler, it won't affect the regex router.
r.NoRoute(noRouteHandler)

// Configure your regular expression routes.
regexRouter := ginregex.New(r, nil)
regexRouter.GET("^/.*$", regexHandler)

r.Run(":8080")
```

Some notes:

1. the `RegexRouter` is compatible with the NoRoute handler of `gin.Engine`, so you can optionally configure it anywhere as you want;

1. the order of registering normal no-regex routes and regex routes does not matter;

1. but registered regular expressions must be unique, duplicate regular expressions will cause panic;

1. middleware added to the `gin.Engine` or the `RegexRouter` will be combined into the final regex handler;

1. named capturing will be filled into gin.Context.Params, you can access them like normal gin path parameters in handler or binding functions;

1. optionally an hook function (`func(c *gin.Context, params gin.Params, httpMethod, regexPattern string)`) can be provided when building the `RegexRouter`, it will be called right after the request path being matched by a regular expression; you may use this feature to integrate with other `gin` middleware;
