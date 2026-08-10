---
title: 动手写RPC框架 - GeeRPC第四天 超时处理(timeout)
date: 2020-10-07 23:00:00
description: 7天用 Go语言/golang 从零实现 RPC 框架 GeeRPC 教程(7 days implement golang remote procedure call framework from scratch tutorial)，动手写 RPC 框架，参照 golang 标准库 net/rpc 的实现，实现了服务端(server)、支持异步和并发的客户端(client)、消息编码与解码(message encoding and decoding)、服务注册(service register)、支持 TCP/Unix/HTTP 等多种传输协议。第四天为RPC框架提供了处理超时的能力(timeout processing)。
tags:
- Go
nav: 从零实现
categories:
- RPC框架 - GeeRPC
keywords:
- Go语言
- 从零实现RPC框架
- 连接超时
image: post/geerpc/geerpc.jpg
github: https://github.com/glaxry/7days-golang
book: 七天用Go从零实现系列
book_title: Day4 超时处理
---

![golang RPC framework](geerpc/geerpc.jpg)

本文是[7天用Go从零实现RPC框架GeeRPC](https://github.com/glaxry/7days-golang/blob/main/gee-rpc/doc/geerpc.md)的第四篇。

- 增加连接超时的处理机制
- 增加服务端处理超时的处理机制，代码约 100 行

## 为什么需要超时处理机制

超时处理是 RPC 框架一个比较基本的能力，如果缺少超时处理机制，无论是服务端还是客户端都容易因为网络或其他错误导致挂死，资源耗尽，这些问题的出现大大地降低了服务的可用性。因此，我们需要在 RPC 框架中加入超时处理的能力。

纵观整个远程调用的过程，需要客户端处理超时的地方有：

- 与服务端建立连接，导致的超时
- 发送请求到服务端，写报文导致的超时
- 等待服务端处理时，等待处理导致的超时（比如服务端已挂死，迟迟不响应）
- 从服务端接收响应时，读报文导致的超时

需要服务端处理超时的地方有：

- 读取客户端请求报文时，读报文导致的超时
- 发送响应报文时，写报文导致的超时
- 调用映射服务的方法时，处理报文导致的超时


GeeRPC 在 3 个地方添加了超时处理机制。分别是：

1）客户端创建连接时
2）客户端 `Client.Call()` 整个过程导致的超时（包含发送报文，等待处理，接收报文所有阶段）
3）服务端处理报文，即 `Server.handleRequest` 超时。

## 创建连接超时

为了实现上的简单，将超时设定放在了 Option 中。ConnectTimeout 默认值为 10s，HandleTimeout 默认值为 0，即不设限。

```go
type Option struct {
	MagicNumber    int           // MagicNumber marks this's a geerpc request
	CodecType      codec.Type    // client may choose different Codec to encode body
	ConnectTimeout time.Duration // 0 means no limit
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      codec.GobType,
	ConnectTimeout: time.Second * 10,
}
```

客户端连接超时，只需要为 Dial 添加一层超时处理的外壳即可。

[day4-timeout/client.go](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day4-timeout)

```go
type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)

func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	// close the connection if client is nil
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult, 1)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	timer := time.NewTimer(opt.ConnectTimeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}

// Dial connects to an RPC server at the specified network address
func Dial(network, address string, opts ...*Option) (*Client, error) {
	return dialTimeout(NewClient, network, address, opts...)
}
```

在这里实现了一个超时处理的外壳 `dialTimeout`，这个壳将 NewClient 作为入参，在 2 个地方添加了超时处理的机制。

1) 将 `net.Dial` 替换为 `net.DialTimeout`，如果连接创建超时，将返回错误。
2）使用子协程执行 NewClient，执行完成后则通过信道 ch 发送结果，如果 `time.After()` 信道先接收到消息，则说明 NewClient 执行超时，返回错误。

## Client.Call 超时

`Client.Call` 的超时处理机制，使用 context 包实现，控制权交给用户，控制更为灵活。

```go
// Call invokes the named function, waits for it to complete,
// and returns its error status.
func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply any) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}
```

用户可以使用 `context.WithTimeout` 创建具备超时检测能力的 context 对象来控制。例如：

```go
ctx, _ := context.WithTimeout(context.Background(), time.Second)
var reply int
err := client.Call(ctx, "Foo.Sum", &Args{1, 2}, &reply)
...
```

## 服务端处理超时

这一部分的实现与客户端很接近，使用可停止的 `time.Timer` 结合 `select` 和带缓冲 channel 完成。

[day4-timeout/server.go](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day4-timeout)

```go
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	send := func(err error) {
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
	}

	if timeout == 0 {
		send(req.svc.call(req.mtype, req.argv, req.replyv))
		return
	}
	called := make(chan error, 1)
	go func() {
		called <- req.svc.call(req.mtype, req.argv, req.replyv)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendResponse(cc, req.h, invalidRequest, sending)
	case err := <-called:
		send(err)
	}
}
```

这里有两个容易忽略的并发细节：

