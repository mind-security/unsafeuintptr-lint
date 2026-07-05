// Package winstub mimics the shape of the Windows dynamic-load API
// (syscall.NewLazyDLL(...).NewProc(...).Call(...)) so the cross-package
// method-resolution path can be tested on any platform. Its Call methods stand
// in for the real //go:uintptrescapes-marked syscall.LazyProc.Call /
// syscall.Proc.Call; the test recognizes them via the -allow flag, since
// keep-alive pragmas are not visible across package boundaries.
package winstub

type LazyDLL struct{ name string }

func NewLazyDLL(name string) *LazyDLL { return &LazyDLL{name} }

type LazyProc struct{ name string }

func (d *LazyDLL) NewProc(name string) *LazyProc { return &LazyProc{name} }

//go:uintptrescapes
func (p *LazyProc) Call(...uintptr) (uintptr, uintptr, error) { return 0, 0, nil }

type Proc struct{}

//go:uintptrescapes
func (p *Proc) Call(...uintptr) (uintptr, uintptr, error) { return 0, 0, nil }

// NotKeptAlive is deliberately absent from the allowlist (control case).
func (d *LazyDLL) NotKeptAlive(uintptr) {}
