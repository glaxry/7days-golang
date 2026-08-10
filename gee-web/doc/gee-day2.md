---
title: Go语言动手写Web框架 - Gee第二天 上下文Context
date: 2019-08-19 00:10:10
description: 7天用 Go语言 从零实现Web框架教程(7 days implement golang web framework from scratch tutorial)，用 Go语言/golang 动手写Web框架，从零实现一个Web框架，以 Gin 为原型从零设计一个Web框架。本文介绍了请求上下文(Context)的设计理念，封装了返回JSON/String/Data/HTML等类型响应的方法。
tags:
- Go
nav: 从零实现
categories:
- Web框架 - Gee
keywords:
- Go语言
- 从零实现Web框架
- 动手写Web框架
- Context
image: post/gee/gee.jpg
github: https://github.com/glaxry/7days-golang
book: 七天用Go从零实现系列
book_title: Day2 上下文
---

本文是 [7天用Go从零实现Web框架Gee教程系列](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee.md)的第二篇。

- 将`路由(router)`独立出来，方便之后增强。
- 设计`上下文(Context)`，封装 Request 和 Response ，提供对 JSON、HTML 等返回类型的支持。
- 动手写 Gee 框架的第二天，**框架代码140行，新增代码约90行**

## 使用效果

为了展示第二天的成果，我们看一看在使用时的效果。

[day2-context/main.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day2-context)


```go
func main() {
	r := gee.New()
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})
	r.GET("/hello", func(c *gee.Context) {
		// expect /hello?name=geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.POST("/login", func(c *gee.Context) {
		c.JSON(http.StatusOK, gee.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	r.Run(":9999")
}
```

- `Handler`的参数变成成了`gee.Context`，提供了查询Query/PostForm参数的功能。
- `gee.Context`封装了`HTML/String/JSON`函数，能够快速构造HTTP响应。

## 设计Context

### 必要性

1. 对Web服务来说，无非是根据请求`*http.Request`，构造响应`http.ResponseWriter`。但是这两个对象提供的接口粒度太细，比如我们要构造一个完整的响应，需要考虑消息头(Header)和消息体(Body)，而 Header 包含了状态码(StatusCode)，消息类型(ContentType)等几乎每次请求都需要设置的信息。因此，如果不进行有效的封装，那么框架的用户将需要写大量重复，繁杂的代码，而且容易出错。针对常用场景，能够高效地构造出 HTTP 响应是一个好的框架必须考虑的点。

用返回 JSON 数据作比较，感受下封装前后的差距。

封装前

```go
obj = map[string]any{
    "name": "geektutu",
    "password": "1234",
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
encoder := json.NewEncoder(w)
if err := encoder.Encode(obj); err != nil {
    http.Error(w, err.Error(), 500)
}
```

VS 封装后：

```go
c.JSON(http.StatusOK, gee.H{
    "username": c.PostForm("username"),
    "password": c.PostForm("password"),
})
```


2. 针对使用场景，封装`*http.Request`和`http.ResponseWriter`的方法，简化相关接口的调用，只是设计 Context 的原因之一。对于框架来说，还需要支撑额外的功能。例如，将来解析动态路由`/hello/:name`，参数`:name`的值放在哪呢？再比如，框架需要支持中间件，那中间件产生的信息放在哪呢？Context 随着每一个请求的出现而产生，请求的结束而销毁，和当前请求强相关的信息都应由 Context 承载。因此，设计 Context 结构，扩展性和复杂性留在了内部，而对外简化了接口。路由的处理函数，以及将要实现的中间件，参数都统一使用 Context 实例， Context 就像一次会话的百宝箱，可以找到任何东西。

### 具体实现

[day2-context/gee/context.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day2-context)

```go
type H map[string]any

type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	// response info
	StatusCode int
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) String(code int, format string, values ...any) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write(fmt.Appendf(nil, format, values...))
}

func (c *Context) JSON(code int, obj any) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}
```

- 代码最开头，给`map[string]any`起了一个别名`gee.H`，构建JSON数据时，显得更简洁。
- `Context`目前只包含了`http.ResponseWriter`和`*http.Request`，另外提供了对 Method 和 Path 这两个常用属性的直接访问。
- 提供了访问Query和PostForm参数的方法。
- 提供了快速构造String/Data/JSON/HTML响应的方法。

