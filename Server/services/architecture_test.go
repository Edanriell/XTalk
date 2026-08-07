package services_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDependencyDirection protects the Clean Architecture rule shared by all
// services: dependencies point inward and inbound ports never reach adapters.
func TestDependencyDirection(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve services directory")
	}
	root := filepath.Dir(filename)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(path)
		imports := string(content)

		assertNoImport := func(forbidden ...string) {
			t.Helper()
			for _, item := range forbidden {
				if strings.Contains(imports, item) {
					t.Errorf("%s imports outer layer %s", normalized, item)
				}
			}
		}

		switch {
		case strings.Contains(normalized, "/domain/"):
			assertNoImport("/application/", "/adapters/", "/ports/")
		case strings.Contains(normalized, "/application/"):
			assertNoImport("/adapters/", "/ports/")
		case strings.Contains(normalized, "/ports/"):
			assertNoImport("/adapters/")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk services: %v", err)
	}
}
