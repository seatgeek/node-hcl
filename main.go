/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
// + build js,wasm
package main

import (
	"fmt"
	"syscall/js"

	"github.com/seatgeek/node-hcl/hcl"
)

var jsGlobal js.Value
var jsRoot js.Value

const (
	bridgeJavaScriptName = "__node_hcl_wasm__"
)

func registerFn(name string, callback func(this js.Value, args []js.Value) (interface{}, error)) {
	jsRoot.Set(name, js.FuncOf(registrationWrapper(callback)))
}

func registrationWrapper(fn func(this js.Value, args []js.Value) (interface{}, error)) func(this js.Value, args []js.Value) interface{} {
	return func(this js.Value, args []js.Value) interface{} {
		cb := args[len(args)-1]

		ret, err := fn(this, args[:len(args)-1])

		if err != nil {
			cb.Invoke(err.Error(), js.Null())
		} else {
			cb.Invoke(js.Null(), ret)
		}

		return ret
	}
}

func main() {
	jsGlobal = js.Global().Get("global")
	jsRoot = jsGlobal.Get(bridgeJavaScriptName)
	c := make(chan struct{}, 0)

	registerFn("merge", func(this js.Value, args []js.Value) (interface{}, error) {
		argCount := len(args)
		if argCount < 2 || argCount > 3 {
			return nil, fmt.Errorf("Invalid number of arguments, expected (2 - 3)")
		}

		if args[0].Type() != js.TypeString {
			return nil, fmt.Errorf("Invalid first argument type, expected string")
		}

		if args[1].Type() != js.TypeString {
			return nil, fmt.Errorf("Invalid second argument type, expected string")
		}

		options := &hcl.MergeOptions{}

		if argCount == 3 {
			arg2Type := args[2].Type()
			if arg2Type != js.TypeObject && arg2Type != js.TypeUndefined {
				return nil, fmt.Errorf("Invalid third argument type, expected optional object")
			}

			if arg2Type == js.TypeObject {
				options.MergeMapKeys = args[2].Get("mergeMapKeys").Bool()
			}
		}

		aHclString := args[0].String()
		bHclString := args[1].String()

		hclmerger := hcl.NewMerger(options)
		return hclmerger.Merge(aHclString, bHclString)
	})

	<-c
}
