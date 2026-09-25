package transfer_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"bilidown/cli/transfer"
)

func TestDownloadResumesPartFileAndPublishesAtomically(t *testing.T) {
	const content = "complete-media"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Range"); got != "bytes=5-" {
			t.Errorf("Range = %q, want bytes=5-", got)
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 5-%d/%d", len(content)-1, len(content)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte(content[5:]))
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "video.m4s")
	if err := os.WriteFile(destination+".part", []byte(content[:5]), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := transfer.Download(context.Background(), server.Client(), []string{server.URL}, destination, true, nil); err != nil {
		t.Fatalf("download: %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("content = %q", got)
	}
	if _, err := os.Stat(destination + ".part"); !os.IsNotExist(err) {
		t.Fatalf("part file still exists: %v", err)
	}
}

func TestDownloadRejectsMismatchedContentRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Range", "bytes 3-7/8")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("wrong"))
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "video.m4s")
	if err := os.WriteFile(destination+".part", []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := transfer.Download(context.Background(), server.Client(), []string{server.URL}, destination, true, nil); err == nil {
		t.Fatal("expected mismatched Content-Range to fail")
	}
}
