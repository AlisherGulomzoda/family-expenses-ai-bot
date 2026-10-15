package expense

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestDomainHasNoInfrastructureImports keeps the domain independent from
// transport, storage and LLM code: only the standard library and the clock.
func TestDomainHasNoInfrastructureImports(t *testing.T) {
	const module = "github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/"
	allowed := map[string]bool{module + "internal/clock": true}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			isStd := !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
			if !isStd && !allowed[path] {
				t.Errorf("%s imports %s: the domain must not depend on it", name, path)
			}
		}
	}
}
