// Package b exercises cross-package callees: the built-in syscall allowlist
// (safe) and an ordinary imported function (unsafe).
package b

import (
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
