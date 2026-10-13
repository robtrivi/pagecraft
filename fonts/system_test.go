package fonts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSystemCatalog_LoadFamilyProvidesWeightVariants(t *testing.T) {
	sourceDir := "/usr/share/fonts/liberation"
	regularPath := filepath.Join(sourceDir, "LiberationSans-Regular.ttf")
	boldPath := filepath.Join(sourceDir, "LiberationSans-Bold.ttf")
	for _, path := range []string{regularPath, boldPath} {
		if _, err := os.Stat(path); err != nil {
			t.Skip("Liberation Sans regular and bold faces are not installed")
		}
	}

	fontDir := t.TempDir()
	for _, path := range []string{regularPath, boldPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if err := os.WriteFile(filepath.Join(fontDir, filepath.Base(path)), data, 0600); err != nil {
			t.Fatalf("write font fixture: %v", err)
		}
	}

	catalog := NewSystemCatalog(fontDir)
	variants, ok, err := catalog.LoadFamily("Liberation Sans")
	if err != nil {
		t.Fatalf("LoadFamily: %v", err)
	}
	if !ok || variants.Regular == nil || variants.Bold == nil {
		t.Fatalf("family variants = %+v, found=%v; want regular and bold faces", variants, ok)
	}
	if variants.Regular == variants.Bold {
		t.Fatal("regular and bold variants must use their own font faces")
	}
}
