package unsafeuintptr

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// keepAliveFuncs indexes the functions declared in the current package that are
// marked //go:uintptrescapes or //go:uintptrkeepalive. Such functions keep their
// uintptr arguments (converted from pointers) alive for the duration of the call.
//
// Pragmas across package boundaries are not visible to go/types, so this only
// covers same-package callees; cross-package callees are handled by the allowlist.
func keepAliveFuncs(pass *analysis.Pass) map[*types.Func]bool {
	m := make(map[*types.Func]bool)
	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || !hasKeepAlivePragma(fd.Doc) {
				continue
			}
			if fn, ok := pass.TypesInfo.Defs[fd.Name].(*types.Func); ok {
				m[fn] = true
			}
		}
	}
	return m
}

func hasKeepAlivePragma(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		switch c.Text {
		case "//go:uintptrescapes", "//go:uintptrkeepalive":
			return true
		}
	}
	return false
}

// isKeepAliveCallee reports whether call invokes a function that keeps its
// uintptr-typed pointer arguments alive for the duration of the call: a
// same-package function marked with a keep-alive pragma, or a cross-package
// function/method on the allowlist.
func (c *checker) isKeepAliveCallee(call *ast.CallExpr) bool {
	fn, ok := typeutil.Callee(c.pass.TypesInfo, call).(*types.Func)
	if !ok || fn == nil {
		// Calls through function values or interface methods cannot be resolved
		// to a declaration, so we cannot prove they keep the pointer alive.
		return false
	}
	if fn.Pkg() == c.pass.Pkg && c.keepAlive[fn] {
		return true
	}
	return c.allow[keyFor(fn)]
}

// keyFor returns the allowlist key for fn: "pkgpath.Func" for a plain function,
// or "pkgpath.Type.Method" for a method (pointer receivers are dereferenced).
// Returns "" when fn cannot be keyed (e.g. no package, or an unnamed receiver).
func keyFor(fn *types.Func) string {
	pkg := fn.Pkg()
	if pkg == nil {
		return ""
	}
	if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
		recv := sig.Recv().Type()
		if ptr, ok := recv.(*types.Pointer); ok {
			recv = ptr.Elem()
		}
		named, ok := recv.(*types.Named)
		if !ok {
			return ""
		}
		return pkg.Path() + "." + named.Obj().Name() + "." + fn.Name()
	}
	return pkg.Path() + "." + fn.Name()
}
