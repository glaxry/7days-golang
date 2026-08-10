---
title: Go语言动手写Web框架 - Gee第五天 中间件Middleware
date: 2019-09-01 20:10:10
description: 7天用 Go语言 从零实现Web框架教程(7 days implement golang web framework from scratch tutorial)，用 Go语言/golang 动手写Web框架，从零实现一个Web框架，以 Gin 为原型从零设计一个Web框架。本文介绍了如何为Web框架添加中间件的功能(middlewares)。
tags:
- Go
nav: 从零实现
categories:
- Web框架 - Gee
keywords:
- Go语言
- 从零实现Web框架
- 动手写Web框架
- Middlewares
image: post/gee-day5/middleware.jpg
github: https://github.com/glaxry/7days-golang
book: 七天用Go从零实现系列
book_title: Day5 中间件
---

本文是 [7天用Go从零实现Web框架Gee教程系列](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee.md)的第五篇。

- 设计并实现 Web 框架的中间件(Middlewares)机制。
- 实现通用的`Logger`中间件，能够记录请求到响应所花费的时间，**代码约50行**

## 中间件是什么

中间件(middlewares)，简单说，就是非业务的技术类组件。Web 框架本身不可能去理解所有的业务，因而不可能实现所有的功能。因此，框架需要有一个插口，允许用户自己定义功能，嵌入到框架中，仿佛这个功能是框架原生支持的一样。因此，对中间件而言，需要考虑2个比较关键的点：

- 插入点在哪？使用框架的人并不关心底层逻辑的具体实现，如果插入点太底层，中间件逻辑就会非常复杂。如果插入点离用户太近，那和用户直接定义一组函数，每次在 Handler 中手工调用没有多大的优势了。
- 中间件的输入是什么？中间件的输入，决定了扩展能力。暴露的参数太少，用户发挥空间有限。

那对于一个 Web 框架而言，中间件应该设计成什么样呢？接下来的实现，基本参考了 Gin 框架。

## 中间件设计

Gee 的中间件的定义与路由映射的 Handler 一致，处理的输入是`Context`对象。插入点是框架接收到请求初始化`Context`对象后，允许用户使用自己定义的中间件做一些额外的处理，例如记录日志等，以及对`Context`进行二次加工。另外通过调用`(*Context).Next()`函数，中间件可等待用户自己定义的 `Handler`处理结束后，再做一些额外的操作，例如计算本次处理所用时间等。即 Gee 的中间件支持用户在请求被处理的前后，做一些额外的操作。举个例子，我们希望最终能够支持如下定义的中间件，`c.Next()`表示等待执行其他的中间件或用户的`Handler`：

****[day4-group/gee/logger.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day5-middleware)****

```go
func Logger() HandlerFunc {
	return func(c *Context) {
		// Start timer
		t := time.Now()
		// Process request
		c.Next()
		// Calculate resolution time
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
```

另外，支持设置多个中间件，依次进行调用。

我们上一篇文章[分组控制 Group Control](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day4.md)中讲到，中间件是应用在`RouterGroup`上的，应用在最顶层的 Group，相当于作用于全局，所有的请求都会被中间件处理。那为什么不作用在每一条路由规则上呢？作用在某条路由规则，那还不如用户直接在 Handler 中调用直观。只作用在某条路由规则的功能通用性太差，不适合定义为中间件。

我们之前的框架设计是这样的，当接收到请求后，匹配路由，该请求的所有信息都保存在`Context`中。中间件也不例外，接收到请求后，应查找所有应作用于该路由的中间件，保存在`Context`中，依次进行调用。为什么依次调用后，还需要在`Context`中保存呢？因为在设计中，中间件不仅作用在处理流程前，也可以作用在处理流程后，即在用户定义的 Handler 处理完毕后，还可以执行剩下的操作。

为此，我们给`Context`添加了2个参数，定义了`Next`方法：

**[day4-group/gee/context.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day5-middleware)**

```go
type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
	// middleware
	handlers []HandlerFunc
	index    int
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Path:   req.URL.Path,
		Method: req.Method,
		Req:    req,
		Writer: w,
		index:  -1,
	}
}

func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}
```

`index`是记录当前执行到第几个中间件，当在中间件中调用`Next`方法时，控制权交给了下一个中间件，直到调用到最后一个中间件，然后再从后往前，调用每个中间件在`Next`方法之后定义的部分。如果我们将用户在映射路由时定义的`Handler`添加到`c.handlers`列表中，结果会怎么样呢？想必你已经猜到了。


```
func A(c *Context) {
    part1
    c.Next()
    part2
}
func B(c *Context) {
    part3
    c.Next()
    part4
}
```

假设我们应用了中间件 A 和 B，和路由映射的 Handler。`c.handlers`是这样的[A, B, Handler]，`c.index`初始化为-1。调用`c.Next()`，接下来的流程是这样的：

