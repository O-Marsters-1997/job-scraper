package jev

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const recruiterQuestion = "Is the company named in this posting a recruitment or staffing agency hiring on behalf of other employers, rather than the employer itself? " +
	"Weigh the company name, the wording of the description and, when present, the Industries line (for example Staffing and Recruiting)."

func (c *Client) ClassifyRecruiter(ctx context.Context, apiKey, companyName string, job dto.Job) (dto.CompanyKind, dto.Usage, error) {
	state := StateFor(job)
	state.Company = companyName

	qs := map[string]choiceQuestion{
		recruiterQuestion: {Type: "choice", Instructions: recruiterQuestion, Criteria: choiceCriteria},
	}
	answers, usage, err := c.answerBatch(ctx, apiKey, state, qs, []string{recruiterQuestion})
	if err != nil {
		return "", dto.Usage{}, err
	}

	a := answers[recruiterQuestion]
	if a.PYes > a.PNo && a.PYes > a.PNotStated {
		return dto.KindLikelyAgency, usage, nil
	}
	return dto.KindLikelyDirect, usage, nil
}
