// main.go
package main

import (
	"syscall/js"
	"time"
)

func fib(i int) int {
	if i == 0 || i == 1 {
		return 1
	}
	return fib(i-1) + fib(i-2)
}

func fibFunc(this js.Value, args []js.Value) any {
	n := args[0].Int()
	callback := args[len(args)-1]
	go func() {
		time.Sleep(3 * time.Second)
		v := fib(n)
		callback.Invoke(v)
	}()

	js.Global().Get("ans").Set("innerHTML", "Waiting 3s...")
	return nil
}

func main() {
	callback := js.FuncOf(fibFunc)
	js.Global().Set("fibFunc", callback)
	select {}
}
