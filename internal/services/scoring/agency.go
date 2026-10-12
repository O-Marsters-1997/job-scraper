package scoring

import "github.com/ollymarsters/job-scraper/internal/dto"

const AgencyQuestion = "Is this job posted by a recruitment or staffing agency on behalf of a client, rather than by the hiring company itself?"

func KindFromAnswer(a dto.Answer) dto.CompanyKind {
	panic("not implemented")
}
