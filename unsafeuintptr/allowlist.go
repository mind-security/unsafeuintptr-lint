package unsafeuintptr

import "strings"

// allowFlag carries the value of -allow: a comma-separated list of additional
// keep-alive callees, each written exactly as keyFor would produce it
// ("pkgpath.Func" or "pkgpath.Type.Method").
var allowFlag string

func init() {
	Analyzer.Flags.StringVar(&allowFlag, "allow", "",
		"comma-separated list of additional keep-alive functions/methods to treat "+
			"as safe, each as \"pkgpath.Func\" or \"pkgpath.Type.Method\" (e.g. "+
			"\"example.com/x.MySyscall,example.com/x.Handle.Call\")")
}

// defaultAllowlist is the set of cross-package functions and methods that the Go
// compiler treats as keeping unsafe.Pointer->uintptr arguments alive for the
// duration of the call (assembly syscalls and //go:uintptrescapes wrappers).
// Keys use the form produced by keyFor.
var defaultAllowlist = map[string]bool{
	// syscall: assembly syscall entry points.
	"syscall.Syscall":            true,
	"syscall.Syscall6":           true,
	"syscall.Syscall9":           true,
	"syscall.Syscall12":          true,
	"syscall.Syscall15":          true,
	"syscall.Syscall18":          true,
	"syscall.RawSyscall":         true,
	"syscall.RawSyscall6":        true,
	"syscall.SyscallN":           true,
	"syscall.RawSyscallN":        true,
	"syscall.AllThreadsSyscall":  true,
	"syscall.AllThreadsSyscall6": true,
	// syscall: Windows dynamic-load Call methods (their Call is //go:uintptrescapes;
	// this is the NewLazyDLL(...).NewProc(...).Call(...) idiom).
	"syscall.Proc.Call":     true,
	"syscall.LazyProc.Call": true,

	// golang.org/x/sys/unix: mirrors the syscall entry points.
	"golang.org/x/sys/unix.Syscall":           true,
	"golang.org/x/sys/unix.Syscall6":          true,
	"golang.org/x/sys/unix.Syscall9":          true,
	"golang.org/x/sys/unix.RawSyscall":        true,
	"golang.org/x/sys/unix.RawSyscall6":       true,
	"golang.org/x/sys/unix.SyscallNoError":    true,
	"golang.org/x/sys/unix.RawSyscallNoError": true,

	// golang.org/x/sys/windows: syscall entry points and dynamic-load Call methods.
	"golang.org/x/sys/windows.Syscall":       true,
	"golang.org/x/sys/windows.Syscall6":      true,
	"golang.org/x/sys/windows.Syscall9":      true,
	"golang.org/x/sys/windows.Syscall12":     true,
	"golang.org/x/sys/windows.Syscall15":     true,
	"golang.org/x/sys/windows.Syscall18":     true,
	"golang.org/x/sys/windows.SyscallN":      true,
	"golang.org/x/sys/windows.Proc.Call":     true,
	"golang.org/x/sys/windows.LazyProc.Call": true,
}

// buildAllowlist returns the effective allowlist: the built-in defaults plus any
// entries supplied via -allow.
func buildAllowlist() map[string]bool {
	allow := make(map[string]bool, len(defaultAllowlist))
	for k := range defaultAllowlist {
		allow[k] = true
	}
	for entry := range strings.SplitSeq(allowFlag, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			allow[entry] = true
		}
	}
	return allow
}
