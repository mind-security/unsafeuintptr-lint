// Package plugin registers unsafeuintptr as a golangci-lint module plugin.
//
// It is imported (for its init side effect) by a custom golangci-lint binary
// built with `golangci-lint custom`; see .custom-gcl.yml and the README.
package plugin

import (
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/mind-security/unsafeuintptr-lint/unsafeuintptr"
)

func init() {
	register.Plugin("unsafeuintptr", New)
}

// Settings mirrors the analyzer's -allow flag so it can be configured from
// .golangci.yml.
type Settings struct {
	// Allow lists additional keep-alive functions/methods to treat as safe,
	// each as "pkgpath.Func" or "pkgpath.Type.Method".
	Allow []string `json:"allow"`
}

func New(conf any) (register.LinterPlugin, error) {
	settings, err := register.DecodeSettings[Settings](conf)
	if err != nil {
		return nil, err
	}
	return &unsafeUintptrPlugin{settings: settings}, nil
}

type unsafeUintptrPlugin struct {
	settings Settings
}

func (p *unsafeUintptrPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	if len(p.settings.Allow) > 0 {
		if err := unsafeuintptr.Analyzer.Flags.Set("allow", strings.Join(p.settings.Allow, ",")); err != nil {
			return nil, err
		}
	}
	return []*analysis.Analyzer{unsafeuintptr.Analyzer}, nil
}

func (p *unsafeUintptrPlugin) GetLoadMode() string {
	// The analyzer needs full type information (it inspects operand types).
	return register.LoadModeTypesInfo
}
