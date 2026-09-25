package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	appconfig "bilidown/cli/config"

	"github.com/spf13/cobra"
)

func newConfigCmd(global *globalOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "View or change persistent settings", Long: "View or change persistent settings.\n\nKeys: download.directory, download.filename, download.quality, download.codec, download.audio-quality, download.mode, download.container, download.jobs, download.resume, download.metadata, ffmpeg.path.", Example: "  bilidl config list\n  bilidl config get download.directory\n  bilidl config set download.quality 1080p\n  bilidl config unset download.codec"}
	cmd.AddCommand(
		&cobra.Command{Use: "list", Short: "List effective settings", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			value, _, err := loadConfig(global)
			if err != nil {
				return err
			}
			if global.quiet {
				return nil
			}
			return writeJSON(cmd.OutOrStdout(), value)
		}},
		&cobra.Command{Use: "get <key>", Short: "Print one setting", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			value, _, err := loadConfig(global)
			if err != nil {
				return err
			}
			result, err := getSetting(value, args[0])
			if err != nil {
				return err
			}
			if global.quiet {
				return nil
			}
			if global.json {
				return writeJSON(cmd.OutOrStdout(), map[string]any{args[0]: result})
			}
			fmt.Fprintln(cmd.OutOrStdout(), result)
			return nil
		}},
		&cobra.Command{Use: "set <key> <value>", Short: "Change one setting", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
			value, path, err := loadConfig(global)
			if err != nil {
				return err
			}
			if err := setSetting(&value, args[0], args[1]); err != nil {
				return err
			}
			return appconfig.Save(path, value)
		}},
		&cobra.Command{Use: "unset <key>", Short: "Restore one setting to its default", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			value, path, err := loadConfig(global)
			if err != nil {
				return err
			}
			defaultValue, err := getSetting(appconfig.Default(), args[0])
			if err != nil {
				return err
			}
			if err := setSetting(&value, args[0], fmt.Sprint(defaultValue)); err != nil {
				return err
			}
			return appconfig.Save(path, value)
		}},
		&cobra.Command{Use: "path", Short: "Print the configuration file path", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			_, path, err := loadConfig(global)
			if err != nil {
				return err
			}
			if global.quiet {
				return nil
			}
			if global.json {
				return writeJSON(cmd.OutOrStdout(), map[string]string{"path": path})
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		}},
	)
	return cmd
}

func settingsMap(value appconfig.Config) map[string]any {
	return map[string]any{
		"download.directory": value.Download.Directory, "download.filename": value.Download.Filename,
		"download.quality": value.Download.Quality, "download.codec": value.Download.Codec,
		"download.audio-quality": value.Download.AudioQuality, "download.mode": value.Download.Mode,
		"download.container": value.Download.Container, "download.jobs": value.Download.Jobs,
		"download.resume": value.Download.Resume, "download.metadata": value.Download.Metadata,
		"ffmpeg.path": value.FFmpeg.Path,
	}
}

func getSetting(value appconfig.Config, key string) (any, error) {
	result, ok := settingsMap(value)[key]
	if !ok {
		return nil, fmt.Errorf("unknown setting %q", key)
	}
	return result, nil
}

func setSetting(value *appconfig.Config, key, raw string) error {
	parseBool := func() (bool, error) { return strconv.ParseBool(raw) }
	switch key {
	case "download.directory":
		value.Download.Directory = raw
	case "download.filename":
		value.Download.Filename = raw
	case "download.quality":
		value.Download.Quality = raw
	case "download.codec":
		value.Download.Codec = raw
	case "download.audio-quality":
		value.Download.AudioQuality = raw
	case "download.mode":
		value.Download.Mode = raw
	case "download.container":
		value.Download.Container = raw
	case "download.jobs":
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return fmt.Errorf("download.jobs must be a positive integer")
		}
		value.Download.Jobs = parsed
	case "download.resume":
		parsed, err := parseBool()
		if err != nil {
			return err
		}
		value.Download.Resume = parsed
	case "download.metadata":
		parsed, err := parseBool()
		if err != nil {
			return err
		}
		value.Download.Metadata = parsed
	case "ffmpeg.path":
		value.FFmpeg.Path = raw
	default:
		return fmt.Errorf("unknown setting %q", key)
	}
	return validateConfig(*value)
}

func validateConfig(value appconfig.Config) error {
	for label, item := range map[string]string{"download.codec": value.Download.Codec, "download.mode": value.Download.Mode, "download.container": value.Download.Container} {
		allowed := map[string]map[string]bool{"download.codec": {"auto": true, "hevc": true, "avc": true, "av1": true}, "download.mode": {"merge": true, "video": true, "audio": true}, "download.container": {"auto": true, "mp4": true, "mkv": true}}
		if !allowed[label][item] {
			encoded, _ := json.Marshal(item)
			return fmt.Errorf("invalid %s value %s", label, encoded)
		}
	}
	return nil
}
