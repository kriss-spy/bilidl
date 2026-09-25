package main

import (
	"fmt"
	"sort"

	"bilidown/cli/catalog"

	"github.com/spf13/cobra"
)

func newInfoCmd(global *globalOptions) *cobra.Command {
	var items string
	cmd := &cobra.Command{
		Use: "info <url-or-id>", Short: "Show information about a Bilibili link", Example: "  bilidl info BV1LLDCYJEU3\n  bilidl info <season-url>\n  bilidl info <favorites-url> --json", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := authenticatedClient()
			if err != nil {
				return err
			}
			parsed, err := parseTargetInput(args[0])
			if err != nil {
				return err
			}
			result, err := catalog.Resolve(client, parsed)
			if err != nil {
				return err
			}
			if items != "" {
				selected, err := selectItems(cmd, result.Items, &downloadOptions{items: items, noInput: true})
				if err != nil {
					return err
				}
				result.Items = selected
			}
			if global.json {
				if global.quiet {
					return nil
				}
				return writeJSON(cmd.OutOrStdout(), result)
			}
			if global.quiet {
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), result.Title)
			if len(result.Items) > 0 && result.Items[0].Owner != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Uploader: %s\n", result.Items[0].Owner)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d item(s)\n", len(result.Items))
			for index, item := range result.Items {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%d:%02d\n", index+1, item.BVID, item.Title, item.Duration/60, item.Duration%60)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&items, "items", "", "show only selected item numbers")
	return cmd
}

func newFormatsCmd(global *globalOptions) *cobra.Command {
	var itemNumber int
	cmd := &cobra.Command{
		Use: "formats <url-or-id>", Short: "List available video and audio streams", Example: "  bilidl formats BV1LLDCYJEU3\n  bilidl formats BV1LLDCYJEU3 --item 2\n  bilidl formats BV1LLDCYJEU3 --json", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := authenticatedClient()
			if err != nil {
				return err
			}
			parsed, err := parseTargetInput(args[0])
			if err != nil {
				return err
			}
			resolved, err := catalog.Resolve(client, parsed)
			if err != nil {
				return err
			}
			if len(resolved.Items) == 0 {
				return fmt.Errorf("target contains no items")
			}
			if itemNumber == 0 && len(resolved.Items) > 1 {
				return fmt.Errorf("target contains %d items; choose one with --item", len(resolved.Items))
			}
			if itemNumber == 0 {
				itemNumber = 1
			}
			if itemNumber < 1 || itemNumber > len(resolved.Items) {
				return fmt.Errorf("--item must be between 1 and %d", len(resolved.Items))
			}
			item := resolved.Items[itemNumber-1]
			play, err := client.GetPlayInfo(item.BVID, item.CID)
			if err != nil {
				return err
			}
			if play.Dash == nil {
				return fmt.Errorf("play information does not contain DASH streams")
			}
			if global.json {
				if global.quiet {
					return nil
				}
				return writeJSON(cmd.OutOrStdout(), play.Dash)
			}
			if global.quiet {
				return nil
			}
			sort.Slice(play.Dash.Video, func(i, j int) bool {
				if play.Dash.Video[i].ID == play.Dash.Video[j].ID {
					return play.Dash.Video[i].Codecid < play.Dash.Video[j].Codecid
				}
				return play.Dash.Video[i].ID > play.Dash.Video[j].ID
			})
			fmt.Fprintln(cmd.OutOrStdout(), "VIDEO")
			fmt.Fprintln(cmd.OutOrStdout(), "QUALITY\tCODEC\tRESOLUTION\tFPS\tBANDWIDTH")
			for _, media := range play.Dash.Video {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%dx%d\t%s\t%d\n", media.ID, codecName(media.Codecid), media.Width, media.Height, media.FrameRate, media.Bandwidth)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nAUDIO")
			fmt.Fprintln(cmd.OutOrStdout(), "QUALITY\tCODEC\tBANDWIDTH")
			if play.Dash.Flac != nil {
				media := play.Dash.Flac.Audio
				fmt.Fprintf(cmd.OutOrStdout(), "hi-res\t%s\t%d\n", media.Codecs, media.Bandwidth)
			}
			for _, media := range play.Dash.Audio {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%d\n", media.ID, media.Codecs, media.Bandwidth)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&itemNumber, "item", 0, "inspect one page or episode")
	return cmd
}

func codecName(code int) string {
	switch code {
	case 12:
		return "HEVC"
	case 7:
		return "AVC"
	case 13:
		return "AV1"
	default:
		return fmt.Sprint(code)
	}
}
