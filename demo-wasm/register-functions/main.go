// main.go
package main

import "syscall/js"

func fib(i int) int {
	if i == 0 || i == 1 {
		return 1
	}
	return fib(i-1) + fib(i-2)
}

func fibFunc(this js.Value, args []js.Value) any {
	return js.ValueOf(fib(args[0].Int()))
}

func main() {
	callback := js.FuncOf(fibFunc)
	js.Global().Set("fibFunc", callback)
	select {}
}
