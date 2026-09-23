package tests

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommandLayerWritesGoThroughMessagesHelper guards the single output choke
// point built for WU3b. Every user-facing write in the command path must go
// through the cmd/utils helpers, so that a future output mode is a change in
// that one package instead of a hunt through the command layer.
//
// It reads the Go sources of cmd/commands and cmd/gitutils and fails on any
// direct fmt.Print/fmt.Printf/fmt.Println or builtin println call. cmd/root and
// cmd/utils are deliberately excluded: cmd/root/completion.go and the helpers
// themselves are outside this unit's scope.
//
// `go test` runs with the package directory as the working directory, so the
// sources are reached relative to it.
func TestCommandLayerWritesGoThroughMessagesHelper(t *testing.T) {
	packages := []string{"../commands", "../gitutils"}

	fset := token.NewFileSet()
	var offenders []string

	for _, pkg := range packages {
		matches, err := filepath.Glob(filepath.Join(pkg, "*.go"))
		if err != nil {
			t.Fatalf("globbing %s: %v", pkg, err)
		}
		if len(matches) == 0 {
			t.Fatalf("no Go sources found in %s; the guard would silently pass", pkg)
		}

		for _, path := range matches {
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", path, err)
			}

			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if name := directWriteName(call); name != "" {
					pos := fset.Position(call.Pos())
					offenders = append(offenders, fmt.Sprintf("%s:%d: %s(...)", pos.Filename, pos.Line, name))
				}
				return true
			})
		}
	}

	if len(offenders) > 0 {
		t.Fatalf("direct stdout writes bypass the cmd/utils output helpers; this must go through cmd/utils (Plain, Prompt, Info, Success, Warn, Error) so a future output mode is a single-file change:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// directWriteName returns the forbidden call name for a direct stdout write, or
// "" when the call is not one of the guarded forms.
func directWriteName(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fmt" {
			switch sel.Sel.Name {
			case "Print", "Printf", "Println":
				return "fmt." + sel.Sel.Name
			}
		}
		return ""
	}

	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "println" {
		return "println"
	}

	return ""
}
