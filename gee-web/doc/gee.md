---
title: 7天用Go从零实现Web框架Gee教程
date: 2019-08-11 02:10:10
description: 7天用 Go语言 从零实现Web框架教程(7 days implement golang web framework from scratch tutorial)，用 Go语言/golang 动手写Web框架，从零实现一个Web框架，以 Gin 为原型从零设计一个Web框架。
tags:
- Go
nav: 从零实现
categories:
- Web框架 - Gee
keywords:
- Gee教程
- 从零实现Web框架
- 动手写
- from scratch
image: post/gee/gee.jpg
github: https://github.com/glaxry/7days-golang
book: 七天用Go从零实现系列
book_title: Day0 序言
---

![gee](gee/gee.jpg)

## 设计一个框架

大部分时候，我们需要实现一个 Web 应用，第一反应是应该使用哪个框架。不同的框架设计理念和提供的功能有很大的差别。比如 Python 语言的 `django`和`flask`，前者大而全，后者小而美。Go语言/golang 也是如此，新框架层出不穷，比如`Beego`，`Gin`，`Iris`等。那为什么不直接使用标准库，而必须使用框架呢？在设计一个框架之前，我们需要回答框架核心为我们解决了什么问题。只有理解了这一点，才能想明白我们需要在框架中实现什么功能。

我们先看看标准库`net/http`如何处理一个请求。

```go
func main() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/count", counter)
    log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "URL.Path = %q\n", r.URL.Path)
}
```

`net/http`提供了基础的Web功能，即监听端口，映射静态路由，解析HTTP报文。一些Web开发中简单的需求并不支持，需要手工实现。

- 动态路由：例如`hello/:name`，`hello/*`这类的规则。
- 鉴权：没有分组/统一鉴权的能力，需要在每个路由映射的handler中实现。
- 模板：没有统一简化的HTML机制。
- ...

当我们离开框架，使用基础库时，需要频繁手工处理的地方，就是框架的价值所在。但并不是每一个频繁处理的地方都适合在框架中完成。Python有一个很著名的Web框架，名叫[`bottle`](https://github.com/bottlepy/bottle)，整个框架由`bottle.py`一个文件构成，共4400行，可以说是一个微框架。那么理解这个微框架提供的特性，可以帮助我们理解框架的核心能力。

- 路由(Routing)：将请求映射到函数，支持动态路由。例如`'/hello/:name`。
- 模板(Templates)：使用内置模板引擎提供模板渲染机制。
- 工具集(Utilites)：提供对 cookies，headers 等处理机制。
- 插件(Plugin)：Bottle本身功能有限，但提供了插件机制。可以选择安装到全局，也可以只针对某几个路由生效。
- ...

## Gee 框架

这个教程将使用 Go 语言实现一个简单的 Web 框架，起名叫做`Gee`，[`geektutu.com`](https://geektutu.com)的前三个字母。我第一次接触的 Go 语言的 Web 框架是`Gin`，`Gin`的代码总共是14K，其中测试代码9K，也就是说实际代码量只有**5K**。`Gin`也是我非常喜欢的一个框架，与Python中的`Flask`很像，小而美。

`7天实现Gee框架`这个教程的很多设计，包括源码，参考了`Gin`，大家可以看到很多`Gin`的影子。

时间关系，同时为了尽可能地简洁明了，这个框架中的很多部分实现的功能都很简单，但是尽可能地体现一个框架核心的设计原则。例如`Router`的设计，虽然支持的动态路由规则有限，但为了性能考虑匹配算法是用`Trie树`实现的，`Router`最重要的指标之一便是性能。

希望这个教程能够对你有所启发，如果对 Gee 有任何好的建议，欢迎提[issues - Github](https://github.com/glaxry/7days-golang/issues) 和 PR。教程中的任何问题，可以直接在文章末尾评论。

## 目录

- 第一天：[前置知识(http.Handler接口)](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day1.md)，[Code - Github](https://github.com/glaxry/7days-golang/tree/main/gee-web/day1-http-base)
- 第二天：[上下文设计(Context)](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day2.md)，[Code - Github](https://github.com/glaxry/7days-golang/tree/main/gee-web/day2-context)
- 第三天：[Trie树路由(Router)](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day3.md)，[Code - Github](https://github.com/glaxry/7days-golang/tree/main/gee-web/day3-router)
- 第四天：[分组控制(Group)](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day4.md)，[Code - Github](https://github.com/glaxry/7days-golang/tree/main/gee-web/day4-group)
- 第五天：[中间件(Middleware)](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day5.md)，[Code - Github](https://github.com/glaxry/7days-golang/tree/main/gee-web/day5-middleware)
- 第六天：[HTML模板(Template)](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day6.md)，[Code - Github](https://github.com/glaxry/7days-golang/tree/main/gee-web/day6-template)
- 第七天：[错误恢复(Panic Recover)](https://github.com/glaxry/7days-golang/blob/main/gee-web/doc/gee-day7.md)，[Code - Github](https://github.com/glaxry/7days-golang/tree/main/gee-web/day7-panic-recover)

## 推荐阅读

- [Go 语言简明教程](https://geektutu.com/post/quick-golang.html)
- [Go Test 单元测试简明教程](https://geektutu.com/post/quick-golang.html)
- [Go Gin 简明教程](https://geektutu.com/post/quick-go-gin.html)

## 通俗学习地图：7 天是在搭一条请求流水线

第一次接触 Web 框架时，很容易把路由、上下文、中间件、模板看成互不相干的功能。更好的理解方式，是始终跟着“一次 HTTP 请求”往前走：

```text
浏览器发出请求
    ↓
Engine 接收请求
    ↓
Router 根据“请求方法 + 路径”找到处理函数
    ↓
Context 把请求、响应、路径参数等信息装到一起
    ↓
Middleware 在处理函数前后执行日志、恢复、鉴权等通用逻辑
    ↓
Handler 生成 JSON、HTML 或文件响应
    ↓
浏览器收到状态码、响应头和响应体
```

七天的内容其实是在逐段补齐这条流水线：

1. 第一天先让服务器能启动，并能把固定路径交给固定函数处理。
2. 第二天用 `Context` 收拢一次请求所需的数据和输出方法，避免业务函数到处传参数。
3. 第三天把静态表升级为前缀树，让 `/hello/:name` 这类动态路径也能匹配。
4. 第四天增加路由分组，统一管理共同前缀和同一范围内的配置。
5. 第五天加入中间件，让日志、鉴权、计时等逻辑可以像洋葱一样包住业务处理。
6. 第六天支持模板和静态资源，使框架不仅能返回文本，也能组成完整网页。
7. 第七天用错误恢复兜住单次请求中的 `panic`，避免一个请求拖垮整个服务。

学习时不要急着背接口。每读到一个结构体，先问三个问题：它替谁保存状态？它在请求流程的哪一步被创建？下一个组件为什么需要它？能顺着上面的箭头回答出来，就说明已经真正理解，而不只是把代码抄了一遍。

这个教学框架刻意省略了生产框架中的许多能力，例如参数校验、路由冲突检查、优雅停机、完整的错误类型和安全策略。它的目标不是直接替代 Gin，而是把一个 Web 框架最核心的骨架展示清楚。
