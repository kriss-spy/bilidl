package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func ExpandPath(value string) (string, error) {
	if value != "~" && !strings.HasPrefix(value, "~/") {
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if value == "~" {
		return home, nil
	}
	return filepath.Join(home, filepath.FromSlash(strings.TrimPrefix(value, "~/"))), nil
}

type Config struct {
	Download Download `json:"download"`
	FFmpeg   FFmpeg   `json:"ffmpeg"`
}

type Download struct {
	Directory    string `json:"directory"`
	Filename     string `json:"filename"`
	Quality      string `json:"quality"`
	Codec        string `json:"codec"`
	AudioQuality string `json:"audio-quality"`
	Mode         string `json:"mode"`
	Container    string `json:"container"`
	Jobs         int    `json:"jobs"`
	Resume       bool   `json:"resume"`
	Metadata     bool   `json:"metadata"`
}

type FFmpeg struct {
	Path string `json:"path"`
}

func Default() Config {
	return Config{Download: Download{
		Directory:    "download",
		Filename:     "{{.Title}} [{{.BVID}}]",
		Quality:      "best",
		Codec:        "auto",
		AudioQuality: "best",
		Mode:         "merge",
		Container:    "auto",
		Jobs:         3,
		Resume:       true,
		Metadata:     true,
	}}
}

func Load(path string) (Config, error) {
	result := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return Config{}, err
	}
	return result, nil
}

func Save(path string, value Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func Path() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "bilidl", "config.json"), nil
}
