// Package b exercises cross-package callees: the built-in syscall allowlist
// (safe) and an ordinary imported function (unsafe).
package b

import (
	"runtime"
	"somepkg"
	"syscall"
	"unsafe"
)

func safeSyscall(trap uintptr, buf []byte) {
	syscall.Syscall(trap, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
}

func unsafeOrdinary(p *int) {
	somepkg.Ordinary(uintptr(unsafe.Pointer(p))) // want `not kept alive`
}

// realWorldBug mirrors mind-security/endpoint#2889: a pointer is converted to
// uintptr and handed to an ordinary (non-keep-alive) cross-package wrapper, which
// only later forwards it to a syscall. The pointer crosses the wrapper boundary
// as a bare integer, so the object may be freed or moved first. The trailing
// runtime.KeepAlive does not make this safe and must not suppress the report.
func realWorldBug() {
	callbackInfo := [2]uintptr{0, 0}
	somepkg.Ordinary(uintptr(unsafe.Pointer(&callbackInfo))) // want `not kept alive`
	runtime.KeepAlive(&callbackInfo)
}
