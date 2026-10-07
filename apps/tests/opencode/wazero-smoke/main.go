// Execute original Wazero bytecode and reflection-based Go host callbacks.
package main

import (
	"context"
	"github.com/tetratelabs/wazero"
	"kos"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode Wazero runtime test")
	if !ok {
		panic("console")
	}
	defer console.Close()
	kos.DebugString("OPENCODE_WAZERO_START")
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	// (module (func (export "answer") (result i32) i32.const 42))
	binary := []byte{
		0, 0x61, 0x73, 0x6d, 1, 0, 0, 0,
		1, 5, 1, 0x60, 0, 1, 0x7f,
		3, 2, 1, 0,
		7, 10, 1, 6, 'a', 'n', 's', 'w', 'e', 'r', 0, 0,
		10, 6, 1, 4, 0, 0x41, 42, 0x0b,
	}
	module, err := r.Instantiate(ctx, binary)
	if err != nil {
		panic(err)
	}
	result, err := module.ExportedFunction("answer").Call(ctx)
	if err != nil || len(result) != 1 || result[0] != 42 {
		panic("bytecode execution")
	}
	kos.DebugString("WAZERO_BYTECODE_OK")
	host, err := r.NewHostModuleBuilder("native").NewFunctionBuilder().
		WithFunc(func(value uint32) uint32 { return value + 1 }).Export("inc").Instantiate(ctx)
	if err != nil {
		panic(err)
	}
	defer host.Close(ctx)
	// (module (import "native" "inc" (func (param i32) (result i32)))
	//   (func (export "run") (param i32) (result i32) local.get 0 call 0))
	caller := []byte{
		0, 0x61, 0x73, 0x6d, 1, 0, 0, 0,
		1, 6, 1, 0x60, 1, 0x7f, 1, 0x7f,
		2, 14, 1, 6, 'n', 'a', 't', 'i', 'v', 'e', 3, 'i', 'n', 'c', 0, 0,
		3, 2, 1, 0,
		7, 7, 1, 3, 'r', 'u', 'n', 0, 1,
		10, 8, 1, 6, 0, 0x20, 0, 0x10, 0, 0x0b,
	}
	guest, err := r.Instantiate(ctx, caller)
	if err != nil {
		panic(err)
	}
	result, err = guest.ExportedFunction("run").Call(ctx, 41)
	if err != nil || len(result) != 1 || result[0] != 42 {
		panic("reflected host callback")
	}
	kos.DebugString("OPENCODE_WAZERO_PASS")
	console.WriteString("OPENCODE_WAZERO_PASS\n")
}
