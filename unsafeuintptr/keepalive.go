package unsafeuintptr

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// isKeepAliveCallee reports whether fun refers to a function that keeps its
// uintptr-typed pointer arguments alive for the duration of the call (i.e. it is
// marked //go:uintptrescapes / //go:uintptrkeepalive, or is a compiler-recognized
// syscall function).
//
// The full recognition (same-package pragma lookup, cross-package allowlist, and
// the -allow flag) is implemented alongside the allowlist. This conservative
// placeholder recognizes nothing.
func isKeepAliveCallee(pass *analysis.Pass, fun ast.Expr) bool {
	return false
}
