package downloader

import (
	"os"
	"path/filepath"
	"testing"

	"bilidown/bilibili"
	"bilidown/cli/stream"
)

func TestPublishWithoutOverwritePreservesExistingFile(t *testing.T) {
	directory := t.TempDir()
	temporary := filepath.Join(directory, "new.part")
	destination := filepath.Join(directory, "output.mp4")
	if err := os.WriteFile(temporary, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := publish(temporary, destination, false); err == nil {
		t.Fatal("publish() succeeded, want destination-exists error")
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("destination = %q, want old content", got)
	}
}

func TestOutputLockRejectsConcurrentWriter(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output.mp4")
	first, err := acquireOutputLock(output)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := acquireOutputLock(output)
	if err == nil {
		second.Close()
		t.Fatal("second acquireOutputLock() succeeded")
	}
}

func TestOutputExtensionUsesMatroskaForFLACMerge(t *testing.T) {
	got, err := outputExtension(Options{Container: "auto", Mode: "merge"}, stream.Plan{
		Audio: bilibili.Media{Codecs: "fLaC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != ".mkv" {
		t.Fatalf("outputExtension() = %q, want .mkv", got)
	}
}

func TestOutputExtensionRejectsFLACInMP4(t *testing.T) {
	_, err := outputExtension(Options{Container: "mp4", Mode: "merge"}, stream.Plan{
		Audio: bilibili.Media{MimeType: "audio/flac"},
	})
	if err == nil {
		t.Fatal("outputExtension() succeeded, want an incompatibility error")
	}
}
