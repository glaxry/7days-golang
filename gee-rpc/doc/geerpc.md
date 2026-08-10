---
title: 7天用Go从零实现RPC框架GeeRPC
date: 2020-10-06 16:00:00
description: 7天用 Go语言/golang 从零实现 RPC 框架 GeeRPC 教程(7 days implement golang remote procedure call framework from scratch tutorial)，动手写 RPC 框架，参照 golang 标准库 net/rpc 的实现，实现了服务端(server)、支持异步和并发的客户端(client)、消息编码与解码(message encoding and decoding)、服务注册(service register)、支持 TCP/Unix/HTTP 等多种传输协议。并在此基础上新增了协议交换(protocol exchange)、注册中心(registry)、服务发现(service discovery)、负载均衡(load balance)、超时处理(timeout processing)等特性。
tags:
- Go
nav: 从零实现
categories:
- RPC框架 - GeeRPC
keywords:
- Go语言
- 从零实现RPC框架
- 动手写RPC框架
- 服务注册与发现
- 负载均衡
image: post/geerpc/geerpc.jpg
github: https://github.com/glaxry/7days-golang
book: 七天用Go从零实现系列
book_title: Day0 序言
---

![golang RPC framework](geerpc/geerpc.jpg)

## 1 谈谈 RPC 框架

RPC(Remote Procedure Call，远程过程调用)是一种计算机通信协议，允许调用不同进程空间的程序。RPC 的客户端和服务器可以在一台机器上，也可以在不同的机器上。程序员使用时，就像调用本地程序一样，无需关注内部的实现细节。

不同的应用程序之间的通信方式有很多，比如浏览器和服务器之间广泛使用的基于 HTTP 协议的 Restful API。与 RPC 相比，Restful API 有相对统一的标准，因而更通用，兼容性更好，支持不同的语言。HTTP 协议是基于文本的，一般具备更好的可读性。但是缺点也很明显：

- Restful 接口需要额外的定义，无论是客户端还是服务端，都需要额外的代码来处理，而 RPC 调用则更接近于直接调用。
- 基于 HTTP 协议的 Restful 报文冗余，承载了过多的无效信息，而 RPC 通常使用自定义的协议格式，减少冗余报文。
- RPC 可以采用更高效的序列化协议，将文本转为二进制传输，获得更高的性能。
- 因为 RPC 的灵活性，所以更容易扩展和集成诸如注册中心、负载均衡等功能。

## 2 RPC 框架需要解决什么问题

RPC 框架需要解决什么问题？或者我们换一个问题，为什么需要 RPC 框架？

我们可以想象下两台机器上，两个应用程序之间需要通信，那么首先，需要确定采用的传输协议是什么？如果这个两个应用程序位于不同的机器，那么一般会选择 TCP 协议或者 HTTP 协议；那如果两个应用程序位于相同的机器，也可以选择 Unix Socket 协议。传输协议确定之后，还需要确定报文的编码格式，比如采用最常用的 JSON 或者 XML，那如果报文比较大，还可能会选择 protobuf 等其他的编码方式，甚至编码之后，再进行压缩。接收端获取报文则需要相反的过程，先解压再解码。

解决了传输协议和报文编码的问题，接下来还需要解决一系列的可用性问题，例如，连接超时了怎么办？是否支持异步请求和并发？

如果服务端的实例很多，客户端并不关心这些实例的地址和部署位置，只关心自己能否获取到期待的结果，那就引出了注册中心(registry)和负载均衡(load balance)的问题。简单地说，即客户端和服务端互相不感知对方的存在，服务端启动时将自己注册到注册中心，客户端调用时，从注册中心获取到所有可用的实例，选择一个来调用。这样服务端和客户端只需要感知注册中心的存在就够了。注册中心通常还需要实现服务动态添加、删除，使用心跳确保服务处于可用状态等功能。

再进一步，假设服务端是不同的团队提供的，如果没有统一的 RPC 框架，各个团队的服务提供方就需要各自实现一套消息编解码、连接池、收发线程、超时处理等“业务之外”的重复技术劳动，造成整体的低效。因此，“业务之外”的这部分公共的能力，即是 RPC 框架所需要具备的能力。

