package downloader

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"bilidown/bilibili"
	"bilidown/cli/catalog"
	"bilidown/cli/stream"
	"bilidown/cli/transfer"
	"bilidown/util"

	"github.com/gofrs/flock"
)

type PlayClient interface {
	GetPlayInfo(string, int) (*bilibili.PlayInfo, error)
}
type Progress func(stage string, downloaded, total int64)

type Options struct {
	Quality, Codec, AudioQuality, Mode, Container string
	Directory, Filename, FFmpeg, SESSDATA         string
	Overwrite, Metadata, KeepParts, Resume        bool
}

type Prepared struct {
	Path  string         `json:"path"`
	Video bilibili.Media `json:"video"`
	Audio bilibili.Media `json:"audio"`
}

func Prepare(client PlayClient, item catalog.Item, options Options) (Prepared, error) {
	play, err := client.GetPlayInfo(item.BVID, item.CID)
	if err != nil {
		return Prepared{}, err
	}
	plan, err := stream.Select(play, stream.Options{Quality: options.Quality, Codec: options.Codec, AudioQuality: options.AudioQuality, Mode: options.Mode})
	if err != nil {
		return Prepared{}, err
	}
	base, err := renderFilename(options.Filename, item, options.Quality)
	if err != nil {
		return Prepared{}, err
	}
	extension, err := outputExtension(options, plan)
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{Path: filepath.Join(options.Directory, base+extension), Video: plan.Video, Audio: plan.Audio}, nil
}

func Download(ctx context.Context, client PlayClient, item catalog.Item, options Options, progress Progress) (string, error) {
	prepared, err := Prepare(client, item, options)
	if err != nil {
		return "", err
	}
	return DownloadPrepared(ctx, item, options, prepared, progress)
}

func DownloadPrepared(ctx context.Context, item catalog.Item, options Options, prepared Prepared, progress Progress) (string, error) {
	if err := os.MkdirAll(options.Directory, 0o755); err != nil {
		return "", err
	}
	final := prepared.Path
	outputLock, err := acquireOutputLock(final)
	if err != nil {
		return "", err
	}
	defer outputLock.Close()
	if _, err := os.Stat(final); err == nil && !options.Overwrite {
		return "", fmt.Errorf("%s already exists (use --overwrite)", final)
	}
	prefix := filepath.Join(options.Directory, "."+filepath.Base(final))
	httpClient := authenticatedHTTPClient(options.SESSDATA)
	callback := func(stage string) transfer.Progress {
		return func(done, total int64) {
			if progress != nil {
				progress(stage, done, total)
			}
		}
	}
	var videoPath, audioPath string
	if prepared.Audio.BaseURL != "" {
		audioPath = prefix + ".audio"
		if err := transfer.Download(ctx, httpClient, mediaURLs(prepared.Audio), audioPath, options.Resume, callback("audio")); err != nil {
			return "", err
		}
	}
	if prepared.Video.BaseURL != "" {
		videoPath = prefix + ".video"
		if err := transfer.Download(ctx, httpClient, mediaURLs(prepared.Video), videoPath, options.Resume, callback("video")); err != nil {
			return "", err
		}
	}
	ffmpeg := options.FFmpeg
	if ffmpeg == "" {
		detected, err := util.GetFFmpegPath()
		if err != nil {
			return "", err
		}
		ffmpeg = detected
	}
	if err := remux(ctx, ffmpeg, final, videoPath, audioPath, item, options.Metadata, options.Overwrite); err != nil {
		return "", err
	}
	if !options.KeepParts {
		if videoPath != "" {
			_ = os.Remove(videoPath)
		}
		if audioPath != "" {
			_ = os.Remove(audioPath)
		}
	}
	return final, nil
}

func acquireOutputLock(outputPath string) (*flock.Flock, error) {
	lockPath := filepath.Join(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+".bilidl.lock")
	outputLock := flock.New(lockPath)
	locked, err := outputLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("lock output %s: %w", outputPath, err)
	}
	if !locked {
		return nil, fmt.Errorf("another bilidl process is writing %s", outputPath)
	}
	return outputLock, nil
}

func mediaURLs(media bilibili.Media) []string {
	return append([]string{media.BaseURL}, media.BackupURL...)
}

type headerTransport struct {
	base     http.RoundTripper
	sessdata string
}

func (t headerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Cookie", "SESSDATA="+t.sessdata)
	return t.base.RoundTrip(clone)
}
func authenticatedHTTPClient(sessdata string) *http.Client {
	return &http.Client{Transport: headerTransport{base: http.DefaultTransport, sessdata: sessdata}}
}

func renderFilename(pattern string, item catalog.Item, quality string) (string, error) {
	parsed, err := template.New("filename").Parse(pattern)
	if err != nil {
		return "", err
	}
	var value strings.Builder
	data := struct{ Title, BVID, Owner, Quality string }{item.Title, item.BVID, item.Owner, quality}
	if err := parsed.Execute(&value, data); err != nil {
		return "", err
	}
	name := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`).ReplaceAllString(value.String(), "_")
	name = strings.Trim(strings.TrimSpace(name), ".")
	if name == "" {
		return "", fmt.Errorf("filename template produced an empty name")
	}
	return name, nil
}

func outputExtension(options Options, plan stream.Plan) (string, error) {
	flac := strings.Contains(strings.ToLower(plan.Audio.Codecs), "flac") || strings.Contains(strings.ToLower(plan.Audio.MimeType), "flac")
	if options.Container == "mp4" && flac {
		return "", fmt.Errorf("FLAC audio is not compatible with --container mp4; use auto or mkv")
	}
	if options.Container == "mkv" {
		return ".mkv", nil
	}
	if options.Mode == "audio" && flac {
		return ".flac", nil
	}
	if options.Mode == "audio" {
		return ".m4a", nil
	}
	if flac {
		return ".mkv", nil
	}
	return ".mp4", nil
}

func remux(ctx context.Context, ffmpeg, destination, video, audio string, item catalog.Item, metadata, overwrite bool) error {
	temporary := destination + ".part" + filepath.Ext(destination)
	args := []string{"-y"}
	if video != "" {
		args = append(args, "-i", video)
	}
	if audio != "" {
		args = append(args, "-i", audio)
	}
	args = append(args, "-c", "copy")
	if metadata {
		args = append(args, "-metadata", "description="+item.BVID, "-metadata", "artist="+item.Owner)
	}
	args = append(args, temporary)
	if output, err := exec.CommandContext(ctx, ffmpeg, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return publish(temporary, destination, overwrite)
}

func publish(temporary, destination string, overwrite bool) error {
	if !overwrite {
		if err := os.Link(temporary, destination); err != nil {
			return err
		}
		return os.Remove(temporary)
	}
	if err := os.Rename(temporary, destination); err == nil {
		return nil
	}
	backupFile, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".backup-*")
	if err != nil {
		return err
	}
	backup := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}
	if err := os.Rename(destination, backup); err != nil {
		return err
	}
	if err := os.Rename(temporary, destination); err != nil {
		if restoreErr := os.Rename(backup, destination); restoreErr != nil {
			return fmt.Errorf("publish output: %w (also failed to restore original: %v)", err, restoreErr)
		}
		return err
	}
	_ = os.Remove(backup)
	return nil
}