- c.index++，c.index 变为 0
- 0 < 3，调用 c.handlers[0]，即 A
- 执行 part1，调用 c.Next()
- c.index++，c.index 变为 1
- 1 < 3，调用 c.handlers[1]，即 B
- 执行 part3，调用 c.Next()
- c.index++，c.index 变为 2
- 2 < 3，调用 c.handlers[2]，即Handler
- Handler 调用完毕，返回到 B 中的 part4，执行 part4
- part4 执行完毕，返回到 A 中的 part2，执行 part2
- part2 执行完毕，结束。

一句话说清楚重点，最终的顺序是`part1 -> part3 -> Handler -> part 4 -> part2`。恰恰满足了我们对中间件的要求，接下来看调用部分的代码，就能全部串起来了。


## 代码实现

- 定义`Use`函数，将中间件应用到某个 Group 。

**[day4-group/gee/gee.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day5-middleware)**

```go
// Use is defined to add middleware to the group
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, req)
	c.handlers = middlewares
	engine.router.handle(c)
}
```

ServeHTTP 函数也有变化，当我们接收到一个具体请求时，要判断该请求适用于哪些中间件，在这里我们简单通过 URL 的前缀来判断。得到中间件列表后，赋值给 `c.handlers`。

- handle 函数中，将从路由匹配得到的 Handler 添加到 `c.handlers`列表中，执行`c.Next()`。

**[day4-group/gee/router.go](https://github.com/glaxry/7days-golang/tree/main/gee-web/day5-middleware)**

```go
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)

	if n != nil {
		key := c.Method + "-" + n.pattern
		c.Params = params
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	c.Next()
}
```

## 使用 Demo

```go
func onlyForV2() gee.HandlerFunc {
	return func(c *gee.Context) {
		// Start timer
		t := time.Now()
		// if a server error occurred
		c.Fail(500, "Internal Server Error")
		// Calculate resolution time
		log.Printf("[%d] %s in %v for group v2", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

func main() {
	r := gee.New()
	r.Use(gee.Logger()) // global midlleware
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})

	v2 := r.Group("/v2")
	v2.Use(onlyForV2()) // v2 group middleware
	{
		v2.GET("/hello/:name", func(c *gee.Context) {
			// expect /hello/geektutu
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
		})
	}

	r.Run(":9999")
}
```

`gee.Logger()`即我们一开始就介绍的中间件，我们将这个中间件和框架代码放在了一起，作为框架默认提供的中间件。在这个例子中，我们将`gee.Logger()`应用在了全局，所有的路由都会应用该中间件。`onlyForV2()`是用来测试功能的，仅在`v2`对应的 Group 中应用了。

接下来使用 curl 测试，可以看到，v2 Group 2个中间件都生效了。

```bash
$ curl http://localhost:9999/
>>> log
2019/08/17 01:37:38 [200] / in 3.14µs

(2) global + group middleware
$ curl http://localhost:9999/v2/hello/geektutu
>>> log
2019/08/17 01:38:48 [200] /v2/hello/geektutu in 61.467µs for group v2
2019/08/17 01:38:48 [200] /v2/hello/geektutu in 281µs
```

## 白话复盘：中间件就是可组合的请求处理链

### 用“洋葱”理解执行顺序

路由函数只应关心具体业务，但记录耗时、捕获异常、校验身份等逻辑几乎每个路由都需要。中间件把这些通用逻辑放在业务函数外层，并允许多个中间件组合。

假设顺序是 `Logger -> Auth -> Handler`，执行并不是简单地把三个函数从头跑到尾，而是：

```text
Logger 前半段
  Auth 前半段
    Handler
  Auth 后半段
Logger 后半段
```

每个中间件在调用 `c.Next()` 前写“进入逻辑”，在它返回后写“退出逻辑”。这就是所谓洋葱模型，也是计时中间件能在业务完成后读取状态码和耗时的原因。

### `Next` 为什么能继续执行

框架先根据请求路径收集所有适用分组的中间件，再把最终路由处理函数放到链尾。`Context` 保存 `handlers` 和当前下标 `index`。`Next` 递增下标并执行后续函数；后续函数全部返回后，控制权又逐层回到前面的中间件。

顺序非常重要。恢复中间件通常应放在外层，才能捕获后续代码的 `panic`；日志中间件放在它的内外侧，记录到的信息也可能不同。中间件不是“注册了就行”，还要明确它包住了谁。

### 容易踩坑

- 在同一个中间件里多次调用 `Next`，可能导致后续处理器重复执行或流程难以推断。
- 把请求级数据放在包级变量中会产生并发竞争，应放在 `Context` 或由它引用的请求级对象中。
- 当前教学版没有完整的 `Abort` 机制。鉴权失败时应写出响应并确保后续业务不再执行，生产框架会提供更明确的中止 API。
- 中间件范围来自路由分组；注册到根组会影响所有请求，注册前先确认作用域。

### 动手检查

写两个中间件，分别在 `Next` 前后打印 A/a 和 B/b。请求一次后，预期顺序应为 `A B handler b a`。如果能解释为什么小写部分倒序出现，就真正理解了处理链。
