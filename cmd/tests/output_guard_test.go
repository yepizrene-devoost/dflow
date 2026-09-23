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
// direct write to stdout: fmt.Print/fmt.Printf/fmt.Println, the writer forms
// fmt.Fprint/fmt.Fprintf/fmt.Fprintln when their first argument is os.Stdout or
// a Cobra out-writer such as cmd.OutOrStdout(), os.Stdout.Write and builtin
// println. Cobra's out-writer defaults to stdout, so it participates in the
// single-document contract too. cmd/root and cmd/utils are deliberately
// excluded: cmd/root/completion.go and the helpers themselves are outside this
// unit's scope.
//
// Writes explicitly targeting os.Stderr are ALLOWED on purpose: stderr is
// outside the single-document stdout contract, so a diagnostic there cannot
// corrupt the machine-readable output a --json caller parses from stdout.
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
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "println" {
			return "println"
		}
		return ""
	}

	// fmt.<Name>(...) forms.
	if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fmt" {
		switch sel.Sel.Name {
		case "Print", "Printf", "Println":
			// These write to stdout implicitly.
			return "fmt." + sel.Sel.Name
		case "Fprint", "Fprintf", "Fprintln":
			// These write wherever the first argument says; only a stdout
			// target is forbidden.
			if len(call.Args) > 0 && writesToStdout(call.Args[0]) {
				return "fmt." + sel.Sel.Name
			}
		}
		return ""
	}

	// <writer>.Write(...) forms, such as os.Stdout.Write or
	// cmd.OutOrStdout().Write.
	if sel.Sel.Name == "Write" && writesToStdout(sel.X) {
		return "Write"
	}

	return ""
}

// writesToStdout reports whether expr names the process stdout stream rather
// than another writer such as os.Stderr. It recognises os.Stdout and the Cobra
// out-writer accessors (cmd.OutOrStdout()), which default to stdout.
func writesToStdout(expr ast.Expr) bool {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "os" && sel.Sel.Name == "Stdout" {
			return true
		}
		return false
	}

	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "OutOrStdout" {
			return true
		}
	}

	return false
}
