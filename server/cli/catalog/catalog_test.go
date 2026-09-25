package catalog_test

import (
	"fmt"
	"testing"

	"bilidown/bilibili"
	"bilidown/cli/catalog"
	"bilidown/cli/target"
)

type fakeClient struct {
	season     *bilibili.SeasonInfo
	videos     map[string]*bilibili.VideoInfo
	collection []string
}

func (f fakeClient) GetVideoInfo(bvid string) (*bilibili.VideoInfo, error) {
	value := f.videos[bvid]
	if value == nil {
		return nil, fmt.Errorf("missing %s", bvid)
	}
	return value, nil
}
func (f fakeClient) GetSeasonInfo(int, int) (*bilibili.SeasonInfo, error) { return f.season, nil }
func (f fakeClient) GetFavlist(int) (*bilibili.FavList, error) {
	value := bilibili.FavList{}
	return &value, nil
}
func (f fakeClient) GetSeasonsArchives(int, int) (string, []string, error) {
	return "Test collection", f.collection, nil
}

func TestEpisodeTargetContainsOnlyRequestedEpisode(t *testing.T) {
	season := &bilibili.SeasonInfo{}
	season.Episodes = []bilibili.Episode{{EPID: 1, Bvid: "BV1one", Cid: 11}, {EPID: 2, Bvid: "BV1two", Cid: 22}}
	got, err := catalog.Resolve(fakeClient{season: season}, target.Target{Kind: target.Episode, ID: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].BVID != "BV1two" {
		t.Fatalf("items = %#v", got.Items)
	}
}

func TestCollectionContainsEveryArchive(t *testing.T) {
	first := &bilibili.VideoInfo{Bvid: "BV1one", Title: "one"}
	first.Pages = []bilibili.Page{{Cid: 11, Part: "one"}}
	second := &bilibili.VideoInfo{Bvid: "BV1two", Title: "two"}
	second.Pages = []bilibili.Page{{Cid: 22, Part: "two"}}
	client := fakeClient{collection: []string{"BV1one", "BV1two"}, videos: map[string]*bilibili.VideoInfo{"BV1one": first, "BV1two": second}}
	got, err := catalog.Resolve(client, target.Target{Kind: target.Collection, ID: 9, OwnerID: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items = %#v", got.Items)
	}
	if got.Title != "Test collection" {
		t.Fatalf("title = %q, want collection title", got.Title)
	}
}

func TestVideoPageTargetContainsOnlyRequestedPage(t *testing.T) {
	video := &bilibili.VideoInfo{Bvid: "BV1pages", Title: "pages"}
	video.Pages = []bilibili.Page{{Cid: 11, Part: "one"}, {Cid: 22, Part: "two"}}
	got, err := catalog.Resolve(fakeClient{videos: map[string]*bilibili.VideoInfo{"BV1pages": video}}, target.Target{Kind: target.Video, BVID: "BV1pages", Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].CID != 22 {
		t.Fatalf("items = %#v", got.Items)
	}
}
