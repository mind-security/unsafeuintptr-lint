package unsafeuintptr_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/mindio/uintptr-cast-lint/unsafeuintptr"
)

func TestSamePackageAndAllowlist(t *testing.T) {
	// Packages a (same-package pragma) and b (built-in syscall allowlist) use
	// the default configuration.
	analysistest.Run(t, analysistest.TestData(), unsafeuintptr.Analyzer, "a", "b")
}

func TestAllowFlag(t *testing.T) {
	// Package c relies on -allow to recognize the winstub Call methods, since
	// keep-alive pragmas are not visible across package boundaries.
	const allow = "winstub.LazyProc.Call,winstub.Proc.Call"
	if err := unsafeuintptr.Analyzer.Flags.Set("allow", allow); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unsafeuintptr.Analyzer.Flags.Set("allow", "") })

	analysistest.Run(t, analysistest.TestData(), unsafeuintptr.Analyzer, "c")
}
