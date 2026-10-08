package checks

import "slices"

type Severity string

const (
	Block Severity = "block"
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

// SkillLine is one line of the Skills section: its label and items in order.
type SkillLine struct {
	Label string
	Items []string
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
	Skills    []SkillLine
	// Bank holds the text of every Achievement the User has.
	Bank         []string
	BaseSkills   []SkillLine
	LegacySkills bool
	// BaseText holds every line of the base CV: headings, slots, profile and skills.
	BaseText  []string
	BasePages int
	// DraftPages of zero means the page count was not measured.
	DraftPages int
	// Contact nil means the Doc was not parsed.
	Contact *ContactInput
	// Parse of nil means the PDF text was not extracted.
	Parse *ParseInput
}

type ContactInput struct {
	InBody         bool
	InHeaderFooter bool
}

func Run(d Draft) []Finding {
	return slices.Concat(Grounding(d), SkillLines(d), BannedWords(d), SlotLength(d), PageCount(d), Contact(d), Parse(d))
}

func Blocking(findings []Finding) []Finding {
	return slices.DeleteFunc(slices.Clone(findings), func(f Finding) bool { return f.Severity != Block })
}
