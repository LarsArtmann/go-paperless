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

		params := ownParamNames(fn)

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
			if variable, ok := call.Args[kindPos].(*ast.Ident); ok && params[variable.Name] {
				return true
			}

			kind := literalKind(t, fset, callee, call.Args[kindPos])

			appendDynamicCodes(callee, kind, add)

			return true
		})
	}

	return codes
}

// ownParamNames collects the enclosing helper's parameter names so kind
// arguments forwarded from them can be recognized as pass-through.
func ownParamNames(fn *ast.FuncDecl) map[string]bool {
	names := map[string]bool{}

	if fn.Type.Params == nil {
		return names
	}

	for _, param := range fn.Type.Params.List {
		for _, name := range param.Names {
			names[name.Name] = true
		}
	}

	return names
}

// literalKind returns the unquoted string value of a helper call's kind
// argument, failing the test when the site passes anything but a literal:
// the catalog can only stay complete if the test can see what the helper
// will emit.
func literalKind(t *testing.T, fset *token.FileSet, callee string, arg ast.Expr) string {
	t.Helper()

	literal, ok := arg.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		t.Fatalf(
			"%s kind argument at %s is not a string literal; "+
				"the error-code catalog test needs a literal to track the emitted codes",
			callee,
			fset.Position(arg.Pos()),
		)
	}

	return strings.Trim(literal.Value, `"`)
}

// appendDynamicCodes maps one literal kind call site to the paperless.*
// codes its helper emits, feeding each through add.
func appendDynamicCodes(callee, kind string, add func(string)) {
	switch callee {
	case "findByName", "findNamed":
		add("paperless.decode_" + snakeUpper(kind) + "s")
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
// its kind/resource parameter (0-based). The named-entity helpers are free
// generic functions taking (c *Client, ctx, endpoint, kind, ...); fetchAllPages
// keeps its (ctx, c, path, resource) shape.
func kindArgumentPosition(callee string) int {
	switch callee {
	case "findByName",
		"findNamed",
		"ensureNamed",
		"createNamed",
		"getNamedDetail",
		"updateMatchingAlgorithm":
		return 3
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

// TestEveryConstructedErrorIsCoded is the mechanical side of the catalog's
// header claim ("every error this SDK constructs carries a paperless.* code",
// ERROR_CODES.md). It scans every non-test source of the root package and
// requires each error construction to be one of:
//
//   - an errorfamily constructor whose code is a paperless.* literal, or a
//     dynamic expression that contains at least one paperless.-prefixed
//     literal (the kind-parameterized helpers assemble codes that way), or
//   - a transparent fmt.Errorf wrap (%w), which preserves the inner code.
//
// The only bare stdlib error is the ErrInvalidConfig sentinel declaration
// itself, and it may never be returned directly — only through its coded
// wraps (paperless.invalid_config / paperless.invalid_retry). Supersedes the
// TODO_LIST f-20 meta-test row.
func TestEveryConstructedErrorIsCoded(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	codePattern := regexp.MustCompile(`^paperless\.[a-z_]+$`)

	type source struct {
		name string
		fset *token.FileSet
		file *ast.File
	}

	sources := []source{}

	var sentinel *ast.ValueSpec

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		fset := token.NewFileSet()

		file, parseErr := parser.ParseFile(fset, name, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", name, parseErr)
		}

		sources = append(sources, source{name: name, fset: fset, file: file})

		if spec := sentinelRange(file); spec != nil {
			if sentinel != nil {
				t.Fatalf(
					"%s: second ErrInvalidConfig declaration — keep exactly one sentinel",
					name,
				)
			}

			sentinel = spec
		}
	}

	if sentinel == nil {
		t.Fatal("the ErrInvalidConfig sentinel declaration disappeared; update this test")
	}

	for _, src := range sources {
		name, fset, file := src.name, src.fset, src.file

		codeLocals := defineStringLocals(file)

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			qualifier, callee := qualifiedCallee(call.Fun)

			switch {
			case qualifier == "errors" && (callee == "New" || callee == "Newf" || callee == "Join"):
				if call.Pos() < sentinel.Pos() || call.Pos() > sentinel.End() {
					t.Errorf(
						"%s: bare stdlib error construction errors.%s at %s — errors must carry a paperless.* code",
						name,
						callee,
						fset.Position(call.Pos()),
					)
				}
			case qualifier == "fmt" && callee == "Errorf":
				if len(call.Args) == 0 {
					return true
				}

				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok || !strings.Contains(literal.Value, "%w") {
					t.Errorf(
						"%s: fmt.Errorf at %s must wrap with %%w so the inner code+family survive",
						name, fset.Position(call.Pos()),
					)
				}
			case qualifier == "errorfamily" && constructorCodePosition(callee) >= 0:
				codePos := constructorCodePosition(callee)
				if codePos >= len(call.Args) {
					return true
				}

				codeArg := call.Args[codePos]

				if ident, ok := codeArg.(*ast.Ident); ok {
					rhs, defined := codeLocals[ident.Name]
					if !defined {
						t.Errorf(
							"%s: errorfamily.%s at %s passes code variable %q of unknown provenance — pass a paperless.* literal or a locally assembled code",
							name,
							callee,
							fset.Position(call.Pos()),
							ident.Name,
						)

						return true
					}

					codeArg = rhs
				}

				if literal, ok := codeArg.(*ast.BasicLit); ok {
					value := strings.Trim(literal.Value, `"`)
					if !codePattern.MatchString(value) {
						t.Errorf(
							"%s: errorfamily.%s at %s has code %q — codes must be non-empty paperless.* literals",
							name,
							callee,
							fset.Position(call.Pos()),
							value,
						)
					}

					return true
				}

				hasNamespace := false

				ast.Inspect(codeArg, func(sub ast.Node) bool {
					if literal, ok := sub.(*ast.BasicLit); ok && literal.Kind == token.STRING {
						if strings.HasPrefix(strings.Trim(literal.Value, `"`), "paperless.") {
							hasNamespace = true
						}
					}

					return true
				})

				if !hasNamespace {
					t.Errorf(
						"%s: dynamic error code at %s must be assembled from paperless.-prefixed literals",
						name,
						fset.Position(call.Pos()),
					)
				}
			}

			return true
		})

		ast.Inspect(file, func(node ast.Node) bool {
			ret, ok := node.(*ast.ReturnStmt)
			if !ok {
				return true
			}

			for _, result := range ret.Results {
				if ident, ok := result.(*ast.Ident); ok && ident.Name == "ErrInvalidConfig" {
					t.Errorf(
						"%s: bare sentinel returned at %s — return it through its coded wrap",
						name, fset.Position(ret.Pos()),
					)
				}
			}

			return true
		})
	}
}

// sentinelRange locates the ErrInvalidConfig var declaration, the single
// sanctioned bare-stdlib error site, so the coded-error test can exempt it.
func sentinelRange(file *ast.File) *ast.ValueSpec {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}

		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != "ErrInvalidConfig" {
				continue
			}

			return value
		}
	}

	return nil
}