1. `called` 必须有容量 1。超时分支返回后，业务函数可能仍在执行；它结束时可以写入 channel 并退出，不会泄漏 goroutine。
2. 业务 goroutine 只返回 `error`，不直接写网络。最终只有 `handleRequest` 选择出的一个分支调用 `sendResponse`，因此不会先发超时响应、稍后又发成功响应。

`time.NewTimer` 配合 `Stop` 明确了定时器生命周期；`timeout == 0` 时直接同步调用，避免创建无意义的 goroutine。

## 测试用例

第一个测试用例，用于测试连接超时。NewClient 函数耗时 2s，ConnectionTimeout 分别设置为 1s 和 0 两种场景。

[day4-timeout/client_test.go](https://github.com/glaxry/7days-golang/tree/main/gee-rpc/day4-timeout)

```go
func TestClient_dialTimeout(t *testing.T) {
	l, _ := net.Listen("tcp", ":0")

	f := func(conn net.Conn, opt *Option) (client *Client, err error) {
		_ = conn.Close()
		time.Sleep(time.Second * 2)
		return nil, nil
	}
	t.Run("timeout", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", l.Addr().String(), &Option{ConnectTimeout: time.Second})
		_assert(err != nil && strings.Contains(err.Error(), "connect timeout"), "expect a timeout error")
	})
	t.Run("0", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", l.Addr().String(), &Option{ConnectTimeout: 0})
		_assert(err == nil, "0 means no limit")
	})
}
```

第二个测试用例，用于测试处理超时。`Bar.Timeout` 耗时 2s，场景一：客户端设置超时时间为 1s，服务端无限制；场景二，服务端设置超时时间为1s，客户端无限制。

```go
type Bar int

func (b Bar) Timeout(argv int, reply *int) error {
	time.Sleep(time.Second * 2)
	return nil
}

func startServer(addr chan string) {
	var b Bar
	_ = Register(&b)
	// pick a free port
	l, _ := net.Listen("tcp", ":0")
	addr <- l.Addr().String()
	Accept(l)
}

func TestClient_Call(t *testing.T) {
	addrCh := make(chan string)
	go startServer(addrCh)
	addr := <-addrCh
	time.Sleep(time.Second)
	t.Run("client timeout", func(t *testing.T) {
		client, _ := Dial("tcp", addr)
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		var reply int
		err := client.Call(ctx, "Bar.Timeout", 1, &reply)
		_assert(err != nil && strings.Contains(err.Error(), ctx.Err().Error()), "expect a timeout error")
	})
	t.Run("server handle timeout", func(t *testing.T) {
		client, _ := Dial("tcp", addr, &Option{
			HandleTimeout: time.Second,
		})
		var reply int
		err := client.Call(context.Background(), "Bar.Timeout", 1, &reply)
		_assert(err != nil && strings.Contains(err.Error(), "handle timeout"), "expect a timeout error")
	})
}
```

## 附 推荐阅读

- [Go 语言简明教程](https://geektutu.com/post/quick-golang.html)
- [Go 语言笔试面试题](https://geektutu.com/post/qa-golang.html)

## 白话复盘：三个超时保护的是三个不同阶段

### 不要只说“给 RPC 加个超时”

一次 RPC 至少经历建立连接、服务端处理、客户端等待三个阶段，它们的失败含义和控制位置不同：

- 连接超时：在规定时间内无法完成 Dial，客户端应放弃这个地址。
- 服务端处理超时：方法执行太久，服务端应及时返回超时响应，避免客户端无限等待。
- 客户端调用超时：调用者自己的等待预算耗尽，即使服务端仍在运行，也要先返回给上层。

这些时间可以不同。例如连接预算 1 秒，调用总预算 3 秒，服务端方法上限 2 秒。把它们混成一个值会让排错和调优都很困难。

### 服务端怎样避免重复响应

处理方法和计时器可能几乎同时完成。如果两个 goroutine 都直接写响应，就会出现一条请求返回两次、字节流损坏。正确做法是让它们只把结果发送给协调者，由一个位置通过 `select` 决定本次请求唯一的响应。

完成通知通道使用容量 1，能避免超时分支已经返回后，业务 goroutine 在发送完成信号时永久阻塞。但要注意：返回超时并不会自动停止正在执行的 Go 方法，它仍可能继续消耗资源或产生副作用。

### 客户端 `context` 的真实边界

客户端可以 `select` 等待 `ctx.Done()` 或 Call 完成。Context 到期能让调用者停止等候，但若协议没有取消帧，服务端并不知道客户端已放弃。完整取消需要把取消信号传过网络，并要求业务方法配合检查。

### 容易踩坑

- `time.After` 在热点路径会创建定时器；需要主动停止或复用时应使用 `time.NewTimer` 并正确 Stop。
- 超时后盲目重试非幂等操作，可能让“第一次其实成功”的写操作执行两遍。
- 测试不应依赖非常接近的毫秒边界，否则慢机器上容易偶发失败；给预期留出合理余量。
- 超时错误要标明发生阶段，不能都只返回模糊的 `timeout`。

### 动手检查

分别制造“端口不接受连接”“方法主动 sleep”“客户端 context 更短”三种情况，记录错误来源和服务端是否仍继续执行。能区分这三种现象，才算真正理解超时边界。
