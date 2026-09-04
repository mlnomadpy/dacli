package docs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRepositoryHygiene(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(here), ".."))

	rootText, err := filepath.Glob(filepath.Join(root, "*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rootText) > 0 {
		t.Fatalf("root text artifacts = %v; put local-only text in .scratch/", rootText)
	}

	ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".scratch/*", "!.scratch/README.md"} {
		if !strings.Contains(string(ignore), want) {
			t.Errorf(".gitignore missing scratch-space contract %q", want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".scratch", "README.md")); err != nil {
		t.Fatalf("documented scratch directory: %v", err)
	}

	assertPublishedRasterAssetsReferenced(t, root)
}

func TestLicenseMetadataUsesBSD3Clause(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(here), ".."))
	license, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"BSD 3-Clause License", "Copyright (c) 2026, Taha Bouhsine", "Neither the name of the copyright holder"} {
		if !strings.Contains(string(license), want) {
			t.Errorf("LICENSE missing canonical BSD-3-Clause term %q", want)
		}
	}
	for _, name := range []string{"README.md", filepath.Join("docs", "index.md")} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "BSD--3--Clause") || strings.Contains(string(body), "license-MIT") {
			t.Errorf("%s does not advertise BSD-3-Clause consistently", name)
		}
	}
}

func assertPublishedRasterAssetsReferenced(t *testing.T, root string) {
	t.Helper()
	var corpus strings.Builder
	for _, start := range []string{"README.md", "docs", "overrides"} {
		path := filepath.Join(root, start)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			corpus.Write(body)
			continue
		}
		err = filepath.WalkDir(path, func(name string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || (filepath.Ext(name) != ".md" && filepath.Ext(name) != ".html") {
				return nil
			}
			body, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			corpus.Write(body)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	assets, err := filepath.Glob(filepath.Join(root, "docs", "assets", "*.png"))
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range assets {
		if !strings.Contains(corpus.String(), filepath.Base(asset)) {
			t.Errorf("unreferenced published image %s; reference it or remove it", filepath.ToSlash(asset))
		}
	}
}
