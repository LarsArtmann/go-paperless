package paperless

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
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

// packageSourceFiles lists the root package's non-test .go files. The
// error-code extraction parses all of them: the API is split across domain
// files (client, upload, tasks, named, documents, transport, notes,
// sharelinks, savedviews), and a single-file extraction would silently
// miss every code constructed outside the file it names.
func packageSourceFiles(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	files := []string{}

	for _, entry := range entries {
		name := entry.Name()

		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			files = append(files, name)
		}
	}

	if len(files) < 5 {
		t.Fatalf(
			"only %d package source files found — the extraction is broken, not the catalog",
			len(files),
		)
	}

	return files
}

// expectedDynamicCodes resolves the paperless.* codes the kind-parameterized
// helpers construct at runtime, for every literal call site in the package
// sources. A call site whose kind is not a string literal fails the test:
// the catalog can only stay complete if the test can see what the helper
// will emit.
func expectedDynamicCodes(t *testing.T) []string {
	t.Helper()

	fset := token.NewFileSet()

	codes := []string{}

	add := func(code string) {
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}

	for _, name := range packageSourceFiles(t) {
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		collectDynamicCodes(t, file, fset, add)
	}

	return codes
}

func collectDynamicCodes(t *testing.T, file *ast.File, fset *token.FileSet, add func(string)) {
	t.Helper()

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

	staticCode := regexp.MustCompile(`^paperless\.[a-z_]+$`)
	codes := expectedDynamicCodes(t)

	for _, name := range packageSourceFiles(t) {
		fset := token.NewFileSet()

		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

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
	}

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
// f-20 backlog row (the coded-errors meta-test).
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
		scan := codedErrorScan{
			t:           t,
			name:        src.name,
			fset:        src.fset,
			codePattern: codePattern,
			codeLocals:  defineStringLocals(src.file),
			sentinel:    sentinel,
		}

		ast.Inspect(src.file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				scan.call(call)
			}

			return true
		})

		scan.rejectBareSentinelReturns(src.file)
	}
}

// codedErrorScan carries one source file's context through the
// every-error-coded checks.
type codedErrorScan struct {
	t           *testing.T
	name        string
	fset        *token.FileSet
	codePattern *regexp.Regexp
	codeLocals  map[string]ast.Expr
	sentinel    *ast.ValueSpec
}

// failf reports one violation prefixed with the source file name.
func (s codedErrorScan) failf(format string, args ...any) {
	s.t.Errorf(s.name+": "+format, args...)
}

// call classifies one call expression and reports violations.
func (s codedErrorScan) call(call *ast.CallExpr) {
	qualifier, callee := qualifiedCallee(call.Fun)

	switch {
	case qualifier == "errors" && (callee == "New" || callee == "Newf" || callee == "Join"):
		if call.Pos() < s.sentinel.Pos() || call.Pos() > s.sentinel.End() {
			s.failf(
				"bare stdlib error construction errors.%s at %s — errors must carry a paperless.* code",
				callee,
				s.fset.Position(call.Pos()),
			)
		}
	case qualifier == "fmt" && callee == "Errorf":
		s.checkErrorfWraps(call)
	case qualifier == "errorfamily" && constructorCodePosition(callee) >= 0:
		s.checkErrorfamilyCode(call, callee)
	}
}

// checkErrorfWraps requires the format string to contain %w so the call-site
// wrap stays transparent for code+family.
func (s codedErrorScan) checkErrorfWraps(call *ast.CallExpr) {
	if len(call.Args) == 0 {
		return
	}

	if literal, ok := call.Args[0].(*ast.BasicLit); !ok || !strings.Contains(literal.Value, "%w") {
		s.failf(
			"fmt.Errorf at %s must wrap with %%w so the inner code+family survive",
			s.fset.Position(call.Pos()),
		)
	}
}

// checkErrorfamilyCode requires the constructor's code argument to be a
// paperless.* literal, a local variable initialized from such an expression,
// or an expression assembled from paperless.-prefixed literals.
func (s codedErrorScan) checkErrorfamilyCode(call *ast.CallExpr, callee string) {
	codePos := constructorCodePosition(callee)
	if codePos >= len(call.Args) {
		return
	}

	codeArg := call.Args[codePos]

	if ident, ok := codeArg.(*ast.Ident); ok {
		rhs, defined := s.codeLocals[ident.Name]
		if !defined {
			s.failf(
				"errorfamily.%s at %s passes code variable %q of unknown provenance — pass a paperless.* literal or a locally assembled code",
				callee,
				s.fset.Position(call.Pos()),
				ident.Name,
			)

			return
		}

		codeArg = rhs
	}

	if literal, ok := codeArg.(*ast.BasicLit); ok {
		if value := strings.Trim(literal.Value, `"`); !s.codePattern.MatchString(value) {
			s.failf(
				"errorfamily.%s at %s has code %q — codes must be non-empty paperless.* literals",
				callee, s.fset.Position(call.Pos()), value,
			)
		}

		return
	}

	if !assembledFromNamespace(codeArg) {
		s.failf(
			"dynamic error code at %s must be assembled from paperless.-prefixed literals",
			s.fset.Position(call.Pos()),
		)
	}
}

