package mapview

import (
	"strconv"
	"strings"
	"testing"
)

func TestProjectEmptyAndFlatLinesStayInView(t *testing.T) {
	if len(Project(nil)) != 0 {
		t.Fatal("nil input should project to nothing")
	}
	lines := Project([]Line{{
		ID:     "flat",
		Name:   "Flat",
		Points: [][2]float64{{-78.8, 35.8}, {-78.8, 35.8}},
	}})
	if lines[0].PointsAttr == "" {
		t.Fatal("expected a point")
	}
	for _, pair := range strings.Split(lines[0].PointsAttr, " ") {
		x, y := split(t, pair)
		if x < 0 || x > width || y < 0 || y > height {
			t.Fatalf("point %s is outside the viewBox", pair)
		}
	}
}

func TestProjectNorthIsUpAndSelectionIsKept(t *testing.T) {
	lines := Project([]Line{{
		ID:       "north",
		Name:     "North",
		Selected: true,
		Points:   [][2]float64{{-78.8, 35.7}, {-78.8, 35.9}},
	}})
	if !lines[0].Selected {
		t.Fatal("selection was dropped")
	}
	parts := strings.Split(lines[0].PointsAttr, " ")
	_, south := split(t, parts[0])
	_, north := split(t, parts[1])
	if north >= south {
		t.Fatalf("north y %v is not above south y %v", north, south)
	}
}

func split(t *testing.T, pair string) (float64, float64) {
	t.Helper()
	xRaw, yRaw, ok := strings.Cut(pair, ",")
	if !ok {
		t.Fatalf("bad point %q", pair)
	}
	x, err := strconv.ParseFloat(xRaw, 64)
	if err != nil {
		t.Fatal(err)
	}
	y, err := strconv.ParseFloat(yRaw, 64)
	if err != nil {
		t.Fatal(err)
	}
	return x, y
}
