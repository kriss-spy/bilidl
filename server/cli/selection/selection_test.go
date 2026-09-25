package selection_test

import (
	"reflect"
	"testing"

	"bilidown/cli/selection"
)

func TestParseItemSelection(t *testing.T) {
	got, err := selection.Parse("1,3-5", 6)
	if err != nil {
		t.Fatalf("parse selection: %v", err)
	}
	want := []int{1, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selection = %v, want %v", got, want)
	}
}
