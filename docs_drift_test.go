package paperless

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The catalog, the flake version, and the README are release contracts this
// package can verify mechanically: a new error code without a catalog row, a
// flake version that drifted from the changelog, or a lookup verb missing
// from the README all fail these tests instead of shipping as doc drift.

func readProjectFile(t *testing.T, name string) string {
	t.Helper()

	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}

	return string(raw)
}

// snakeUpper codes the helper kind/resource the way the error builders do:
// spaces become underscores ("document type" -> document_type).
func snakeUpper(kind string) string {
	return strings.ReplaceAll(kind, " ", "_")
}

// expectedDynamicCodes resolves the paperless.* codes the kind-parameterized
// helpers construct at runtime, for every literal call site in client.go.
// A call site whose kind is not a string literal fails the test: the catalog
// can only stay complete if the test can see what the helper will emit.
func expectedDynamicCodes(t *testing.T) []string {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "client.go", nil, 0)
	if err != nil {
		t.Fatalf("parse client.go: %v", err)
	}

	codes := []string{}

	add := func(code string) {
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		ownParams := map[string]bool{}
		if fn.Type.Params != nil {
			for _, param := range fn.Type.Params.List {
				for _, name := range param.Names {
					ownParams[name.Name] = true
				}
			}
		}

		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			callee := calledName(call.Fun)

			kindPos := kindArgumentPosition(callee)
			if kindPos < 0 {
				return true
			}

			if kindPos >= len(call.Args) {
				t.Fatalf("%s call at %s lacks its kind argument", callee, fset.Position(call.Pos()))
			}

			// A kind forwarded from the enclosing helper's own parameter
			// (ensureNamed -> findNamed) is pass-through delegation: the
			// caller's literal sites already carry the emitted codes.
			if variable, ok := call.Args[kindPos].(*ast.Ident); ok && ownParams[variable.Name] {
				return true
			}

			literal, ok := call.Args[kindPos].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Fatalf(
					"%s kind argument at %s is not a string literal; the error-code catalog test needs a literal to track the emitted codes",
					callee,
					fset.Position(call.Pos()),
				)
			}

			kind := strings.Trim(literal.Value, `"`)

			switch callee {
			case "findNamed":
				add("paperless.decode_" + kind + "s")
			case "ensureNamed":
				add("paperless.decode_" + kind + "s")
				add("paperless.marshal_" + snakeUpper(kind))
				add("paperless.decode_" + snakeUpper(kind))
			case "createNamed":
				add("paperless.marshal_" + snakeUpper(kind))
				add("paperless.decode_" + snakeUpper(kind))
			case "getNamedDetail":
				add("paperless.decode_" + snakeUpper(kind))
			case "updateMatchingAlgorithm":
				add("paperless.marshal_" + kind + "_update")
			case "fetchAllPages":
				add("paperless.decode_" + snakeUpper(kind) + "s")
			}

			return true
		})
	}

	return codes
}

// calledName unwraps generic instantiation (fetchAllPages[T](...)) and
// selector calls (c.findNamed(...)) down to the plain function name.
func calledName(fun ast.Expr) string {
	switch named := fun.(type) {
	case *ast.Ident:
		return named.Name
	case *ast.IndexExpr:
		return calledName(named.X)
	case *ast.IndexListExpr:
		return calledName(named.X)
	case *ast.SelectorExpr:
		return named.Sel.Name
	default:
		return ""
	}
}

// kindArgumentPosition maps each kind-parameterized helper to the index of
// its kind/resource parameter (0-based, after the receiver).
func kindArgumentPosition(callee string) int {
	switch callee {
	case "findNamed", "ensureNamed", "createNamed", "getNamedDetail", "updateMatchingAlgorithm":
		return 2
	case "fetchAllPages":
		return 3
	default:
		return -1
	}
}

func TestErrorCodesAreDocumented(t *testing.T) {
	t.Parallel()

	catalog := readProjectFile(t, "docs/ERROR_CODES.md")

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "client.go", nil, 0)
	if err != nil {
		t.Fatalf("parse client.go: %v", err)
	}

	staticCode := regexp.MustCompile(`^paperless\.[a-z_]+$`)
	codes := expectedDynamicCodes(t)

	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}

		value := strings.Trim(literal.Value, `"`)
		if staticCode.MatchString(value) && !slices.Contains(codes, value) {
			codes = append(codes, value)
		}

		return true
	})

	if len(codes) < 30 {
		t.Fatalf("only %d codes collected — the extraction is broken, not the catalog", len(codes))
	}

	for _, code := range codes {
		if !strings.Contains(catalog, code) {
			t.Errorf("error code %q is missing from docs/ERROR_CODES.md", code)
		}
	}
}

func TestFlakeVersionMatchesChangelog(t *testing.T) {
	t.Parallel()

	flake := readProjectFile(t, "flake.nix")

	versionMatch := regexp.MustCompile(`version = "([^"]+)"`).FindStringSubmatch(flake)
	if versionMatch == nil {
		t.Fatal("flake.nix declares no version attribute")
	}

	changelog := readProjectFile(t, "CHANGELOG.md")

	releaseHeading := regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\]`).
		FindStringSubmatch(changelog)
	if releaseHeading == nil {
		t.Fatal("CHANGELOG.md declares no release version heading")
	}

	if versionMatch[1] != releaseHeading[1] {
		t.Errorf(
			"flake.nix version %s drifted from the newest CHANGELOG release %s — bump both in the release commit",
			versionMatch[1],
			releaseHeading[1],
		)
	}
}

func TestREADMEListsLookupVerbs(t *testing.T) {
	t.Parallel()

	readme := readProjectFile(t, "README.md")

	for _, verb := range []string{"Find", "Get", "Ensure"} {
		for method := range reflect.TypeFor[*Client]().Methods() {
			name := method.Name

			if !strings.HasPrefix(name, verb) || !ast.IsExported(name) {
				continue
			}

			if !strings.Contains(readme, name) {
				t.Errorf("exported lookup verb %s is missing from README.md", name)
			}
		}
	}
}
