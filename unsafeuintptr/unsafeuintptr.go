// Package unsafeuintptr defines an analyzer that reports conversions of
// unsafe.Pointer to uintptr that are not protected by a keep-alive mechanism.
//
// Converting an unsafe.Pointer to a uintptr yields a plain integer with no
// pointer semantics: the garbage collector will not keep the referenced object
// alive, and (on a moving collector) will not update the stored value. Go's
// unsafe documentation defines a small set of safe patterns for this
// conversion; anything else risks referencing freed or moved memory.
//
// This analyzer is the complement of golang.org/x/tools/go/analysis/passes/unsafeptr,
// which checks the reverse direction (uintptr -> unsafe.Pointer).
package unsafeuintptr

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `report unsafe.Pointer->uintptr conversions that are not kept alive

Converting an unsafe.Pointer to a uintptr produces an integer that the garbage
collector cannot track. If the object it referenced is not otherwise kept alive,
it may be freed or moved, leaving the uintptr dangling.

The conversion is considered safe only when:
  - it is converted straight back to unsafe.Pointer within the same expression,
    with only pointer arithmetic (+, -, &^) in between, or
  - it is a call argument to a function marked //go:uintptrescapes or
    //go:uintptrkeepalive, or to a compiler-recognized syscall function.

Any other use (storing in a variable, returning, placing in a struct/slice/map,
or passing to an ordinary function) is reported.`

var Analyzer = &analysis.Analyzer{
	Name:     "unsafeuintptr",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// checker holds the per-package state needed to classify conversions.
type checker struct {
	pass *analysis.Pass
	// keepAlive is the set of same-package functions marked
	// //go:uintptrescapes or //go:uintptrkeepalive.
	keepAlive map[*types.Func]bool
	// allow is the set of cross-package keep-alive callees, keyed by keyFor.
	allow map[string]bool
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	c := &checker{
		pass:      pass,
		keepAlive: keepAliveFuncs(pass),
		allow:     buildAllowlist(),
	}

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	insp.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		call := n.(*ast.CallExpr)
		if !isPointerToUintptr(pass, call) {
			return true
		}
		if c.isSafeContext(stack) {
			return true
		}
		pass.Reportf(call.Pos(), "unsafe.Pointer converted to uintptr is not kept alive; "+
			"the referenced object may be freed or moved. Convert only in-place in pointer "+
			"arithmetic or as an argument to a //go:uintptrescapes function")
		return true
	})

	return nil, nil
}

// isPointerToUintptr reports whether call is a conversion of an
// unsafe.Pointer-typed value to uintptr, e.g. uintptr(unsafe.Pointer(p)) or
// uintptr(u) where u has type unsafe.Pointer.
func isPointerToUintptr(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) != 1 {
		return false
	}
	tv, ok := pass.TypesInfo.Types[call.Fun]
	if !ok || !tv.IsType() {
		return false
	}
	if !isBasic(tv.Type, types.Uintptr) {
		return false
	}
	return isBasic(pass.TypesInfo.TypeOf(call.Args[0]), types.UnsafePointer)
}

// isSafeContext walks up the ancestor stack from the conversion node and reports
// whether it sits in a provably-safe position: converted back to unsafe.Pointer
// through pointer arithmetic, or passed directly (possibly through arithmetic) to
// a keep-alive function.
//
// stack is ordered from the root file down to the conversion node itself, so
// stack[len(stack)-1] is the conversion and we walk toward index 0.
func (c *checker) isSafeContext(stack []ast.Node) bool {
	child := stack[len(stack)-1]
	for i := len(stack) - 2; i >= 0; i-- {
		parent := stack[i]
		switch p := parent.(type) {
		case *ast.ParenExpr:
			// Transparent; keep walking with the same child role.
		case *ast.BinaryExpr:
			// Pointer arithmetic keeps the pointer on one operand; the offset is
			// an ordinary integer. Only continue if our node is the operand that
			// carries the pointer.
			if !isPointerArith(p) || (child != p.X && child != p.Y) {
				return false
			}
		case *ast.CallExpr:
			// Reached a call/conversion enclosing the (possibly arithmetic)
			// expression.
			if child == p.Fun {
				// Our expression is the callee, not an argument (e.g. a
				// method value); that is not a keep-alive use.
				return false
			}
			if isConversionTo(c.pass, p, types.UnsafePointer) {
				// unsafe.Pointer(uintptr(...) + off): converted straight back.
				return true
			}
			// A call argument is only safe if the callee keeps the pointer alive.
			return c.isKeepAliveCallee(p)
		default:
			return false
		}
		child = parent
	}
	return false
}

// isPointerArith reports whether op is arithmetic permitted between a
// unsafe.Pointer->uintptr conversion and its conversion back to unsafe.Pointer.
func isPointerArith(b *ast.BinaryExpr) bool {
	switch b.Op {
	case token.ADD, token.SUB, token.AND_NOT:
		return true
	}
	return false
}

// isConversionTo reports whether call is a type conversion to the named basic kind.
func isConversionTo(pass *analysis.Pass, call *ast.CallExpr, kind types.BasicKind) bool {
	if len(call.Args) != 1 {
		return false
	}
	tv, ok := pass.TypesInfo.Types[call.Fun]
	if !ok || !tv.IsType() {
		return false
	}
	return isBasic(tv.Type, kind)
}

func isBasic(t types.Type, kind types.BasicKind) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == kind
}
