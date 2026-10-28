package dto

type CompanyKind string

const (
	KindUnclassified CompanyKind = "unclassified"
	KindLikelyAgency CompanyKind = "likely_agency"
	KindLikelyDirect CompanyKind = "likely_direct"
	KindAgency       CompanyKind = "agency"
	KindDirect       CompanyKind = "direct"
)

type Poster string

const (
	PosterDirect Poster = "direct"
	PosterAgency Poster = "agency"
)

// SetCompanyKindInput's Kind is agency or direct; a user never sets a
// likely_ or unclassified kind.
type SetCompanyKindInput struct {
	CompanyID string      `json:"-" path:"id"`
	Kind      CompanyKind `json:"kind"`
}

type UndoCompanyKindInput struct {
	CompanyID string      `json:"-" path:"id"`
	Kind      CompanyKind `json:"kind"`
}
