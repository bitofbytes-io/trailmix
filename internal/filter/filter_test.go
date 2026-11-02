package filter

import (
	"net/url"
	"testing"

	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/google/uuid"
)

func sampleTrails() []model.Trail {
	bond := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	umstead := uuid.MustParse("10000000-0000-4000-8000-000000000003")
	return []model.Trail{
		trail("Bond Park Lake Trail", bond, "Fred G. Bond Metro Park", model.DogAllowedLeashed, 2.1, 0.45, model.DifficultyEasy),
		trail("Company Mill Trail", umstead, "William B. Umstead State Park", model.DogAllowedLeashed, 5.8, 1.37, model.DifficultyModerate),
		trail("Hemlock Bluffs Loop", uuid.New(), "Hemlock Bluffs Nature Preserve", model.DogProhibited, 1.3, 1.31, model.DifficultyEasy),
		trail("Johnston Mill Creek Trail", uuid.New(), "Johnston Mill Nature Preserve", model.DogUnknown, 2.3, 1.15, model.DifficultyEasy),
		trail("Occoneechee Summit Loop", uuid.New(), "Occoneechee Mountain", model.DogAllowedLeashed, 2.2, 4.13, model.DifficultyHard),
		trail("Bond Park Dog Meadow (sample)", bond, "Fred G. Bond Metro Park", model.DogAllowedOffleash, 0.4, 0.71, model.DifficultyEasy),
		trail("Wilkerson Preserve Loop", uuid.New(), "Wilkerson", model.DogProhibited, 2.6, 2.19, model.DifficultyModerate),
	}
}

func trail(name string, parkID uuid.UUID, park string, dog model.DogRuleCode, miles, grade float64, difficulty model.Difficulty) model.Trail {
	return model.Trail{
		ID:            uuid.New(),
		Name:          name,
		DistanceMiles: miles,
		GradePercent:  grade,
		Difficulty:    difficulty,
		Park:          model.Park{ID: parkID, Name: park},
		DogRule:       model.DogRule{Code: dog, Label: dog.Label()},
	}
}

func names(trails []model.Trail) []string {
	out := make([]string, len(trails))
	for i, trail := range trails {
		out[i] = trail.Name
	}
	return out
}

func TestApplyDistanceIncludesTheBoundary(t *testing.T) {
	max := 2.1
	got := names(Apply(sampleTrails(), Query{MaxDistanceMiles: &max}))
	want := []string{"Bond Park Dog Meadow (sample)", "Hemlock Bluffs Loop", "Bond Park Lake Trail"}
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestApplyGradeDifficultyParkAndDog(t *testing.T) {
	trails := sampleTrails()
	grade := 2.0
	got := Apply(trails, Query{MaxGradePercent: &grade, Difficulty: model.DifficultyModerate})
	if len(got) != 1 || got[0].Name != "Company Mill Trail" {
		t.Fatalf("grade+difficulty = %v", names(got))
	}

	bond := trails[0].Park.ID
	got = Apply(trails, Query{ParkID: &bond, DogRule: model.DogAllowedOffleash})
	if len(got) != 1 || got[0].Name != "Bond Park Dog Meadow (sample)" {
		t.Fatalf("park+dog = %v", names(got))
	}

	for _, rule := range []model.DogRuleCode{
		model.DogAllowedLeashed,
		model.DogAllowedOffleash,
		model.DogProhibited,
		model.DogUnknown,
	} {
		matched := Apply(trails, Query{DogRule: rule})
		if len(matched) == 0 {
			t.Fatalf("dog rule %s matched nothing", rule)
		}
		for _, trail := range matched {
			if trail.DogRule.Code != rule {
				t.Fatalf("dog rule filter let %s through for %s", trail.Name, rule)
			}
		}
	}
}

func TestApplyTextIsCaseInsensitiveAndDoesNotReorderInput(t *testing.T) {
	trails := sampleTrails()
	before := names(trails)
	// Johnston Mill is 2.3 mi and Company Mill is 5.8, so Johnston comes first.
	got := names(Apply(trails, Query{Text: "  MILL "}))
	if stringsJoin(got) != "Johnston Mill Creek Trail, Company Mill Trail" {
		t.Fatalf("text matches = %v", got)
	}
	if stringsJoin(names(trails)) != stringsJoin(before) {
		t.Fatal("Apply reordered the input slice")
	}
}

func TestParseRejectsBadValuesAndAcceptsEmpty(t *testing.T) {
	if _, err := Parse(url.Values{"max_distance": {"soon"}}); err == nil {
		t.Fatal("expected distance error")
	}
	if _, err := Parse(url.Values{"max_grade": {"-1"}}); err == nil {
		t.Fatal("expected grade error")
	}
	if _, err := Parse(url.Values{"difficulty": {"brutal"}}); err == nil {
		t.Fatal("expected difficulty error")
	}
	if _, err := Parse(url.Values{"dog": {"maybe"}}); err == nil {
		t.Fatal("expected dog-rule error")
	}
	if _, err := Parse(url.Values{"park": {"bond"}}); err == nil {
		t.Fatal("expected park error")
	}
	for _, raw := range []string{"NaN", "nan", "Inf", "+Inf", "-Inf", "Infinity", "+Infinity"} {
		if _, err := Parse(url.Values{"max_distance": {raw}}); err == nil {
			t.Fatalf("accepted non-finite distance %q", raw)
		}
		if _, err := Parse(url.Values{"max_grade": {raw}}); err == nil {
			t.Fatalf("accepted non-finite grade %q", raw)
		}
	}
	query, err := Parse(url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	if len(Apply(sampleTrails(), query)) != len(sampleTrails()) {
		t.Fatal("empty query should keep every trail")
	}
}

func stringsJoin(parts []string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += ", "
		}
		out += part
	}
	return out
}
