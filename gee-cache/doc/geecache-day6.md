---
title: 动手写分布式缓存 - GeeCache第六天 防止缓存击穿
date: 2020-02-16 23:00:00
description: 7天用 Go语言/golang 从零实现分布式缓存 GeeCache 教程(7 days implement golang distributed cache from scratch tutorial)，动手写分布式缓存，参照 groupcache 的实现。本文介绍了缓存雪崩、缓存击穿与缓存穿透的概念，使用 singleflight 防止缓存击穿，实现与测试。
tags:
- Go
nav: 从零实现
categories:
- 分布式缓存 - GeeCache
keywords:
- Go语言
- 从零实现
- HTTP客户端
- 分布式节点
image: post/geecache-day6/singleflight_logo.jpg
github: https://github.com/glaxry/7days-golang
book: 七天用Go从零实现系列
book_title: Day6 防止缓存击穿
---

![geecache single flight](geecache-day6/singleflight.jpg)

本文是[7天用Go从零实现分布式缓存GeeCache](https://github.com/glaxry/7days-golang/blob/main/gee-cache/doc/geecache.md)的第六篇。

- 缓存雪崩、缓存击穿与缓存穿透的概念简介。
- 使用 singleflight 防止缓存击穿，实现与测试。**代码约70行**

## 1 缓存雪崩、缓存击穿与缓存穿透

[GeeCache 第五天](https://github.com/glaxry/7days-golang/blob/main/gee-cache/doc/geecache-day5.md) 提到了缓存雪崩和缓存击穿，在这里做下总结：

> **缓存雪崩**：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。

> **缓存击穿**：一个存在的key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时DB请求量大、压力骤增。

> **缓存穿透**：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。

## 2 singleflight 的实现

还记得 [GeeCache 第五天](https://github.com/glaxry/7days-golang/blob/main/gee-cache/doc/geecache-day5.md) 最后的测试结果吗？

```bash
2020/02/16 21:17:45 [Server http://localhost:8003] Pick peer http://localhost:8001
2020/02/16 21:17:45 [Server http://localhost:8003] Pick peer http://localhost:8001
2020/02/16 21:17:45 [Server http://localhost:8003] Pick peer http://localhost:8001
```

我们并发了 N 个请求 `?key=Tom`，8003 节点向 8001 同时发起了 N 次请求。假设对数据库的访问没有做任何限制的，很可能向数据库也发起 N 次请求，容易导致缓存击穿和穿透。即使对数据库做了防护，HTTP 请求是非常耗费资源的操作，针对相同的 key，8003 节点向 8001 发起三次请求也是没有必要的。那这种情况下，我们如何做到只向远端节点发起一次请求呢？

geecache 实现了一个名为 singleflight 的 package 来解决这个问题。

[day6-single-flight/geecache/singleflight/singleflight.go - github](https://github.com/glaxry/7days-golang/tree/main/gee-cache/day6-single-flight/geecache/singleflight)

首先创建 `call` 和 `Group` 类型。

```go
package singleflight

import "sync"

type call struct {
	wg  sync.WaitGroup
	val any
	err error
}

type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call
}
```

- `call` 代表正在进行中，或已经结束的请求。使用 `sync.WaitGroup` 锁避免重入。
- `Group` 是 singleflight 的主数据结构，管理不同 key 的请求(call)。

实现 `Do` 方法

```go
func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
```

- Do 方法，接收 2 个参数，第一个参数是 `key`，第二个参数是一个函数 `fn`。Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 `fn` 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。

`g.mu` 是保护 Group 的成员变量 `m` 不被并发读写而加上的锁。为了便于理解 `Do` 函数，我们将 `g.mu` 暂时去掉。并且把 `g.m` 延迟初始化的部分去掉，延迟初始化的目的很简单，提高内存使用效率。

剩下的逻辑就很清晰了：

```go
func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	if c, ok := g.m[key]; ok {
		c.wg.Wait()   // 如果请求正在进行中，则等待
		return c.val, c.err  // 请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)       // 发起请求前加锁
	g.m[key] = c      // 添加到 g.m，表明 key 已经有对应的请求在处理

	c.val, c.err = fn() // 调用 fn，发起请求
	c.wg.Done()         // 请求结束

    delete(g.m, key)    // 更新 g.m
    
	return c.val, c.err // 返回结果
}
```

并发协程之间不需要消息传递，非常适合 `sync.WaitGroup`。

- wg.Add(1) 锁加1。
- wg.Wait() 阻塞，直到锁被释放。
- wg.Done() 锁减1。

## 3 singleflight 的使用

[day6-single-flight/geecache/geecache.go - github](https://github.com/glaxry/7days-golang/tree/main/gee-cache/day6-single-flight/geecache)

```go
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
    // ...
	g := &Group{
        // ...
		loader:    &singleflight.Group{},
	}
	return g
}

func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	viewi, err := g.loader.Do(key, func() (any, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}
```

- 修改 `geecache.go` 中的 `Group`，添加成员变量 loader，并更新构建函数 `NewGroup`。
- 修改 `load` 函数，将原来的 load 的逻辑，使用 `g.loader.Do` 包裹起来即可，这样确保了并发场景下针对相同的 key，`load` 过程只会调用一次。

## 4 测试

执行 `run.sh` 就可以看到效果了。

```bash
$ ./run.sh
2020/02/16 22:36:00 [Server http://localhost:8003] Pick peer http://localhost:8001
2020/02/16 22:36:00 [Server http://localhost:8001] GET /_geecache/scores/Tom
2020/02/16 22:36:00 [SlowDB] search key Tom
630630630
```

可以看到，向 API 发起了三次并发请求，但8003 只向 8001 发起了一次请求，就搞定了。

如果并发度不够高，可能仍会看到向 8001 请求三次的场景。这种情况下三次请求是串行执行的，并没有触发 `singleflight` 的锁机制工作，可以加大并发数量再测试。即，将 `run.sh` 中的 `curl` 命令复制 N 次。

## 附 推荐

- [Go 语言简明教程#并发编程](https://geektutu.com/post/quick-golang.html#7-%E5%B9%B6%E5%8F%91%E7%BC%96%E7%A8%8B-goroutine)
- [Go Test 单元测试简明教程](https://geektutu.com/post/quick-go-test.html)

## 白话复盘：singleflight 合并的是“正在进行的同一件事”

### 缓存击穿是怎样发生的

一个热门 key 刚好不在缓存中时，可能有几百个请求同时到达。它们几乎同时发现 miss，又几乎同时查询数据库。即使第一个查询很快会把结果写入缓存，其他查询已经发出，源站仍会遭受突发压力。

singleflight 的思路是：同一进程内，同一个 key 在同一时刻只允许一个执行者真正调用加载函数；后来者不重复工作，而是在旁边等待。执行者完成后，所有等待者共享同一个结果和错误。

### `Group.Do` 怎样协调

1. 用互斥锁检查 `map[key]*call`。
2. 如果已有 call，说明有人正在加载，释放锁后等待它的 `WaitGroup`。
3. 如果没有，就创建 call、把计数设为 1 并放入 map，然后释放锁执行真正的函数。
4. 执行完成后保存结果，唤醒等待者，再从 map 删除该 call，让未来的新请求可以重新执行。

锁只用于保护 call 表，绝不能覆盖慢速加载过程；否则不同 key 也会被无谓地串行化。`WaitGroup` 则只让相同 key 的跟随者等待对应任务。

### 能解决什么，不能解决什么

- 它减少并发重复回源，但不会让失败结果自动重试；同一批等待者会一起收到相同错误。
- 它只在当前进程内生效。三个缓存节点同时 miss，最多仍可能各自回源一次；跨节点合并需要额外的分布式协调。
- 执行者非常慢时，所有跟随者都会一起等待，因此仍需要超时、取消和源站保护。
- singleflight 不是普通结果缓存。call 完成后会从表中删除，长期复用结果仍由 LRU 负责。

### 动手检查

让加载函数休眠并递增计数，启动十个 goroutine 同时对同一个 key 调用 `Do`。最终计数应为 1，十个调用结果相同；再同时请求两个不同 key，确认它们能并行执行而不是互相阻塞。
