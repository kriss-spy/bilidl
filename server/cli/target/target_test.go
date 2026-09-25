package target_test

import (
	"testing"

	"bilidown/cli/target"
)

func TestParseVideoTarget(t *testing.T) {
	for _, input := range []string{
		"BV1LLDCYJEU3",
	} {
		got, err := target.Parse(input)
		if err != nil {
			t.Fatalf("parse %q: %v", input, err)
		}
		if got.Kind != target.Video || got.BVID != "BV1LLDCYJEU3" {
			t.Errorf("parse %q = %#v", input, got)
		}
	}
	got, err := target.Parse("https://www.bilibili.com/video/BV1LLDCYJEU3?p=2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Page != 2 {
		t.Fatalf("page = %d, want 2", got.Page)
	}
}

func TestParseCollectionTargets(t *testing.T) {
	tests := []struct {
		input string
		kind  target.Kind
		id    int
	}{
		{"https://www.bilibili.com/bangumi/play/ss48831", target.Season, 48831},
		{"https://www.bilibili.com/bangumi/play/ep820709", target.Episode, 820709},
		{"https://space.bilibili.com/1/favlist?fid=1234", target.Favorite, 1234},
		{"https://space.bilibili.com/282/channel/collectiondetail?sid=987", target.Collection, 987},
	}
	for _, test := range tests {
		got, err := target.Parse(test.input)
		if err != nil {
			t.Fatalf("parse %q: %v", test.input, err)
		}
		if got.Kind != test.kind || got.ID != test.id {
			t.Errorf("parse %q = %#v", test.input, got)
		}
	}
}
