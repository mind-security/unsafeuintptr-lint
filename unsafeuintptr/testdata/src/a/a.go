// Package a exercises same-package cases: unsafe conversions, the safe
// pointer-arithmetic pattern, and calls to a locally //go:uintptrescapes-marked
// function versus an ordinary one.
package a

import "unsafe"

//go:nosplit
//go:uintptrescapes
func keepAlive(...uintptr) {}

func unmarked(uintptr) {}

func stored(p *int) {
	x := uintptr(unsafe.Pointer(p)) // want `not kept alive`
	_ = x
}

func returned(p *int) uintptr {
	return uintptr(unsafe.Pointer(p)) // want `not kept alive`
}

type holder struct{ addr uintptr }

func inStruct(p *int) holder {
	return holder{addr: uintptr(unsafe.Pointer(p))} // want `not kept alive`
}

func inSlice(p *int) []uintptr {
	return []uintptr{uintptr(unsafe.Pointer(p))} // want `not kept alive`
}

// viaVariable stores the unsafe.Pointer first; detection is by type, not syntax.
func viaVariable(p *int) {
	u := unsafe.Pointer(p)
	x := uintptr(u) // want `not kept alive`
	_ = x
}

func safeArith(p *int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 8)
}

func safeKeepAliveCall(p *int) {
	keepAlive(uintptr(unsafe.Pointer(p)))
}

func unsafeUnmarkedCall(p *int) {
	unmarked(uintptr(unsafe.Pointer(p))) // want `not kept alive`
}

// The following exercise //nolint suppression: no diagnostic is expected.

func suppressedTrailingNamed(p *int) {
	x := uintptr(unsafe.Pointer(p)) //nolint:unsafeuintptr
	_ = x
}

func suppressedTrailingBare(p *int) {
	x := uintptr(unsafe.Pointer(p)) //nolint
	_ = x
}

func suppressedStandaloneAbove(p *int) uintptr {
	//nolint:unsafeuintptr
	return uintptr(unsafe.Pointer(p))
}

// A //nolint for a different linter must NOT suppress this one.
func notSuppressedOtherLinter(p *int) {
	//nolint:otherlinter
	z := uintptr(unsafe.Pointer(p)) // want `not kept alive`
	_ = z
}
