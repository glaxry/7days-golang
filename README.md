# 7 days golang programs from scratch

[![CodeSize](https://img.shields.io/github/languages/code-size/geektutu/7days-golang)](https://github.com/geektutu/7days-golang)
[![LICENSE](https://img.shields.io/badge/license-MIT-green)](https://mit-license.org/)

## 2026 现代化版本说明

本分支以 **Go 1.26** 为语言基线（当前稳定补丁版为 Go 1.26.5），保留“每天一个可独立运行快照”的教学结构，并同步更新了所有 Day 的代码与文中代码片段。

- 模块声明从 Go 1.13 升级到 Go 1.26，采用 `any`、整数 `range`、`reflect.TypeFor`、反射迭代器和 `fmt.Appendf` 等当前写法。
- GeeORM 改用无 CGO 的 `modernc.org/sqlite v1.56.0`，Windows、macOS 和 Linux 不再需要额外安装 GCC。
- GeeCache 使用 `google.golang.org/protobuf v1.36.11`，补充 `.proto` 源文件，并用当前生成器重建代码。
- GeeRPC 修复握手预读丢包、连接超时 goroutine 泄漏、服务端超时 goroutine 泄漏和重复响应风险。
- GeeBolt 的 mmap 草稿已补成 Windows/Unix 均可运行的示例，并增加持久化测试。
- GitHub Actions 会在 Windows 和 Linux 上测试全部 45 个 Go 模块，并编译 4 个 WebAssembly 示例。

每个 Day 都是独立模块，请进入对应目录运行 `go test ./...`。也可以在仓库根目录一次验证全部内容：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-all.ps1
```

```bash
bash scripts/test-all.sh
```

升级依据、兼容性取舍和容易踩坑的地方见 [现代 Go 迁移说明](MODERNIZATION.md)。

<details>
<summary><strong>README 中文版本</strong></summary>
<div>

## 7天用Go从零实现系列

7天能写什么呢？类似 gin 的 web 框架？类似 groupcache 的分布式缓存？或者一个简单的 Python 解释器？希望这个仓库能给你答案。

推荐先阅读 **[Go 语言简明教程](https://geektutu.com/post/quick-golang.html)**，一篇文章了解Go的基本语法、并发编程，依赖管理等内容。

推荐 **[Go 语言笔试面试题](https://geektutu.com/post/qa-golang.html)**，加深对 Go 语言的理解。

推荐 **[Go 语言高性能编程](https://geektutu.com/post/high-performance-go.html)**([项目地址](https://github.com/geektutu/high-performance-go))，写出高性能的 Go 代码。

期待关注我的「[知乎专栏](https://zhuanlan.zhihu.com/geekgo)」和「[微博](http://weibo.com/geektutu)」，查看最近的文章和动态。

### 7天用Go从零实现Web框架 - Gee

[Gee](https://geektutu.com/post/gee.html) 是一个模仿 [gin](https://github.com/gin-gonic/gin) 实现的 Web 框架，[Go Gin简明教程](https://geektutu.com/post/quick-go-gin.html)可以快速入门。

- 第一天：[前置知识(http.Handler接口)](https://geektutu.com/post/gee-day1.html) | [Code](gee-web/day1-http-base)
- 第二天：[上下文设计(Context)](https://geektutu.com/post/gee-day2.html) | [Code](gee-web/day2-context)
- 第三天：[Trie树路由(Router)](https://geektutu.com/post/gee-day3.html) | [Code](gee-web/day3-router)
- 第四天：[分组控制(Group)](https://geektutu.com/post/gee-day4.html) | [Code](gee-web/day4-group)
- 第五天：[中间件(Middleware)](https://geektutu.com/post/gee-day5.html) | [Code](gee-web/day5-middleware)
- 第六天：[HTML模板(Template)](https://geektutu.com/post/gee-day6.html) | [Code](gee-web/day6-template)
- 第七天：[错误恢复(Panic Recover)](https://geektutu.com/post/gee-day7.html) | [Code](gee-web/day7-panic-recover)

### 7天用Go从零实现分布式缓存 GeeCache

[GeeCache](https://geektutu.com/post/geecache.html) 是一个模仿 [groupcache](https://github.com/golang/groupcache) 实现的分布式缓存系统

- 第一天：[LRU 缓存淘汰策略](https://geektutu.com/post/geecache-day1.html) | [Code](gee-cache/day1-lru)
- 第二天：[单机并发缓存](https://geektutu.com/post/geecache-day2.html) | [Code](gee-cache/day2-single-node)
- 第三天：[HTTP 服务端](https://geektutu.com/post/geecache-day3.html) | [Code](gee-cache/day3-http-server)
- 第四天：[一致性哈希(Hash)](https://geektutu.com/post/geecache-day4.html) | [Code](gee-cache/day4-consistent-hash)
- 第五天：[分布式节点](https://geektutu.com/post/geecache-day5.html) | [Code](gee-cache/day5-multi-nodes)
- 第六天：[防止缓存击穿](https://geektutu.com/post/geecache-day6.html) | [Code](gee-cache/day6-single-flight)
- 第七天：[使用 Protobuf 通信](https://geektutu.com/post/geecache-day7.html) | [Code](gee-cache/day7-proto-buf)

### 7天用Go从零实现ORM框架 GeeORM

[GeeORM](https://geektutu.com/post/geeorm.html) 是一个参考 [GORM](https://github.com/go-gorm/gorm) 和 [XORM](https://gitea.com/xorm/xorm) 设计的教学型 ORM 框架。

原教程写作时 GORM v2 尚在开发；现在 GORM v2 已是主版本。本教程仍保留更适合从零讲解的简化接口，不以复刻任何框架的当前完整 API 为目标。

- 第一天：[database/sql 基础](https://geektutu.com/post/geeorm-day1.html) | [Code](gee-orm/day1-database-sql)
- 第二天：[对象表结构映射](https://geektutu.com/post/geeorm-day2.html) | [Code](gee-orm/day2-reflect-schema)
- 第三天：[记录新增和查询](https://geektutu.com/post/geeorm-day3.html) | [Code](gee-orm/day3-save-query)
- 第四天：[链式操作与更新删除](https://geektutu.com/post/geeorm-day4.html) | [Code](gee-orm/day4-chain-operation)
- 第五天：[实现钩子(Hooks)](https://geektutu.com/post/geeorm-day5.html) | [Code](gee-orm/day5-hooks)
- 第六天：[支持事务(Transaction)](https://geektutu.com/post/geeorm-day6.html) | [Code](gee-orm/day6-transaction)
- 第七天：[数据库迁移(Migrate)](https://geektutu.com/post/geeorm-day7.html) | [Code](gee-orm/day7-migrate)


### 7天用Go从零实现RPC框架 GeeRPC

[GeeRPC](https://geektutu.com/post/geerpc.html) 是一个基于 [net/rpc](https://github.com/golang/go/tree/master/src/net/rpc) 开发的 RPC 框架
GeeRPC 是基于 Go 语言标准库 `net/rpc` 实现的，添加了协议交换、服务注册与发现、负载均衡等功能，代码约 1k。

- 第一天 - [服务端与消息编码](https://geektutu.com/post/geerpc-day1.html) | [Code](gee-rpc/day1-codec)
- 第二天 - [支持并发与异步的客户端](https://geektutu.com/post/geerpc-day2.html) | [Code](gee-rpc/day2-client)
- 第三天 - [服务注册(service register)](https://geektutu.com/post/geerpc-day3.html) | [Code](gee-rpc/day3-service )
- 第四天 - [超时处理(timeout)](https://geektutu.com/post/geerpc-day4.html) | [Code](gee-rpc/day4-timeout )
- 第五天 - [支持HTTP协议](https://geektutu.com/post/geerpc-day5.html) | [Code](gee-rpc/day5-http-debug)
- 第六天 - [负载均衡(load balance)](https://geektutu.com/post/geerpc-day6.html) | [Code](gee-rpc/day6-load-balance)
- 第七天 - [服务发现与注册中心(registry)](https://geektutu.com/post/geerpc-day7.html) | [Code](gee-rpc/day7-registry)

### WebAssembly 使用示例

具体的实践过程记录在 [Go WebAssembly 简明教程](https://geektutu.com/post/quick-go-wasm.html)。

- 示例一：Hello World | [Code](demo-wasm/hello-world)
- 示例二：注册函数 | [Code](demo-wasm/register-functions)
- 示例三：操作 DOM | [Code](demo-wasm/manipulate-dom)
- 示例四：回调函数 | [Code](demo-wasm/callback)

</div>
</details>

What can be accomplished in 7 days? A gin-like web framework? A distributed cache like groupcache? Or a simple Python interpreter? Hope this repo can give you the answer.

## Web Framework - Gee

[Gee](https://geektutu.com/post/gee.html) is a [gin](https://github.com/gin-gonic/gin)-like framework

- Day 1 - http.Handler Interface Basic [Code](gee-web/day1-http-base)
- Day 2 - Design a Flexiable Context [Code](gee-web/day2-context)
- Day 3 - Router with Trie-Tree Algorithm [Code](gee-web/day3-router)
- Day 4 - Group Control [Code](gee-web/day4-group)
- Day 5 - Middleware Mechanism [Code](gee-web/day5-middleware)
- Day 6 - Embeded Template Support [Code](gee-web/day6-template)
- Day 7 - Panic Recover & Make it Robust [Code](gee-web/day7-panic-recover)

## Distributed Cache - GeeCache

[GeeCache](https://geektutu.com/post/geecache.html) is a [groupcache](https://github.com/golang/groupcache)-like distributed cache

- Day 1 - LRU (Least Recently Used) Caching Strategy [Code](gee-cache/day1-lru)
- Day 2 - Single Machine Concurrent Cache [Code](gee-cache/day2-single-node)
- Day 3 - Launch a HTTP Server [Code](gee-cache/day3-http-server)
- Day 4 - Consistent Hash Algorithm [Code](gee-cache/day4-consistent-hash)
- Day 5 - Communication between Distributed Nodes [Code](gee-cache/day5-multi-nodes)
- Day 6 - Cache Breakdown & Single Flight  | [Code](gee-cache/day6-single-flight)
- Day 7 - Use Protobuf as RPC Data Exchange Type | [Code](gee-cache/day7-proto-buf)

## Object Relational Mapping - GeeORM

[GeeORM](https://geektutu.com/post/geeorm.html) is an educational ORM inspired by [GORM](https://github.com/go-gorm/gorm) and [XORM](https://gitea.com/xorm/xorm).

Xorm's desgin is easier to understand than gorm-v1, so the main designs references xorm and some detailed implementions references gorm-v1.

- Day 1 - database/sql Basic | [Code](gee-orm/day1-database-sql)
- Day 2 - Object Schame Mapping | [Code](gee-orm/day2-reflect-schema)
- Day 3 - Insert and Query | [Code](gee-orm/day3-save-query)
- Day 4 - Chain, Delete and Update | [Code](gee-orm/day4-chain-operation)
- Day 5 - Support Hooks | [Code](gee-orm/day5-hooks)
- Day 6 - Support Transaction | [Code](gee-orm/day6-transaction)
- Day 7 - Migrate Database | [Code](gee-orm/day7-migrate)

## RPC Framework - GeeRPC

[GeeRPC](https://geektutu.com/post/geerpc.html) is a [net/rpc](https://github.com/golang/go/tree/master/src/net/rpc)-like RPC framework

Based on golang standard library `net/rpc`, GeeRPC implements more features. eg, protocol exchange, service registration and discovery, load balance, etc.

- Day 1 - Server Message Codec | [Code](gee-rpc/day1-codec)
- Day 2 - Concurrent Client | [Code](gee-rpc/day2-client)
- Day 3 - Service Register | [Code](gee-rpc/day3-service )
- Day 4 - Timeout Processing | [Code](gee-rpc/day4-timeout )
- Day 5 - Support HTTP Protocol | [Code](gee-rpc/day5-http-debug)
- Day 6 - Load Balance | [Code](gee-rpc/day6-load-balance)
- Day 7 - Discovery and Registry | [Code](gee-rpc/day7-registry)

## Golang WebAssembly Demo

- Demo 1 - Hello World [Code](demo-wasm/hello-world)
- Demo 2 - Register Functions [Code](demo-wasm/register-functions)
- Demo 3 - Manipulate DOM [Code](demo-wasm/manipulate-dom)
- Demo 4 - Callback [Code](demo-wasm/callback)