## 3 关于 GeeRPC

Go 语言广泛地应用于云计算和微服务，成熟的 RPC 框架和微服务框架汗牛充栋。`grpc`、`rpcx`、`go-micro` 等都是非常成熟的框架。一般而言，RPC 是微服务框架的一个子集，微服务框架可以自己实现 RPC 部分，当然，也可以选择不同的 RPC 框架作为通信基座。

考虑性能和功能，上述成熟的框架代码量都比较庞大，而且通常和第三方库，例如 `protobuf`、`etcd`、`zookeeper` 等有比较深的耦合，难以直观地窥视框架的本质。GeeRPC 的目的是以最少的代码，实现 RPC 框架中最为重要的部分，帮助大家理解 RPC 框架在设计时需要考虑什么。代码简洁是第一位的，功能是第二位的。

因此，GeeRPC 选择从零实现 Go 语言官方的标准库 `net/rpc`，并在此基础上，新增了协议交换(protocol exchange)、注册中心(registry)、服务发现(service discovery)、负载均衡(load balance)、超时处理(timeout processing)等特性。分七天完成，最终代码约 1000 行。

## 4 目录

- 第一天 - [服务端与消息编码](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc-day1.md) | [Code](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day1-codec)
- 第二天 - [支持并发与异步的客户端](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc-day2.md) | [Code](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day2-client)
- 第三天 - [服务注册(service register)](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc-day3.md) | [Code](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day3-service )
- 第四天 - [超时处理(timeout)](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc-day4.md) | [Code](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day4-timeout )
- 第五天 - [支持HTTP协议](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc-day5.md) | [Code](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day5-http-debug)
- 第六天 - [负载均衡(load balance)](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc-day6.md) | [Code](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day6-load-balance)
- 第七天 - [服务发现与注册中心(registry)](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc-day7.md) | [Code](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day7-registry)

## 附 推荐阅读

- [Go 语言简明教程](https://geektutu.com/post/quick-golang.html)
- [Go 语言笔试面试题](https://geektutu.com/post/qa-golang.html)

## 通俗学习地图：RPC 是把网络调用包装成方法调用

RPC 想提供的体验是 `client.Call("Foo.Sum", args, &reply)`，但网络另一端并不能直接看到 Go 函数。框架必须把“想调用谁、参数是什么”编码成字节发出去，服务端再解码、找到方法、执行并把结果编码回来。

```text
客户端方法调用
    ↓ 分配序号、登记 pending
Header{ServiceMethod, Seq} + Args
    ↓ Codec 编码并通过 TCP/HTTP 发送
服务端解码 → 反射找到服务和方法 → 执行
    ↓
Header{Seq, Error} + Reply
    ↓ 客户端接收循环按 Seq 找回原调用
同步等待返回 / 异步通知完成
```

七天分别补齐这条链的关键部分：

1. 第一天定义消息头与 Codec，打通 TCP 服务端的读取、调用和响应。
2. 第二天实现并发客户端，用序号把乱序返回的响应交还给正确调用者。
3. 第三天通过反射注册服务，把 `Service.Method` 映射到真正的 Go 方法。
4. 第四天分别控制连接、服务端处理和客户端调用的超时。
5. 第五天用 HTTP CONNECT 建立 RPC 隧道，并提供服务调试页面。
6. 第六天加入服务发现、随机/轮询选择、客户端连接复用与广播调用。
7. 第七天实现一个基于 HTTP 的注册中心和心跳，让节点列表能够动态更新。

学习 RPC 时最好分清四层：传输层负责字节怎样到达；Codec 负责字节与消息怎样转换；Client/Server 负责并发请求的生命周期；Service/Registry 负责“调用谁”和“去哪里找”。超时、错误和关闭信号必须跨越这些边界正确传播。

GeeRPC 是为了讲透核心机制而缩小的实现。生产系统还需要协议版本协商、身份认证、TLS、取消传播、流控、消息大小限制、健康检查、熔断、重试幂等、可观测性和更成熟的注册中心。尤其不要把“能通过网络调用”误解为“网络与本地函数一样可靠”。
