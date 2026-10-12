package dto

type CompanyKind string

const (
	KindUnclassified CompanyKind = "unclassified"
	KindLikelyAgency CompanyKind = "likely_agency"
	KindLikelyDirect CompanyKind = "likely_direct"
	KindAgency       CompanyKind = "agency"
	KindDirect       CompanyKind = "direct"
)

func (k CompanyKind) IsAgency() bool {
	panic("not implemented")
}

func (k CompanyKind) Confirmed() bool {
	panic("not implemented")
}

type SetCompanyKindInput struct {
	CompanyID string      `json:"-" path:"id"`
	Kind      CompanyKind `json:"kind"`
}