// qualifiedCallee splits a call's function expression into package qualifier
// and callee name ("" qualifier for local calls), for constructor scanning.
func qualifiedCallee(fun ast.Expr) (qualifier, name string) {
	selector, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return "", calledName(fun)
	}

	if ident, ok := selector.X.(*ast.Ident); ok {
		return ident.Name, selector.Sel.Name
	}

	return "", selector.Sel.Name
}

// constructorCodePosition maps an errorfamily constructor name to the
// 0-based argument index of its code parameter, or -1 for non-constructors:
// New/Newf take (family, code, ...), the typed NewXxx take (code, ...),
// Wrap/Wrapf/WrapOnce/WrapOncef take (err, family, code, ...), and the typed
// WrapXxx take (err, code, ...).
func constructorCodePosition(callee string) int {
	switch callee {
	case "New", "Newf":
		return 1
	case "Wrap", "Wrapf", "WrapOnce", "WrapOncef":
		return 2
	}

	if strings.HasPrefix(callee, "New") {
		return 0
	}

	if strings.HasPrefix(callee, "Wrap") {
		return 1
	}

	return -1
}

// defineStringLocals maps every function-local `name := expr` definition in
// the file whose expression contains a string literal, so error-code
// arguments passed as identifiers resolve to their assembling expression.
func defineStringLocals(file *ast.File) map[string]ast.Expr {
	locals := map[string]ast.Expr{}

	ast.Inspect(file, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}

		ident, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}

		hasString := false

		ast.Inspect(assign.Rhs[0], func(sub ast.Node) bool {
			if literal, ok := sub.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				hasString = true
			}

			return true
		})

		if hasString {
			locals[ident.Name] = assign.Rhs[0]
		}

		return true
	})

	return locals
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

	for _, verb := range []string{
		"Add", "Create", "Delete", "Download", "Ensure", "Find", "Get",
		"List", "Update", "Upload",
	} {
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
