package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"bilidown/cli/config"
)

func TestLoadMergesFileOverDefaults(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	if err := os.WriteFile(path, []byte(`{"download":{"quality":"4k","jobs":5}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.Download.Quality != "4k" || got.Download.Jobs != 5 {
		t.Fatalf("unexpected file values: %#v", got.Download)
	}
	if got.Download.Codec != "auto" || got.Download.Mode != "merge" {
		t.Fatalf("defaults were not preserved: %#v", got.Download)
	}
}

func TestExpandPathExpandsHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := config.ExpandPath("~/Downloads/bilidl")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "Downloads", "bilidl")
	if got != want {
		t.Fatalf("ExpandPath() = %q, want %q", got, want)
	}
}
