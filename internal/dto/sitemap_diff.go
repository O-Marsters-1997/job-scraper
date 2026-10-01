package dto

type SitemapDiff struct {
	New          []string
	Gone         []string
	NewCompanies []string
	GoneSkipped  bool
}
