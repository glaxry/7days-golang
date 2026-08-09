# 现代 Go 迁移说明（2026）

本仓库最初以 Go 1.13 为基线，最后一次上游提交时间为 2021 年。本次升级以 Go 1.26 为语言基线，并在不破坏“每天一个阶段快照”的前提下，让每一天的代码、依赖和文章示例保持一致。

## 版本策略

- `go.mod` 使用 `go 1.26.0`，表示代码使用 Go 1.26 语言与标准库能力。
- CI 使用 `1.26.x`，自动选择 Go 1.26 的最新安全补丁版；升级时核对的当前版本是 Go 1.26.5。
- 没有在 45 个模块中重复写死 `toolchain go1.26.5`，避免读者仅运行一个小示例时被强制下载特定工具链。
- 每个 Day 继续保留独立 `go.mod`。这些快照存在重复的模块名，不能合并进同一个 `go.work`。

官方参考：[Go 1.26 Release Notes](https://go.dev/doc/go1.26)、[Go Release History](https://go.dev/doc/devel/release)。

## 语言和标准库写法

### `any` 代替空接口拼写

`any` 是 `interface{}` 的别名，运行时行为完全相同。新写法更短，也更容易看出这里表达的是“任意值”，不是一个带方法约束的业务接口。

### 使用 Go 1.26 现代化工具

所有模块都经过 Go 1.26 的 `go fix`。它把可安全替换的旧写法更新为当前标准库惯用法，例如：

- `reflect.Ptr` → `reflect.Pointer`；
- `reflect.TypeOf((*error)(nil)).Elem()` → `reflect.TypeFor[error]()`；
- 按下标遍历结构字段和方法 → `reflect.Type.Fields()` / `Methods()` 迭代器；
- 仅为写入响应而创建临时字符串 → `fmt.Appendf`；
- 简单计数循环 → 整数 `range`。

这些变化不是为了炫技：教程代码会直接展示当前 Go 文档中的 API，读者不必先学习一套已经过时的写法再迁移。

### 删除 `io/ioutil`

`io/ioutil` 从 Go 1.16 起已弃用。本仓库分别使用 `io.ReadAll` 和 `io.Discard`。官方说明见 [`io/ioutil` package](https://pkg.go.dev/io/ioutil)。

### 格式字符串必须明确

Go 1.26 的检查会拒绝 `fmt.Errorf(remoteMessage)` 这类把外部字符串直接当格式串的调用。本仓库改用 `errors.New(remoteMessage)`；需要格式化时则始终传入常量格式串。这既消除检查错误，也避免远端消息里的 `%` 被错误解释。

## GeeORM：纯 Go SQLite

旧教程使用 `github.com/mattn/go-sqlite3 v2.0.3+incompatible`，需要 CGO 和本机 C 编译器。本版改用 `modernc.org/sqlite v1.56.0`：

```go
import _ "modernc.org/sqlite"

db, err := sql.Open("sqlite", "gee.db")
```

注意驱动名也从 `sqlite3` 变为 `sqlite`。GeeORM 的方言注册名、测试和文章代码都已同步。

测试不再让根包与 `session` 子包并行共享同一个 `gee.db`：根包使用 `t.TempDir()`，session 测试使用进程内共享的内存数据库。这样 `go test ./...` 不会产生跨包锁冲突，也不会在工作区残留数据库文件。

数据库迁移还修复了一个旧实现依赖驱动宽松行为的问题：执行 `ALTER TABLE` 前必须关闭用于读取列信息的 `*sql.Rows`。打开的游标会持有 SQLite 读锁，严格实现会返回 `SQLITE_LOCKED`。

官方包说明：[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)。

## GeeCache：Protobuf API v2

旧代码使用已迁移的 `github.com/golang/protobuf` API，而且仓库缺少文中提到的 `.proto` 文件。本版：

- 使用 `google.golang.org/protobuf v1.36.11`；
- 补充 `geecachepb.proto`；
- 添加必须的 `go_package`；
- 使用 `protoc 35.0` 与 `protoc-gen-go v1.36.11` 重新生成代码。

复现生成过程：

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
protoc --go_out=. --go_opt=paths=source_relative geecachepb.proto
```

`paths=source_relative` 让生成文件和 `.proto` 保持在同一目录，适合这个小型教学模块。官方参考：[Go Generated Code Guide](https://protobuf.dev/reference/go/go-generated/)。

## GeeRPC：修复并发和协议边界

### 用明确帧边界保留 JSON 握手后的 Gob 请求

协议先发送 JSON `Option`，再发送 Gob 请求。旧实现用 `json.Decoder` 读取握手后直接丢弃 decoder；decoder 如果预读了紧随其后的 Gob 字节，这些字节也会被丢弃，服务端随后永久等待。

客户端使用 `json.Encoder.Encode` 发送以换行结尾的 Option。现在服务端以这条换行为明确帧边界，通过同一个 `bufio.Reader` 读取一行 JSON，再把 reader 交给 Gob codec。无论 TCP 是否合包，reader 预取的 Gob 数据都不会跨协议边界丢失；仓库还增加了将握手与首条请求合并写入的回归测试。

### 超时路径不再泄漏 goroutine

- 连接建立结果使用容量为 1 的 channel。调用方超时返回后，后台连接 goroutine 仍能投递结果并退出。
- 服务调用结果也使用容量为 1 的 channel。服务端超时后，业务 goroutine 完成时不会永久阻塞。
- 只有 `handleRequest` 的协调分支负责写响应，因此超时响应和迟到的业务响应不会重复写入同一连接。
- `time.NewTimer` 会显式 `Stop`，清楚表达定时器的生命周期。

RPC 测试会修改包级默认服务并依赖真实网络时序，因此改为串行执行，避免并行测试相互污染全局状态。

## GeeBolt：补全 mmap 草稿

上游 `day2-mmap` 只有一个没有参数、没有返回值且无法编译的 `syscall.Mmap()` 占位调用。本版使用 `golang.org/x/sys v0.47.0` 补全：

- Unix 使用 `unix.Mmap` / `Msync` / `Munmap`；
- Windows 使用 `CreateFileMapping` / `MapViewOfFile` / `FlushViewOfFile`；
- `Open`、`Sync`、`Close` 都返回可追踪的错误；
- 测试覆盖首次创建、写入持久化和重新打开。

这部分仍是数据库存储结构实验，不是完整 BoltDB 实现。

## 验证

单独学习某一天时，在该目录运行：

```bash
go test ./...
```

验证全部 45 个模块和 4 个 WebAssembly 示例：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-all.ps1
```

```bash
bash scripts/test-all.sh
```

GitHub Actions 同时覆盖 Windows 与 Linux。

## 教学边界

这些项目用于解释框架核心机制，不应直接当作生产库。特别是：GeeWeb 没有完整的安全响应头和优雅停机；GeeCache 没有节点认证、传输加密和响应大小限制；GeeORM 没有完整的标识符转义、参数校验和关联模型；GeeRPC 没有身份认证、TLS、真正取消远端执行或协议版本协商。教程中会指出这些边界，避免把“能说明原理”误解为“已具备生产保证”。
