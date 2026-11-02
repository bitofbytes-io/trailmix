package pages

import "github.com/a-h/templ"

// LogItem is one visit, used in the list and the entry panel.
type LogItem struct {
	ID            string
	Href          string
	EditHref      string
	TrailName     string
	TrailHref     string
	Distance      string
	Dog           string
	Visited       string
	VisitedISO    string
	Duration      string
	DurationInput string
	Notes         string
	HasPhoto      bool
	PhotoURL      string
	Selected      bool
}

// HomeView is the log screen from the wireframe: random button, map, list, entry.
type HomeView struct {
	Entries     []LogItem
	HasSelected bool
	Selected    LogItem
	Map         templ.Component
}

// ParkOption is a filter choice.
type ParkOption struct {
	ID   string
	Name string
}

// DogOption is a filter choice.
type DogOption struct {
	Code  string
	Label string
}

// TrailItem is one catalog row.
type TrailItem struct {
	ID         string
	Name       string
	Park       string
	County     string
	Distance   string
	Grade      string
	Elevation  string
	Difficulty string
	Dog        string
	Href       string
}

// SearchView is the trail browser.
type SearchView struct {
	QueryText   string
	MaxDistance string
	Difficulty  string
	MaxGrade    string
	ParkID      string
	Dog         string
	Parks       []ParkOption
	DogRules    []DogOption
	Trails      []TrailItem
	Summary     string
	Error       string
	Map         templ.Component
}

// TrailView is one trail plus the form that logs a visit.
type TrailView struct {
	TrailItem
	Source        string
	Confidence    string
	LowConfidence bool
	Sample        bool
	Visited       string
	Notes         string
	Duration      string
	Map           templ.Component
	Error         string
}

// EntryView is the editable log entry.
type EntryView struct {
	LogItem
	Error       string
	RemovePhoto bool
}
