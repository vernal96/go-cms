package extensions_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Public packages must remain importable by an application outside this module.
func TestPublicPackagesDoNotImportProjectInternal(t *testing.T) {
	for _, root := range []string{"../../kernel", "../../connectors"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				importPath, _ := strconv.Unquote(spec.Path.Value)
				if strings.HasPrefix(importPath, "github.com/vernal96/go-cms/internal/") || strings.HasPrefix(importPath, "github.com/vernal96/go-cms/connectors/internal/") {
					t.Errorf("%s imports private package %s", path, importPath)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