// assembledFromNamespace reports whether the expression contains at least
// one paperless.-prefixed string literal, so dynamically assembled codes
// cannot lose their namespace.
func assembledFromNamespace(expr ast.Expr) bool {
	hasNamespace := false

	ast.Inspect(expr, func(sub ast.Node) bool {
		if literal, ok := sub.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			if strings.HasPrefix(strings.Trim(literal.Value, `"`), "paperless.") {
				hasNamespace = true
			}
		}

		return true
	})

	return hasNamespace
}

// rejectBareSentinelReturns fails on any return of the bare sentinel ident:
// the sentinel leaves the SDK only through its coded wraps.
func (s codedErrorScan) rejectBareSentinelReturns(file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		ret, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}

		for _, result := range ret.Results {
			if ident, ok := result.(*ast.Ident); ok && ident.Name == "ErrInvalidConfig" {
				s.failf(
					"bare sentinel returned at %s — return it through its coded wrap",
					s.fset.Position(ret.Pos()),
				)
			}
		}

		return true
	})
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
func qualifiedCallee(fun ast.Expr) (string, string) {
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

// TestREADMEOptionsTableMatchesExportedOptions pins the README options
// table against the real exported Option-returning functions: every table
// row names an existing exported option, and every exported option not in
// the zero-value core (With*) must have a row — a renamed or new option
// fails here instead of drifting out of the docs.
func TestREADMEOptionsTableMatchesExportedOptions(t *testing.T) {
	t.Parallel()

	readme := readProjectFile(t, "README.md")

	fset := token.NewFileSet()

	options := map[string]bool{}

	for _, name := range packageSourceFiles(t) {
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() || !strings.HasPrefix(fn.Name.Name, "With") {
				continue
			}

			if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
				continue
			}

			if ident, ok := fn.Type.Results.List[0].Type.(*ast.Ident); ok &&
				ident.Name == "Option" {
				options[fn.Name.Name] = true
			}
		}
	}

	if len(options) < 5 {
		t.Fatalf(
			"only %d exported options found — the extraction is broken, not the table",
			len(options),
		)
	}

	rowPattern := regexp.MustCompile("`With[A-Z][A-Za-z]*\\(")

	for name := range options {
		if !rowPattern.MatchString(readme) || !strings.Contains(readme, "`"+name+"(") {
			t.Errorf("exported option %s has no README options-table row", name)
		}
	}

	for _, match := range rowPattern.FindAllStringSubmatch(readme, -1) {
		name := strings.TrimSuffix(strings.TrimPrefix(match[0], "`"), "(")

		if !options[name] {
			t.Errorf("README documents option %s, which no longer exists in the package", name)
		}
	}
}

