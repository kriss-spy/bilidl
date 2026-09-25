package stream

import (
	"fmt"
	"sort"

	"bilidown/bilibili"
	"bilidown/common"
)

type Options struct {
	Quality      string
	Codec        string
	AudioQuality string
	Mode         string
}

type Plan struct {
	Video bilibili.Media
	Audio bilibili.Media
}

var qualities = map[string]common.MediaFormat{
	"8k": 127, "dolby": 126, "hdr": 125, "4k": 120, "1080p60": 116,
	"1080p+": 112, "1080p": 80, "720p60": 74, "720p": 64,
	"480p": 32, "360p": 16, "240p": 6,
}

var codecs = map[string]int{"hevc": 12, "avc": 7, "av1": 13}
var audioQualities = map[string]common.MediaFormat{"192k": 30280, "132k": 30232, "64k": 30216}

func Select(play *bilibili.PlayInfo, options Options) (Plan, error) {
	if play == nil || play.Dash == nil {
		return Plan{}, fmt.Errorf("play information does not contain DASH streams")
	}
	result := Plan{}
	if options.Mode != "audio" {
		quality, err := selectQuality(play.Dash.Video, options.Quality)
		if err != nil {
			return Plan{}, err
		}
		codecOrder := []int{12, 7, 13}
		if preferred, ok := codecs[options.Codec]; ok {
			codecOrder = append([]int{preferred}, codecOrder...)
		}
		seen := map[int]bool{}
		for _, codec := range codecOrder {
			if seen[codec] {
				continue
			}
			seen[codec] = true
			for _, media := range play.Dash.Video {
				if media.ID == quality && media.Codecid == codec {
					result.Video = media
					break
				}
			}
			if result.Video.BaseURL != "" {
				break
			}
		}
		if result.Video.BaseURL == "" {
			return Plan{}, fmt.Errorf("quality %s has no supported video codec", options.Quality)
		}
	}
	if options.Mode != "video" {
		if (options.AudioQuality == "best" || options.AudioQuality == "hires") && play.Dash.Flac != nil {
			result.Audio = play.Dash.Flac.Audio
		} else if options.AudioQuality == "hires" {
			return Plan{}, fmt.Errorf("Hi-Res audio is not available")
		} else if requested, ok := audioQualities[options.AudioQuality]; ok {
			for _, media := range play.Dash.Audio {
				if media.ID == requested {
					result.Audio = media
					break
				}
			}
			if result.Audio.BaseURL == "" {
				return Plan{}, fmt.Errorf("audio quality %s is not available", options.AudioQuality)
			}
		} else if options.AudioQuality != "" && options.AudioQuality != "best" && options.AudioQuality != "hires" {
			return Plan{}, fmt.Errorf("unknown audio quality %q", options.AudioQuality)
		} else if len(play.Dash.Audio) > 0 {
			audio := append([]bilibili.Media(nil), play.Dash.Audio...)
			sort.Slice(audio, func(i, j int) bool { return audio[i].ID > audio[j].ID })
			result.Audio = audio[0]
		}
		if result.Audio.BaseURL == "" {
			return Plan{}, fmt.Errorf("no audio stream is available")
		}
	}
	return result, nil
}

func selectQuality(media []bilibili.Media, name string) (common.MediaFormat, error) {
	if name != "" && name != "best" {
		quality, ok := qualities[name]
		if !ok {
			return 0, fmt.Errorf("unknown video quality %q", name)
		}
		return quality, nil
	}
	var best common.MediaFormat
	for _, item := range media {
		if item.ID > best {
			best = item.ID
		}
	}
	if best == 0 {
		return 0, fmt.Errorf("no video stream is available")
	}
	return best, nil
}
