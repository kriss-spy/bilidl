package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	appconfig "bilidown/cli/config"
	"bilidown/util"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

func newDoctorCmd(global *globalOptions) *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Check authentication and external dependencies", Long: "Check the configuration file, download directory, FFmpeg version, Bilibili authentication, and Bilibili API connectivity.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		type check struct {
			Name, Status, Detail string `json:",omitempty"`
		}
		checks := []check{}
		failed := false
		settings, path, err := loadConfig(global)
		if err != nil {
			checks = append(checks, check{"configuration", "failed", err.Error()})
			failed = true
		} else {
			checks = append(checks, check{"configuration", "ok", path})
			directory, directoryErr := appconfig.ExpandPath(settings.Download.Directory)
			if directoryErr != nil {
				checks = append(checks, check{"download directory", "failed", directoryErr.Error()})
				failed = true
			} else if err := os.MkdirAll(directory, 0o755); err != nil {
				checks = append(checks, check{"download directory", "failed", err.Error()})
				failed = true
			} else {
				checks = append(checks, check{"download directory", "ok", directory})
			}
		}
		ffmpeg, err := util.GetFFmpegPath()
		if settings.FFmpeg.Path != "" {
			ffmpeg, err = settings.FFmpeg.Path, nil
		}
		if err != nil {
			checks = append(checks, check{"ffmpeg", "failed", err.Error()})
			failed = true
		} else {
			versionOutput, versionErr := exec.Command(ffmpeg, "-version").Output()
			if versionErr != nil {
				checks = append(checks, check{"ffmpeg", "failed", versionErr.Error()})
				failed = true
			} else {
				checks = append(checks, check{"ffmpeg", "ok", strings.SplitN(string(versionOutput), "\n", 2)[0]})
			}
		}
		client, err := authenticatedClient()
		if err != nil {
			checks = append(checks, check{"authentication", "failed", err.Error()})
			failed = true
		} else if valid, checkErr := client.CheckLogin(); checkErr != nil || !valid {
			detail := "session is invalid"
			if checkErr != nil {
				detail = checkErr.Error()
			}
			checks = append(checks, check{"authentication", "failed", detail})
			failed = true
		} else {
			checks = append(checks, check{"authentication", "ok", "Bilibili session is valid"})
			if _, apiErr := client.GetPopularVideos(); apiErr != nil {
				checks = append(checks, check{"Bilibili API", "failed", apiErr.Error()})
				failed = true
			} else {
				checks = append(checks, check{"Bilibili API", "ok", "reachable"})
			}
		}
		if global.quiet {
			if failed {
				return fmt.Errorf("one or more checks failed")
			}
			return nil
		}
		if global.json {
			if err := writeJSON(cmd.OutOrStdout(), checks); err != nil {
				return err
			}
		} else {
			for _, item := range checks {
				fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-15s %s\n", item.Name, coloredStatus(cmd, global, item.Status), item.Detail)
			}
		}
		if failed {
			return fmt.Errorf("one or more checks failed")
		}
		return nil
	}}
}

func coloredStatus(cmd *cobra.Command, global *globalOptions, status string) string {
	enabled := global.color == "always"
	if global.color == "auto" {
		if file, ok := cmd.OutOrStdout().(*os.File); ok {
			info, err := file.Stat()
			enabled = err == nil && info.Mode()&os.ModeCharDevice != 0
		}
	}
	if !enabled {
		return status
	}
	if status == "ok" {
		return "\x1b[32mok\x1b[0m"
	}
	return "\x1b[31mfailed\x1b[0m"
}

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{Use: "completion <shell>", Short: "Generate shell completion scripts", Long: "Generate a completion script for bash, zsh, fish, or powershell.", Example: "  bilidl completion zsh > \"${fpath[1]}/_bilidl\"\n  bilidl completion bash > ~/.local/share/bash-completion/completions/bilidl", Args: cobra.ExactArgs(1), ValidArgs: []string{"bash", "zsh", "fish", "powershell"}, RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return root.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return root.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return root.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return root.GenPowerShellCompletion(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
	}}
}

func newVersionCmd(global *globalOptions) *cobra.Command {
	var short bool
	cmd := &cobra.Command{Use: "version", Short: "Print version information", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, _ []string) {
		if global.quiet {
			return
		}
		if global.json {
			_ = writeJSON(cmd.OutOrStdout(), map[string]string{"version": version, "commit": commit, "built": builtAt, "go": runtime.Version()})
			return
		}
		if short {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "bilidl %s\ncommit: %s\nbuilt: %s\ngo: %s\n", version, commit, builtAt, runtime.Version())
	}}
	cmd.Flags().BoolVar(&short, "short", false, "print only the version number")
	return cmd
}
