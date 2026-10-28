package dto

type DraftState string

const (
	DraftStatePending DraftState = "pending"
	DraftStateReady   DraftState = "ready"
	DraftStateFailed  DraftState = "failed"
)

type DueList struct {
	Jobs []DueJob `json:"jobs"`
}

type DueJob struct {
	JobID         string      `json:"job_id"`
	Title         string      `json:"title"`
	CompanySlug   string      `json:"company_slug"`
	CompanyKind   CompanyKind `json:"company_kind"`
	Band          string      `json:"band"`
	Score         *int        `json:"score"`
	ApplicationID string      `json:"application_id"`
	ChaseBy       *string     `json:"chase_by"` // YYYY-MM-DD
	ApplyTaskID   *string     `json:"apply_task_id"`
	Tasks         []DueTask   `json:"tasks"`
	Draft         *DueDraft   `json:"draft"`
	KeptDraftID   *string     `json:"kept_draft_id"`
}

type DueTask struct {
	ID        string            `json:"id"`
	TypeName  string            `json:"type_name"`
	SystemKey TaskTypeSystemKey `json:"system_key"`
	Notes     string            `json:"notes"`
}

type DueDraft struct {
	ID      string     `json:"id"`
	State   DraftState `json:"state"`
	CVTitle string     `json:"cv_title"`
}
