package checks

type Severity string

const (
	Block Severity = "block"
	Warn  Severity = "warn"
	Info  Severity = "info"
)

type Finding struct {
	Check    string   `json:"check"`
	Severity Severity `json:"severity"`
	SlotID   string   `json:"slotId,omitempty"`
	Message  string   `json:"message"`
	Score    *float64 `json:"score,omitempty"`
}

// Slot is one editable text: a bullet, or the profile paragraph.
type Slot struct {
	ID       string
	Text     string
	BaseText string
	// Cited holds the text of the Achievements the slot claims to draw on.
	Cited []string
}

type Position struct {
	ID string
	// Achievements holds the text of every Achievement in this Position.
	Achievements []string
	Bullets      []Slot
}

type Draft struct {
	Positions []Position
	Profile   *Slot
	Skills    []string
	// Bank holds the text of every Achievement the User has.
	Bank       []string
	BaseSkills []string
	JobSkills  []string
	BasePages  int
	// DraftPages of zero means the page count was not measured.
	DraftPages int
}

type Check func(Draft) []Finding

var All = []Check{Grounding, BannedWords, SlotLength, PageCount}

func Run(d Draft) []Finding {
	var out []Finding
	for _, c := range All {
		out = append(out, c(d)...)
	}
	return out
}
