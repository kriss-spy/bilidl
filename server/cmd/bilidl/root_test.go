package main

import (
	"bytes"
	"strings"
	"testing"

	"bilidown/cli/downloader"
)

func TestDuplicateOutputPathsAreRejected(t *testing.T) {
	err := validateUniquePaths([]downloader.Prepared{{Path: "Same.mp4"}, {Path: "same.mp4"}})
	if err == nil {
		t.Fatal("validateUniquePaths() succeeded, want collision error")
	}
}

func TestRootHelpDescribesTheCommandSurface(t *testing.T) {
	var output bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}

	help := output.String()
	for _, want := range []string{
		"Download video and audio from Bilibili.",
		"download",
		"info",
		"formats",
		"auth",
		"config",
		"doctor",
		"completion",
		"version",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help does not contain %q:\n%s", want, help)
		}
	}
}
