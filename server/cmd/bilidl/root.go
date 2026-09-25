package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type globalOptions struct {
	configPath string
	json       bool
	color      string
	quiet      bool
	verbose    bool
}

func newRootCmd() *cobra.Command {
	global := &globalOptions{}
	shortcut := defaultDownloadOptions()
	cmd := &cobra.Command{
		Use:           "bilidl",
		Short:         "Download video and audio from Bilibili.",
		Long:          "Download video and audio from Bilibili.\n\nRun bilidl <url-or-id> as shorthand for bilidl download <url-or-id>.",
		Example:       "  bilidl BV1LLDCYJEU3\n  bilidl download BV1LLDCYJEU3 --quality 1080p\n  bilidl formats BV1LLDCYJEU3\n  bilidl auth login",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if global.color != "auto" && global.color != "always" && global.color != "never" {
				return fmt.Errorf("--color must be auto, always, or never")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runDownload(cmd, global, shortcut, args)
		},
	}
	flags := cmd.PersistentFlags()
	flags.StringVar(&global.configPath, "config", "", "use a specific configuration file")
	flags.BoolVar(&global.json, "json", false, "write machine-readable JSON")
	flags.StringVar(&global.color, "color", "auto", "color output: auto, always, never")
	flags.BoolVarP(&global.quiet, "quiet", "q", false, "suppress non-error output")
	flags.BoolVarP(&global.verbose, "verbose", "v", false, "show diagnostic output")

	for _, name := range addDownloadFlags(cmd, shortcut) {
		_ = cmd.Flags().MarkHidden(name)
	}

	cmd.AddCommand(
		newDownloadCmd(global),
		newInfoCmd(global),
		newFormatsCmd(global),
		newAuthCmd(global),
		newConfigCmd(global),
		newDoctorCmd(global),
		newCompletionCmd(cmd),
		newVersionCmd(global),
	)
	cmd.SetHelpCommand(&cobra.Command{Use: "help [command]", Short: "Show help for a command", RunE: func(help *cobra.Command, args []string) error {
		target, _, err := cmd.Find(args)
		if err != nil {
			return err
		}
		return target.Help()
	}})
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return fmt.Errorf("%w", err) })
	return cmd
}
