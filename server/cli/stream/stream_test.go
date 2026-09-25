package stream_test

import (
	"testing"

	"bilidown/bilibili"
	"bilidown/cli/stream"
	"bilidown/common"
)

func TestSelectUsesRequestedQualityAndFallsBackToAvailableCodec(t *testing.T) {
	play := &bilibili.PlayInfo{Dash: &bilibili.Dash{
		Video: []bilibili.Media{
			{ID: common.MediaFormat(120), Codecid: 7, BaseURL: "4k-avc"},
			{ID: common.MediaFormat(80), Codecid: 12, BaseURL: "1080-hevc"},
		},
		Audio: []bilibili.Media{{ID: common.MediaFormat(30280), BaseURL: "audio"}},
	}}

	got, err := stream.Select(play, stream.Options{Quality: "4k", Codec: "hevc", AudioQuality: "best", Mode: "merge"})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Video.BaseURL != "4k-avc" || got.Audio.BaseURL != "audio" {
		t.Fatalf("unexpected plan: %#v", got)
	}
}

func TestSelectUsesRequestedAudioQuality(t *testing.T) {
	play := &bilibili.PlayInfo{Dash: &bilibili.Dash{Audio: []bilibili.Media{
		{ID: common.MediaFormat(30280), BaseURL: "192k"},
		{ID: common.MediaFormat(30232), BaseURL: "132k"},
	}}}
	got, err := stream.Select(play, stream.Options{Mode: "audio", AudioQuality: "132k"})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Audio.BaseURL != "132k" {
		t.Fatalf("audio = %q", got.Audio.BaseURL)
	}
}
