package catalog

import (
	"fmt"

	"bilidown/bilibili"
	"bilidown/cli/target"
)

type Item struct {
	BVID     string `json:"bvid"`
	CID      int    `json:"cid"`
	Title    string `json:"title"`
	Owner    string `json:"owner"`
	Cover    string `json:"cover"`
	Duration int    `json:"duration"`
}

type Result struct {
	Title string `json:"title"`
	Items []Item `json:"items"`
}

type Client interface {
	GetVideoInfo(string) (*bilibili.VideoInfo, error)
	GetSeasonInfo(int, int) (*bilibili.SeasonInfo, error)
	GetFavlist(int) (*bilibili.FavList, error)
	GetSeasonsArchives(int, int) (string, []string, error)
}

func Resolve(client Client, parsed target.Target) (Result, error) {
	switch parsed.Kind {
	case target.Video:
		result, err := resolveVideo(client, parsed.BVID)
		if err != nil || parsed.Page == 0 {
			return result, err
		}
		if parsed.Page > len(result.Items) {
			return Result{}, fmt.Errorf("video page %d does not exist (video has %d pages)", parsed.Page, len(result.Items))
		}
		result.Items = result.Items[parsed.Page-1 : parsed.Page]
		return result, nil
	case target.Season, target.Episode:
		epid, ssid := 0, 0
		if parsed.Kind == target.Episode {
			epid = parsed.ID
		} else {
			ssid = parsed.ID
		}
		season, err := client.GetSeasonInfo(epid, ssid)
		if err != nil {
			return Result{}, err
		}
		result := Result{Title: season.Title}
		for _, episode := range season.Episodes {
			if parsed.Kind == target.Episode && episode.EPID != parsed.ID {
				continue
			}
			owner := ""
			if video, ownerErr := client.GetVideoInfo(episode.Bvid); ownerErr == nil {
				owner = video.Owner.Name
				if len(video.Staff) > 0 {
					owner = video.Staff[0].Name
				}
			}
			result.Items = append(result.Items, Item{BVID: episode.Bvid, CID: episode.Cid, Title: episode.LongTitle, Owner: owner, Cover: episode.Cover, Duration: episode.Duration})
		}
		if parsed.Kind == target.Episode && len(result.Items) == 0 {
			return Result{}, fmt.Errorf("episode ep%d was not found in the season", parsed.ID)
		}
		return result, nil
	case target.Favorite:
		favorite, err := client.GetFavlist(parsed.ID)
		if err != nil {
			return Result{}, err
		}
		result := Result{Title: "Favorites"}
		for _, video := range *favorite {
			result.Items = append(result.Items, Item{BVID: video.Bvid, CID: video.Ugc.FirstCid, Title: video.Title, Owner: video.Upper.Name, Cover: video.Cover, Duration: video.Duration})
		}
		return result, nil
	case target.Collection:
		if parsed.OwnerID == 0 {
			return Result{}, fmt.Errorf("collection URL does not contain an uploader ID")
		}
		title, bvids, err := client.GetSeasonsArchives(parsed.OwnerID, parsed.ID)
		if err != nil {
			return Result{}, err
		}
		if title == "" {
			title = "Collection"
		}
		result := Result{Title: title}
		for _, bvid := range bvids {
			video, err := resolveVideo(client, bvid)
			if err != nil {
				return Result{}, err
			}
			result.Items = append(result.Items, video.Items...)
		}
		return result, nil
	default:
		return Result{}, fmt.Errorf("unsupported target type %q", parsed.Kind)
	}
}

func resolveVideo(client Client, bvid string) (Result, error) {
	video, err := client.GetVideoInfo(bvid)
	if err != nil {
		return Result{}, err
	}
	owner := video.Owner.Name
	if len(video.Staff) > 0 {
		owner = video.Staff[0].Name
	}
	result := Result{Title: video.Title}
	for _, page := range video.Pages {
		result.Items = append(result.Items, Item{BVID: video.Bvid, CID: page.Cid, Title: page.Part, Owner: owner, Cover: video.Pic, Duration: page.Duration})
	}
	return result, nil
}
