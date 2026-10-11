package web

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"
)

var embeddedAssetPattern = regexp.MustCompile(`(?:src|href)="(/assets/[^"?#]+)`)

func TestEmbeddedIndexReferencesExistingAssets(t *testing.T) {
	index, err := fs.ReadFile(DistFS, "index.html")
	if err != nil {
		t.Fatalf("read embedded index: %v", err)
	}

	matches := embeddedAssetPattern.FindAllStringSubmatch(string(index), -1)
	if len(matches) == 0 {
		t.Fatal("embedded index does not reference any frontend assets")
	}
	for _, match := range matches {
		assetPath := strings.TrimPrefix(match[1], "/")
		if _, err := fs.Stat(DistFS, assetPath); err != nil {
			t.Errorf("embedded index references missing asset %q: %v", assetPath, err)
		}
	}
}

func TestEmbeddedAssetsIncludePrefixedFiles(t *testing.T) {
	entries, err := os.ReadDir("dist/assets")
	if err != nil {
		t.Fatalf("read generated assets: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasPrefix(entry.Name(), "_") && !strings.HasPrefix(entry.Name(), ".")) {
			continue
		}
		assetPath := "assets/" + entry.Name()
		if _, err := fs.Stat(DistFS, assetPath); err != nil {
			t.Errorf("embedded filesystem omitted prefixed asset %q: %v", assetPath, err)
		}
	}
}
