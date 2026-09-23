package paperlesstest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
)

// The endpoint-coverage test is the fake's drift guard (decision D2 of the
// paperlesstest plan): every route constant client.go declares must have a
// served route here, and every served route must map back to the SDK. A
// new SDK endpoint without fake support fails this suite instead of
// failing mysteriously inside a consumer's test with an unexpected 404.

// clientRouteConstants parses ../client.go and returns every path* route
// constant's literal value.
func clientRouteConstants(tb testing.TB) []string {
	tb.Helper()

	source, err := os.ReadFile("../client.go")
	if err != nil {
		tb.Fatalf("read client.go: %v", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "client.go", source, 0)
	if err != nil {
		tb.Fatalf("parse client.go: %v", err)
	}

	routes := []string{}

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}

			name := valueSpec.Names[0].Name
			if !strings.HasPrefix(name, "path") {
				continue
			}

			literal, ok := valueSpec.Values[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				tb.Fatalf(
					"route constant %s is not a string literal; the coverage test cannot track it",
					name,
				)
			}

			routes = append(routes, strings.Trim(literal.Value, `"`))
		}
	}

	if len(routes) == 0 {
		tb.Fatal(
			"no path* route constants found in client.go; the coverage test's extraction broke",
		)
	}

	return routes
}

func TestFakeCoversEveryClientRoute(t *testing.T) {
	t.Parallel()

	clientRoutes := clientRouteConstants(t)

	served := slices.Clone(servedRoutes())
	slices.Sort(served)

	uncovered := []string{}

	for _, route := range clientRoutes {
		if !slices.Contains(served, route) {
			uncovered = append(uncovered, route)
		}
	}

	if len(uncovered) > 0 {
		t.Errorf(
			"SDK routes without fake support (add handlers + servedRoutes entries): %v",
			uncovered,
		)
	}

	unknown := []string{}

	for _, route := range servedRoutes() {
		if !slices.Contains(clientRoutes, route) {
			unknown = append(unknown, route)
		}
	}

	if len(unknown) > 0 {
		t.Errorf(
			"fake serves routes the SDK never references (remove or sync with client.go): %v",
			unknown,
		)
	}
}
