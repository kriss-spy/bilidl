package selection

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func Parse(value string, maximum int) ([]int, error) {
	selected := map[int]struct{}{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty item in selection %q", value)
		}
		bounds := strings.Split(part, "-")
		if len(bounds) > 2 {
			return nil, fmt.Errorf("invalid item range %q", part)
		}
		first, err := parseIndex(bounds[0], maximum)
		if err != nil {
			return nil, err
		}
		last := first
		if len(bounds) == 2 {
			last, err = parseIndex(bounds[1], maximum)
			if err != nil {
				return nil, err
			}
			if last < first {
				return nil, fmt.Errorf("item range %q is descending", part)
			}
		}
		for index := first; index <= last; index++ {
			selected[index] = struct{}{}
		}
	}

	result := make([]int, 0, len(selected))
	for index := range selected {
		result = append(result, index)
	}
	sort.Ints(result)
	return result, nil
}

func parseIndex(value string, maximum int) (int, error) {
	index, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || index < 1 || index > maximum {
		return 0, fmt.Errorf("item %q must be between 1 and %d", value, maximum)
	}
	return index, nil
}
