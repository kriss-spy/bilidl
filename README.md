# bilidl

`bilidl` is a command-line client for inspecting and downloading video and audio from Bilibili.

It supports individual videos, multipart videos, episodes, seasons, collections, and favorites. Downloads use Bilibili's DASH streams and FFmpeg stream copying, so media is remuxed without re-encoding.

## Requirements

- Go 1.23 or newer when building from source
- FFmpeg available in `PATH`, or configured with `bilidl config set ffmpeg.path <path>`
- A Bilibili account; login is currently required for inspection and download commands

## Build and install

Clone the repository and build the CLI:

```shell
git clone https://github.com/kriss-spy/bilidl.git
cd bilidl/server
go build -o bilidl ./cmd/bilidl
```

Install it for the current user on Linux:

```shell
install -Dm755 bilidl "$HOME/.local/bin/bilidl"
```

Ensure `$HOME/.local/bin` is included in `PATH`, then verify the installation:

```shell
bilidl version
bilidl doctor
```

## Quick start

Authenticate by scanning a QR code:

```shell
bilidl auth login
bilidl auth status
```

Inspect and download a video:

```shell
bilidl info BV1LLDCYJEU3
bilidl formats BV1LLDCYJEU3
bilidl BV1LLDCYJEU3
```

A URL or BVID passed directly to `bilidl` is shorthand for `bilidl download`.

## Supported targets

- Video or multipart-video URLs and BVIDs
- Bangumi episode URLs containing `ep<ID>`
- Season URLs containing `ss<ID>`
- Collection URLs containing `sid=<ID>` and an uploader ID
- Favorites URLs containing `fid=<ID>`
- Short links from `b23.tv` and `bili2233.cn`

For a multipart video, `?p=N` selects that page directly:

```shell
bilidl 'https://www.bilibili.com/video/BV1LLDCYJEU3?p=2'
```

## Download options

Select video quality, codec, audio quality, output mode, and container:

```shell
bilidl download BV1LLDCYJEU3 \
  --quality 1080p \
  --codec hevc \
  --audio-quality best \
  --mode merge \
  --container auto
```

Preview the resolved streams and output path without downloading:

```shell
bilidl download BV1LLDCYJEU3 --dry-run
```

For targets containing multiple items, select explicit item numbers or all items:

```shell
bilidl download '<season-or-collection-url>' --items 1,3-5
bilidl download '<season-or-collection-url>' --all
```

Non-interactive sessions require `--items` or `--all` for multi-item targets.

Downloads resume from partial files by default. Use `--no-resume` to restart, `--overwrite` to replace an existing output, and `--jobs N` to control item concurrency.

## Configuration

View the effective configuration and its location:

```shell
bilidl config list
bilidl config path
```

Change or restore a setting:

```shell
bilidl config set download.directory "$HOME/Videos"
bilidl config set download.quality 1080p
bilidl config unset download.quality
```

Run `bilidl config --help` for the complete key list.

## Machine-readable output

Commands that return structured information support `--json`:

```shell
bilidl --json info BV1LLDCYJEU3
bilidl --json formats BV1LLDCYJEU3
bilidl --json auth status
```

Use `--quiet` to suppress non-error output and `--verbose` for diagnostics.

## Shell completion

Generate completion scripts for Bash, Zsh, Fish, or PowerShell:

```shell
bilidl completion bash
bilidl completion zsh
bilidl completion fish
bilidl completion powershell
```

Consult your shell's documentation to load the generated script permanently.

## Development

The Go module and CLI entry point live under `server/`:

```shell
cd server
go test ./cmd/bilidl ./cli/...
go test -race ./cmd/bilidl ./cli/...
go vet ./cmd/bilidl ./cli/...
go build -o bilidl ./cmd/bilidl
```

## License

See [LICENSE](LICENSE).
