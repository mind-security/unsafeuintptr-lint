// Command unsafeuintptr runs the unsafeuintptr analyzer.
//
// It reports conversions of unsafe.Pointer to uintptr that are not kept alive.
// Run it standalone, or as a vet tool:
//
//	go vet -vettool=$(which unsafeuintptr) ./...
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/mind-security/unsafeuintptr-lint/unsafeuintptr"
)

func main() {
	singlechecker.Main(unsafeuintptr.Analyzer)
}
