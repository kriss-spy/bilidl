package credentials

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Credentials struct {
	SESSDATA string `json:"sessdata"`
}

func Path() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "bilidl", "credentials.json"), nil
}

func Load() (Credentials, error) {
	path, err := Path()
	if err != nil {
		return Credentials{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Credentials{SESSDATA: os.Getenv("BILIBILI_SESSDATA")}, nil
	}
	if err != nil {
		return Credentials{}, err
	}
	var result Credentials
	if err := json.Unmarshal(data, &result); err != nil {
		return Credentials{}, err
	}
	return result, nil
}

func Save(value Credentials) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func Clear() error {
	path, err := Path()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
