# WebAssembly 示例（Go 1.26）

这里的四个示例依次演示弹窗、操作 DOM、向 JavaScript 注册 Go 函数，以及异步回调。

## 构建与运行

进入任意示例目录，执行：

```bash
make
```

该命令会：

1. 使用 `GOOS=js GOARCH=wasm` 生成 `static/main.wasm`；
2. 从 Go 1.26 的 `$GOROOT/lib/wasm/wasm_exec.js` 复制浏览器运行时；
3. 使用仓库内置的 Go 静态文件服务器在 <http://localhost:9999> 启动示例。

如果不使用 Make，也可以手动执行：

```bash
mkdir -p static
GOOS=js GOARCH=wasm go build -o static/main.wasm main.go
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" static/wasm_exec.js
go run ../serve.go -port 9999 .
```

`js.FuncOf` 创建的函数必须保持可达，浏览器仍可能调用它时不能执行 `Release`。这些示例会一直运行，因此在 `main` 中保存函数值并使用 `select {}` 阻塞。异步示例还会先把 JavaScript 参数转换为普通 Go 值，再启动 goroutine，避免在回调返回后继续持有临时的参数切片。