// TestOutgoingRequestsCarryVersionAndTokenHeaders pins the wire headers
// every request must carry: the static token Authorization scheme and the
// Accept-based API version negotiation (apiVersion constant, echoed back
// by the server as negotiatedAPIVersion).
func TestOutgoingRequestsCarryVersionAndTokenHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Token secret" {
			t.Errorf("Authorization = %q, want the static token scheme", got)
		}

		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "version="+apiVersion) {
			t.Errorf("Accept = %q, want version negotiation for %q", accept, apiVersion)
		}

		w.Header().Set("Content-Type", "application/json; version=10")

		_, _ = w.Write([]byte(`{"results": [], "count": 0, "next": null, "previous": null}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "secret")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.ListStoragePaths(t.Context()); err != nil {
		t.Fatalf("ListStoragePaths: %v", err)
	}
}

// TestErrorCodeCatalogRowsAreUnique pins the catalog's one-code-one-row
// rule (ADR 0002: one code, one meaning, one family): a duplicated
// catalog row means two meanings share a code, which breaks code-based
// alerting and routing.
func TestErrorCodeCatalogRowsAreUnique(t *testing.T) {
	t.Parallel()

	catalog := readProjectFile(t, "docs/ERROR_CODES.md")

	rowPattern := regexp.MustCompile(`(?m)^\|\s*` + "`(paperless\\.[a-z_]+)`")

	counts := map[string]int{}

	for _, match := range rowPattern.FindAllStringSubmatch(catalog, -1) {
		counts[match[1]]++
	}

	if len(counts) < 20 {
		t.Fatalf(
			"only %d catalog rows parsed — the extraction is broken, not the catalog",
			len(counts),
		)
	}

	for code, count := range counts {
		if count > 1 {
			t.Errorf(
				"catalog row for %s appears %d times — one code, one meaning, one row",
				code,
				count,
			)
		}
	}
}

// TestMigrationGuideMatchesPackage pins MIGRATION.md (the v0.4→v0.5
// consumer guide) against the package it documents: every constructor,
// method, fixture helper, and field the guide names must still exist
// with the documented shape, and the two wire deltas must still be what
// the wire goldens pin. A renamed symbol or changed wire shape fails
// here instead of leaving consumers a guide that lies.
func TestMigrationGuideMatchesPackage(t *testing.T) {
	t.Parallel()

	guide := readProjectFile(t, "MIGRATION.md")

	extractedConstructors := map[string]bool{}

	fset := token.NewFileSet()

	for _, name := range packageSourceFiles(t) {
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && strings.HasPrefix(fn.Name.Name, "New") &&
				strings.HasSuffix(fn.Name.Name, "ID") {
				extractedConstructors[fn.Name.Name] = true
			}
		}
	}

	if len(extractedConstructors) < 10 {
		t.Fatalf(
			"only %d New…ID constructors found — the extraction is broken, not the guide",
			len(extractedConstructors),
		)
	}

	constructorPattern := regexp.MustCompile("`New[A-Z][A-Za-z]*ID\\(")
	for _, match := range constructorPattern.FindAllStringSubmatch(guide, -1) {
		name := strings.TrimSuffix(strings.TrimPrefix(match[0], "`"), "(")

		if !extractedConstructors[name] {
			t.Errorf("MIGRATION.md documents constructor %s, which no longer exists in the package", name)
		}
	}

	for _, documented := range []struct {
		constructor   any
		constructorID string
		typeName      any
		backing       reflect.Kind
	}{
		{NewTagID, "NewTagID", TagID{}, reflect.Int},
		{NewDocumentID, "NewDocumentID", DocumentID{}, reflect.Int64},
		{NewTaskID, "NewTaskID", TaskID{}, reflect.String},
	} {
		constructorType := reflect.TypeOf(documented.constructor)
		if constructorType == nil || constructorType.Kind() != reflect.Func || constructorType.NumIn() != 1 {
			t.Fatalf("%s is not a single-argument constructor — the extraction above should have caught this", documented.constructorID)
		}

		if got := constructorType.In(0).Kind(); got != documented.backing {
			t.Errorf(
				"%s takes %s, MIGRATION.md pins backing kind %s",
				documented.constructorID,
				got,
				documented.backing,
			)
		}

		idType := reflect.TypeOf(documented.typeName)

		getMethod, ok := idType.MethodByName("Get")
		if !ok {
			t.Fatalf("%s has no Get method — the branded-ID contract drifted", documented.constructorID)
		}

		if got := getMethod.Type.Out(0).Kind(); got != documented.backing {
			t.Errorf(
				"%s.Get() returns %s, MIGRATION.md pins backing kind %s",
				documented.constructorID,
				got,
				documented.backing,
			)
		}

		for _, methodName := range []string{"IsZero", "String", "Equal"} {
			if _, ok := idType.MethodByName(methodName); !ok {
				t.Errorf("%s lost its %s method, which MIGRATION.md documents", documented.constructorID, methodName)
			}
		}
	}

	fakeDir := "paperlesstest"

	fakeEntries, err := os.ReadDir(fakeDir)
	if err != nil {
		t.Fatalf("read paperlesstest dir: %v", err)
	}

	fakeMethods := map[string]bool{}

	for _, entry := range fakeEntries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, fakeDir+string(os.PathSeparator)+entry.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parse paperlesstest/%s: %v", entry.Name(), err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}

			fakeMethods[fn.Name.Name] = true
		}
	}

	for _, helper := range []string{"AddDocument", "AddTag", "SeedTask"} {
		if !fakeMethods[helper] {
			t.Errorf("MIGRATION.md documents paperlesstest helper %s, which no longer exists on the fake's Server", helper)
		}
	}

	updateRequest := reflect.TypeFor[UpdateDocumentRequest]()

	for _, fieldName := range []string{"CorrespondentID", "DocumentTypeID"} {
		field, ok := updateRequest.FieldByName(fieldName)
		if !ok {
			t.Fatalf("UpdateDocumentRequest lost field %s, which MIGRATION.md's FK-clearing idiom uses", fieldName)
		}

		if field.Type.Kind() != reflect.Pointer || field.Type.Elem().Kind() != reflect.Struct {
			t.Errorf(
				"UpdateDocumentRequest.%s is %s, want a pointer to a branded ID struct (the pointer-to-zero clearing idiom)",
				fieldName,
				field.Type,
			)
		}
	}

	for _, delta := range []struct {
		golden string
		marker string
	}{
		{goldenNamedCreateBody, `"id":null`},
		{goldenFKClearingPatchBody, `"correspondent":null`},
		{goldenFKClearingPatchBody, `"document_type":null`},
	} {
		if !strings.Contains(delta.golden, delta.marker) {
			t.Errorf(
				"wire golden no longer carries %s — MIGRATION.md's wire-delta section drifted from the pinned bodies",
				delta.marker,
			)
		}
	}
}
