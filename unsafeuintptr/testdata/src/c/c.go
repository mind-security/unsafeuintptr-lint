// Package c exercises method-receiver resolution for the dynamic-load Call
// idiom, recognized via the -allow flag set by the test.
package c

import (
	"unsafe"
	"winstub"
)

func inlineChain(p *int) {
	winstub.NewLazyDLL("k").NewProc("f").Call(uintptr(unsafe.Pointer(p)))
}

// splitChain holds the proc in a variable to prove the receiver still resolves.
func splitChain(p *int) {
	d := winstub.NewLazyDLL("k")
	pr := d.NewProc("f")
	pr.Call(uintptr(unsafe.Pointer(p)))
}

func notAllowlisted(p *int) {
	d := winstub.NewLazyDLL("k")
	d.NotKeptAlive(uintptr(unsafe.Pointer(p))) // want `not kept alive`
}
