package target

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
)

type Kind string

const (
	Video      Kind = "video"
	Season     Kind = "season"
	Episode    Kind = "episode"
	Collection Kind = "collection"
	Favorite   Kind = "favorite"
)

type Target struct {
	Kind    Kind
	BVID    string
	ID      int
	OwnerID int
	Page    int
}

var videoPattern = regexp.MustCompile(`(?i)(BV1[0-9A-Za-z]+)`)
var seasonPattern = regexp.MustCompile(`(?:/play/|^)ss(\d+)`)
var episodePattern = regexp.MustCompile(`(?:/play/|^)ep(\d+)`)

func Parse(input string) (Target, error) {
	parsed, parseErr := url.Parse(input)
	match := videoPattern.FindStringSubmatch(input)
	if len(match) == 2 {
		result := Target{Kind: Video, BVID: match[1]}
		if parseErr == nil && parsed.Query().Has("p") {
			page, err := strconv.Atoi(parsed.Query().Get("p"))
			if err != nil || page < 1 {
				return Target{}, fmt.Errorf("invalid video page %q", parsed.Query().Get("p"))
			}
			result.Page = page
		}
		return result, nil
	}
	if match = seasonPattern.FindStringSubmatch(input); len(match) == 2 {
		return numericTarget(Season, match[1])
	}
	if match = episodePattern.FindStringSubmatch(input); len(match) == 2 {
		return numericTarget(Episode, match[1])
	}
	if parseErr == nil {
		if value := parsed.Query().Get("fid"); value != "" {
			return numericTarget(Favorite, value)
		}
		if value := parsed.Query().Get("sid"); value != "" {
			result, err := numericTarget(Collection, value)
			if err != nil {
				return Target{}, err
			}
			if match := regexp.MustCompile(`/([0-9]+)/channel/`).FindStringSubmatch(parsed.Path); len(match) == 2 {
				result.OwnerID, _ = strconv.Atoi(match[1])
			}
			return result, nil
		}
	}
	return Target{}, fmt.Errorf("unsupported Bilibili target %q", input)
}

func numericTarget(kind Kind, value string) (Target, error) {
	id, err := strconv.Atoi(value)
	if err != nil || id < 1 {
		return Target{}, fmt.Errorf("invalid %s ID %q", kind, value)
	}
	return Target{Kind: kind, ID: id}, nil
}
