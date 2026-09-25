package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"bilidown/bilibili"
	appconfig "bilidown/cli/config"
	"bilidown/cli/credentials"
	"bilidown/cli/target"
	"bilidown/util"
)

func loadConfig(global *globalOptions) (appconfig.Config, string, error) {
	path := global.configPath
	if path == "" {
		var err error
		path, err = appconfig.Path()
		if err != nil {
			return appconfig.Config{}, "", err
		}
	}
	value, err := appconfig.Load(path)
	return value, path, err
}

func parseTargetInput(input string) (target.Target, error) {
	parsedURL, _ := url.Parse(input)
	if parsedURL != nil && (parsedURL.Hostname() == "b23.tv" || parsedURL.Hostname() == "bili2233.cn") {
		redirected, err := util.GetRedirectedLocation(input)
		if err != nil {
			return target.Target{}, fmt.Errorf("resolve short link: %w", err)
		}
		input = redirected
	}
	return target.Parse(input)
}

func authenticatedClient() (*bilibili.BiliClient, error) {
	credential, err := credentials.Load()
	if err != nil {
		return nil, err
	}
	if credential.SESSDATA == "" {
		return nil, fmt.Errorf("not logged in; run 'bilidl auth login'")
	}
	return &bilibili.BiliClient{SESSDATA: credential.SESSDATA}, nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