## 路由(Router)

我们将和路由相关的方法和结构提取了出来，放到了一个新的文件中`router.go`，方便我们下一次对 router 的功能进行增强，例如提供动态路由的支持。 router 的 handle 方法作了一个细微的调整，即 handler 的参数，变成了 Context。

[day2-context/gee/router.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day2-context)

```go
type router struct {
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{handlers: make(map[string]HandlerFunc)}
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Route %4s - %s", method, pattern)
	key := method + "-" + pattern
	r.handlers[key] = handler
}

func (r *router) handle(c *Context) {
	key := c.Method + "-" + c.Path
	if handler, ok := r.handlers[key]; ok {
		handler(c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}
```

## 框架入口

[day2-context/gee/gee.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day2-context)

```go
// HandlerFunc defines the request handler used by gee
type HandlerFunc func(*Context)

// Engine implement the interface of ServeHTTP
type Engine struct {
	router *router
}

// New is the constructor of gee.Engine
func New() *Engine {
	return &Engine{router: newRouter()}
}

func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}
```

将`router`相关的代码独立后，`gee.go`简单了不少。最重要的还是通过实现了 ServeHTTP 接口，接管了所有的 HTTP 请求。相比第一天的代码，这个方法也有细微的调整，在调用 router.handle 之前，构造了一个 Context 对象。这个对象目前还非常简单，仅仅是包装了原来的两个参数，之后我们会慢慢地给Context插上翅膀。

如何使用，`main.go`一开始就已经亮相了。运行`go run main.go`，借助 curl ，一起看一看今天的成果吧。

```bash
$ curl -i http://localhost:9999/
HTTP/1.1 200 OK
Date: Mon, 12 Aug 2019 16:52:52 GMT
Content-Length: 18
Content-Type: text/html; charset=utf-8
<h1>Hello Gee</h1>

$ curl "http://localhost:9999/hello?name=geektutu"
hello geektutu, you're at /hello

$ curl "http://localhost:9999/login" -X POST -d 'username=geektutu&password=1234'
{"password":"1234","username":"geektutu"}

$ curl "http://localhost:9999/xxx"
404 NOT FOUND: /xxx
```

## 白话复盘：Context 是一次请求的工作台

### 为什么需要它

第一天的处理函数直接接收 `http.ResponseWriter` 和 `*http.Request`。功能一多，业务代码还会需要路径参数、查询参数、状态码、响应格式以及中间件共享的数据。如果继续逐个传参，函数签名会越来越长，而且每个处理函数都要重复设置响应头、序列化 JSON。

`Context` 的作用就是给每次请求准备一张独立工作台：输入信息和输出工具都放在上面。它不是全局对象，也不要和标准库中用于取消、超时和值传递的 `context.Context` 混淆；Gee 的 `Context` 是 Web 框架自己的请求包装器。

### 一次请求怎样走

1. `ServeHTTP` 收到标准库提供的 `Writer` 和 `Request`。
2. 框架立即创建新的 `Context`，因此并发请求不会共享 `Path`、`StatusCode` 等可变字段。
3. `Router` 根据 `c.Method` 和 `c.Path` 查找处理函数。
4. 业务函数通过 `c.Query()` 读取参数，通过 `c.String()`、`c.JSON()` 或 `c.Data()` 生成响应。
5. 响应辅助方法统一设置 `Content-Type`、状态码并写入响应体。

这种设计的关键收益不是少写几个字符，而是建立稳定的扩展位置。后续动态路由会把 `Params` 放进来，中间件会通过它控制处理链，模板渲染也会复用同一套响应出口。

### 容易踩坑

- 一个请求必须创建一个 `Context`，不能为了省分配而随意把同一个对象交给并发请求使用。
- HTTP 响应头必须在写响应体之前确定；封装响应方法就是为了集中保证这个顺序。
- `Query("name")` 读的是 `?name=tom`，不是 `/hello/tom` 中的路径参数。路径参数会在下一天由路由器解析。
- JSON 编码也可能失败。教学代码较简洁，生产代码还应统一处理编码错误和“响应已经写出”的情况。

### 动手检查

新增一个 `/inspect?name=gee` 路由，同时返回请求方法、路径和查询参数。再并发请求不同的 `name`，确认每个响应都只包含自己的数据，这能直观看出 `Context` 的请求级生命周期。
