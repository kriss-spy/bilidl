package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"bilidown/bilibili"
	"bilidown/cli/catalog"
	appconfig "bilidown/cli/config"
	"bilidown/cli/downloader"
	"bilidown/cli/selection"

	"github.com/spf13/cobra"
)

type downloadOptions struct {
	items, quality, codec, audioQuality, mode, container string
	output, filename                                     string
	all, noInput, overwrite, noMetadata                  bool
	keepParts, noResume, dryRun                          bool
	jobs                                                 int
}

type downloadResult struct {
	BVID  string `json:"bvid"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

func defaultDownloadOptions() *downloadOptions {
	return &downloadOptions{quality: "best", codec: "auto", audioQuality: "best", mode: "merge", container: "auto", jobs: 3}
}

func newDownloadCmd(global *globalOptions) *cobra.Command {
	options := defaultDownloadOptions()
	cmd := &cobra.Command{
		Use:     "download <url-or-id>...",
		Short:   "Download videos, episodes, collections, or favorites",
		Long:    "Download video and audio from Bilibili.\n\nWhen a link contains multiple items, select them with --items or --all.",
		Example: "  bilidl download BV1LLDCYJEU3\n  bilidl download BV1LLDCYJEU3 --quality 4k\n  bilidl download <season-url> --items 1,3-5\n  bilidl download <favorites-url> --all",
		Args:    cobra.MinimumNArgs(1),
		RunE:    func(cmd *cobra.Command, args []string) error { return runDownload(cmd, global, options, args) },
	}
	addDownloadFlags(cmd, options)
	return cmd
}

func addDownloadFlags(cmd *cobra.Command, options *downloadOptions) []string {
	f := cmd.Flags()
	f.StringVar(&options.items, "items", "", "select item numbers, for example 1,3-5")
	f.BoolVar(&options.all, "all", false, "download every resolved item")
	f.BoolVar(&options.noInput, "no-input", false, "never prompt for input")
	f.StringVar(&options.quality, "quality", options.quality, "video quality: best, 8k, dolby, hdr, 4k, 1080p60, 1080p+, 1080p, 720p60, 720p, 480p, 360p, 240p")
	f.StringVar(&options.codec, "codec", options.codec, "video codec: auto, hevc, avc, av1")
	f.StringVar(&options.audioQuality, "audio-quality", options.audioQuality, "audio quality")
	f.StringVar(&options.mode, "mode", options.mode, "output mode: merge, video, audio")
	f.StringVar(&options.container, "container", options.container, "output container: auto, mp4, mkv")
	f.StringVarP(&options.output, "output", "o", "", "destination directory")
	f.StringVar(&options.filename, "filename", "", "output filename template")
	f.BoolVar(&options.overwrite, "overwrite", false, "replace existing output files")
	f.BoolVar(&options.noMetadata, "no-metadata", false, "do not add BVID and uploader metadata")
	f.BoolVar(&options.keepParts, "keep-parts", false, "keep downloaded audio and video parts")
	f.IntVarP(&options.jobs, "jobs", "j", options.jobs, "concurrent item downloads")
	f.BoolVar(&options.noResume, "no-resume", false, "restart partial downloads")
	f.BoolVar(&options.dryRun, "dry-run", false, "print the plan without downloading")
	cmd.MarkFlagsMutuallyExclusive("items", "all")
	return []string{"items", "all", "no-input", "quality", "codec", "audio-quality", "mode", "container", "output", "filename", "overwrite", "no-metadata", "keep-parts", "jobs", "no-resume", "dry-run"}
}

func runDownload(cmd *cobra.Command, global *globalOptions, options *downloadOptions, inputs []string) error {
	settings, _, err := loadConfig(global)
	if err != nil {
		return err
	}
	applyDownloadDefaults(cmd, options, settings.Download)
	options.output, err = appconfig.ExpandPath(options.output)
	if err != nil {
		return fmt.Errorf("expand output directory: %w", err)
	}
	if err := validateDownloadOptions(options); err != nil {
		return err
	}
	client, err := authenticatedClient()
	if err != nil {
		return err
	}
	var items []catalog.Item
	for _, input := range inputs {
		parsed, err := parseTargetInput(input)
		if err != nil {
			return err
		}
		resolved, err := catalog.Resolve(client, parsed)
		if err != nil {
			return fmt.Errorf("resolve %s: %w", input, err)
		}
		selected, err := selectItems(cmd, resolved.Items, options)
		if err != nil {
			return err
		}
		items = append(items, selected...)
		if global.verbose {
			fmt.Fprintf(cmd.ErrOrStderr(), "resolved %s to %d item(s), selected %d\n", input, len(resolved.Items), len(selected))
		}
	}
	if options.dryRun {
		if global.quiet {
			return nil
		}
		type planned struct {
			BVID, Title, Path, VideoCodec, AudioCodec string
			VideoQuality, AudioQuality                int
		}
		preparedItems, err := prepareDownloads(client, items, options, settings.FFmpeg.Path)
		if err != nil {
			return err
		}
		plans := make([]planned, 0, len(items))
		for index, item := range items {
			prepared := preparedItems[index]
			plans = append(plans, planned{item.BVID, item.Title, prepared.Path, prepared.Video.Codecs, prepared.Audio.Codecs, int(prepared.Video.ID), int(prepared.Audio.ID)})
		}
		if global.json {
			return writeJSON(cmd.OutOrStdout(), plans)
		}
		for i, plan := range plans {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\tvideo=%d/%s\taudio=%d/%s\t%s\n", i+1, plan.BVID, plan.Title, plan.VideoQuality, plan.VideoCodec, plan.AudioQuality, plan.AudioCodec, plan.Path)
		}
		return nil
	}
	if options.jobs < 1 {
		return fmt.Errorf("--jobs must be at least 1")
	}
	preparedItems, err := prepareDownloads(client, items, options, settings.FFmpeg.Path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	sem := make(chan struct{}, options.jobs)
	errs := make(chan error, len(items))
	results := make([]downloadResult, len(items))
	var wg sync.WaitGroup
	for index, item := range items {
		index := index
		item, prepared := item, preparedItems[index]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			path, err := downloadItem(ctx, cmd, global, client, item, prepared, options, settings.FFmpeg.Path)
			if err != nil {
				errs <- fmt.Errorf("%s: %w", item.Title, err)
				cancel()
				return
			}
			results[index] = downloadResult{BVID: item.BVID, Title: item.Title, Path: path}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return err
	}
	if global.json && !global.quiet {
		return writeJSON(cmd.OutOrStdout(), results)
	}
	return nil
}

func applyDownloadDefaults(cmd *cobra.Command, options *downloadOptions, value appconfig.Download) {
	if !cmd.Flags().Changed("output") {
		options.output = value.Directory
	}
	if !cmd.Flags().Changed("filename") {
		options.filename = value.Filename
	}
	if !cmd.Flags().Changed("quality") {
		options.quality = value.Quality
	}
	if !cmd.Flags().Changed("codec") {
		options.codec = value.Codec
	}
	if !cmd.Flags().Changed("audio-quality") {
		options.audioQuality = value.AudioQuality
	}
	if !cmd.Flags().Changed("mode") {
		options.mode = value.Mode
	}
	if !cmd.Flags().Changed("container") {
		options.container = value.Container
	}
	if !cmd.Flags().Changed("jobs") {
		options.jobs = value.Jobs
	}
	if !cmd.Flags().Changed("no-resume") {
		options.noResume = !value.Resume
	}
	if !cmd.Flags().Changed("no-metadata") {
		options.noMetadata = !value.Metadata
	}
}

func selectItems(cmd *cobra.Command, items []catalog.Item, options *downloadOptions) ([]catalog.Item, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("target contains no downloadable items")
	}
	if options.all || len(items) == 1 {
		return items, nil
	}
	value := options.items
	if value == "" {
		if options.noInput || !isTerminal(cmd.InOrStdin()) {
			return nil, fmt.Errorf("target contains %d items; use --items or --all", len(items))
		}
		for i, item := range items {
			fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n", i+1, item.Title)
		}
		fmt.Fprint(cmd.OutOrStdout(), "Select items (for example 1,3-5): ")
		if _, err := fmt.Fscanln(cmd.InOrStdin(), &value); err != nil {
			return nil, fmt.Errorf("read selection: %w", err)
		}
	}
	indices, err := selection.Parse(value, len(items))
	if err != nil {
		return nil, err
	}
	result := make([]catalog.Item, 0, len(indices))
	for _, index := range indices {
		result = append(result, items[index-1])
	}
	return result, nil
}

func isTerminal(reader any) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func validateDownloadOptions(options *downloadOptions) error {
	valid := func(value string, values ...string) bool {
		for _, candidate := range values {
			if value == candidate {
				return true
			}
		}
		return false
	}
	if !valid(options.codec, "auto", "hevc", "avc", "av1") {
		return fmt.Errorf("invalid --codec %q", options.codec)
	}
	if !valid(options.mode, "merge", "video", "audio") {
		return fmt.Errorf("invalid --mode %q", options.mode)
	}
	if !valid(options.container, "auto", "mp4", "mkv") {
		return fmt.Errorf("invalid --container %q", options.container)
	}
	if !valid(options.audioQuality, "best", "hires", "192k", "132k", "64k") {
		return fmt.Errorf("invalid --audio-quality %q", options.audioQuality)
	}
	return nil
}

func prepareDownloads(client *bilibili.BiliClient, items []catalog.Item, options *downloadOptions, configuredFFmpeg string) ([]downloader.Prepared, error) {
	prepared := make([]downloader.Prepared, len(items))
	for index, item := range items {
		plan, err := downloader.Prepare(client, item, makeDownloaderOptions(client, options, configuredFFmpeg))
		if err != nil {
			return nil, fmt.Errorf("plan %s: %w", item.Title, err)
		}
		prepared[index] = plan
	}
	if err := validateUniquePaths(prepared); err != nil {
		return nil, err
	}
	return prepared, nil
}

func validateUniquePaths(prepared []downloader.Prepared) error {
	paths := make(map[string]int, len(prepared))
	for index, plan := range prepared {
		key := strings.ToLower(filepath.Clean(plan.Path))
		if previous, exists := paths[key]; exists {
			return fmt.Errorf("items %d and %d resolve to the same output path %q; include a unique field such as {{.BVID}} in --filename", previous+1, index+1, plan.Path)
		}
		paths[key] = index
	}
	return nil
}

func downloadItem(ctx context.Context, cmd *cobra.Command, global *globalOptions, client *bilibili.BiliClient, item catalog.Item, prepared downloader.Prepared, options *downloadOptions, configuredFFmpeg string) (string, error) {
	downloaderOptions := makeDownloaderOptions(client, options, configuredFFmpeg)
	fresh, err := downloader.Prepare(client, item, downloaderOptions)
	if err != nil {
		return "", fmt.Errorf("refresh media URLs: %w", err)
	}
	if fresh.Path != prepared.Path {
		return "", fmt.Errorf("output plan changed from %q to %q; retry the command", prepared.Path, fresh.Path)
	}
	progress := func(stage string, done, total int64) {
		if !global.quiet && !global.json && total > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "\r%s %s %.1f%%", item.Title, stage, float64(done)*100/float64(total))
		}
	}
	final, err := downloader.DownloadPrepared(ctx, item, downloaderOptions, fresh, progress)
	if err != nil {
		return "", err
	}
	if !global.quiet && !global.json {
		fmt.Fprintln(cmd.ErrOrStderr())
		fmt.Fprintln(cmd.OutOrStdout(), final)
	}
	if !global.quiet && !global.json && global.verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: completed\n", item.Title)
	}
	return final, nil
}

func makeDownloaderOptions(client *bilibili.BiliClient, options *downloadOptions, configuredFFmpeg string) downloader.Options {
	return downloader.Options{
		Quality: options.quality, Codec: options.codec, AudioQuality: options.audioQuality,
		Mode: options.mode, Container: options.container, Directory: options.output,
		Filename: options.filename, FFmpeg: configuredFFmpeg, SESSDATA: client.SESSDATA,
		Overwrite: options.overwrite, Metadata: !options.noMetadata, KeepParts: options.keepParts, Resume: !options.noResume,
	}
}
