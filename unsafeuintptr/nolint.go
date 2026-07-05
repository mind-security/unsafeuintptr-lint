package unsafeuintptr

import (
	"go/ast"
	"go/token"
	"os"
	"strings"
)

// directive records what a single //nolint comment suppresses.
type directive struct {
	all     bool            // bare //nolint suppresses every linter
	linters map[string]bool // //nolint:a,b suppresses only the named linters
}

func (d directive) covers(name string) bool {
	return d.all || d.linters[name]
}

// buildNolint scans the files for //nolint directives and returns a predicate
// reporting whether a diagnostic at pos is suppressed for this analyzer.
//
// It honors two placements, matching common //nolint usage:
//   - a trailing directive on the same line as the finding, and
//   - a standalone directive on the line immediately above the finding.
//
// A bare //nolint suppresses everything; //nolint:a,b only suppresses when
// "unsafeuintptr" is among the listed names.
func buildNolint(fset *token.FileSet, files []*ast.File) func(token.Pos) bool {
	// sameLine[filename][line] applies to findings on that line (trailing form).
	// nextLine[filename][line] applies to findings on that line (standalone
	// directive sitting on the previous line).
	sameLine := map[string]map[int]directive{}
	nextLine := map[string]map[int]directive{}

	lines := newLineCache()
	for _, f := range files {
		for _, group := range f.Comments {
			for _, comment := range group.List {
				d, ok := parseNolint(comment.Text)
				if !ok {
					continue
				}
				pos := fset.Position(comment.Slash)
				record(sameLine, pos.Filename, pos.Line, d)
				if lines.isStandalone(pos.Filename, pos.Line, pos.Column) {
					record(nextLine, pos.Filename, pos.Line+1, d)
				}
			}
		}
	}

	return func(pos token.Pos) bool {
		p := fset.Position(pos)
		if d, ok := sameLine[p.Filename][p.Line]; ok && d.covers(analyzerName) {
			return true
		}
		if d, ok := nextLine[p.Filename][p.Line]; ok && d.covers(analyzerName) {
			return true
		}
		return false
	}
}

// parseNolint parses a comment's text (including the leading //) into a
// directive. ok is false when the comment is not a //nolint directive.
func parseNolint(text string) (directive, bool) {
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	if !strings.HasPrefix(body, "nolint") {
		return directive{}, false
	}
	rest := strings.TrimPrefix(body, "nolint")
	if rest == "" {
		return directive{all: true}, true // bare //nolint
	}
	if !strings.HasPrefix(rest, ":") {
		return directive{}, false // e.g. "nolintother", not a directive
	}
	// Drop any trailing explanation, e.g. "//nolint:foo // reason".
	rest = rest[1:]
	if i := strings.IndexAny(rest, " \t"); i >= 0 {
		rest = rest[:i]
	}
	linters := map[string]bool{}
	for name := range strings.SplitSeq(rest, ",") {
		if name = strings.TrimSpace(name); name != "" {
			linters[name] = true
		}
	}
	if len(linters) == 0 {
		return directive{}, false
	}
	return directive{linters: linters}, true
}

func record(m map[string]map[int]directive, file string, line int, d directive) {
	byLine := m[file]
	if byLine == nil {
		byLine = map[int]directive{}
		m[file] = byLine
	}
	existing := byLine[line]
	existing.all = existing.all || d.all
	for name := range d.linters {
		if existing.linters == nil {
			existing.linters = map[string]bool{}
		}
		existing.linters[name] = true
	}
	byLine[line] = existing
}

// lineCache lazily reads source files so standalone directives (whose line has
// only whitespace before the comment) can be distinguished from trailing ones.
type lineCache struct {
	files map[string][]string
}

func newLineCache() *lineCache {
	return &lineCache{files: map[string][]string{}}
}

func (lc *lineCache) isStandalone(filename string, line, column int) bool {
	src, ok := lc.files[filename]
	if !ok {
		if data, err := os.ReadFile(filename); err == nil {
			src = strings.Split(string(data), "\n")
		}
		lc.files[filename] = src
	}
	if line < 1 || line > len(src) {
		return false
	}
	prefixLen := column - 1 // Column is 1-based byte offset of the comment.
	text := src[line-1]
	if prefixLen < 0 || prefixLen > len(text) {
		return false
	}
	return strings.TrimSpace(text[:prefixLen]) == ""
}
