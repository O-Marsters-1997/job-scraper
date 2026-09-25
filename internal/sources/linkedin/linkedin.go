package linkedin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

const (
	baseURL   = "https://www.linkedin.com"
	searchURL = baseURL + "/jobs-guest/jobs/api/seeMoreJobPostings/search"

	// maxStart bounds pagination — seeMoreJobPostings never returns a total result
	// count, so an empty page is the only end-of-results signal LinkedIn gives us.
	// LinkedIn also tends to start 429ing a given IP after ~page 10; the proxy
	// (UseProxy below) mitigates that, and this cap bounds the damage if it doesn't.
	maxStart = 1000

	minWait = 2 * time.Second
	maxWait = 7 * time.Second

	selCard         = `div.base-search-card`
	selCardLink     = `a.base-card__full-link`
	selCardTitle    = `h3.base-search-card__title`
	selCardTitleAlt = `span.sr-only`
	selCardCompany  = `h4.base-search-card__subtitle a`
	selCardLocation = `span.job-search-card__location`

	selDetailTitle        = `h1.top-card-layout__title.topcard__title`
	selDetailCompany      = `a.topcard__org-name-link`
	selDetailFlavorRow    = `.topcard__flavor-row`
	selDetailFlavorBullet = `.topcard__flavor--bullet`
	selDescription        = `div.show-more-less-html__markup`
	selCriteriaItem       = `li.description__job-criteria-item`
	selCriteriaHeader     = `h3.description__job-criteria-subheader`
	selCriteriaText       = `span.description__job-criteria-text`
	selSalary             = `div.compensation__salary-range div.salary.compensation__salary`
	selApplyURLCode       = `code#applyUrl`
)

var applyURLRe = regexp.MustCompile(`\?url=([^"]+)`)

type Search struct {
	Keywords string // maps to the keywords URL param
	Location string // maps to the location URL param; empty means no filter

	CompanyID   string // maps to f_C; numeric LinkedIn company ID
	Recency     string // maps to f_TPR; one of r86400, r604800, r2592000
	Arrangement string // maps to f_WT; 1=on-site, 2=remote, 3=hybrid
	Experience  string // maps to f_E; 1-6
	JobType     string // maps to f_JT; one of F, P, C, T, I
	GeoID       string // maps to geoId; numeric
	Distance    string // maps to f_D; numeric, miles
	SalaryBand  string // maps to f_SB2; 1-9
}

var (
	validRecency     = map[string]bool{"r86400": true, "r604800": true, "r2592000": true}
	validArrangement = map[string]bool{"1": true, "2": true, "3": true}
	validExperience  = map[string]bool{"1": true, "2": true, "3": true, "4": true, "5": true, "6": true}
	validJobType     = map[string]bool{"F": true, "P": true, "C": true, "T": true, "I": true}
	validSalaryBand  = map[string]bool{"1": true, "2": true, "3": true, "4": true, "5": true, "6": true, "7": true, "8": true, "9": true}
)

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func inSet(set map[string]bool) func(string) bool {
	return func(s string) bool { return set[s] }
}

// setFilter sets param to value if value is non-empty and valid. An invalid value is
// dropped with a warning rather than an error — a bad filter shouldn't kill the search.
func setFilter(v url.Values, param, value string, valid func(string) bool) {
	if value == "" {
		return
	}
	if !valid(value) {
		slog.Warn("linkedin: dropping invalid filter value", slog.String("param", param), slog.String("value", value))
		return
	}
	v.Set(param, value)
}

func (s Search) pageURL(start int) string {
	v := url.Values{}
	v.Set("keywords", s.Keywords)
	if s.Location != "" {
		v.Set("location", s.Location)
	}
	setFilter(v, "f_C", s.CompanyID, isNumeric)
	setFilter(v, "f_TPR", s.Recency, inSet(validRecency))
	setFilter(v, "f_WT", s.Arrangement, inSet(validArrangement))
	setFilter(v, "f_E", s.Experience, inSet(validExperience))
	setFilter(v, "f_JT", s.JobType, inSet(validJobType))
	setFilter(v, "geoId", s.GeoID, isNumeric)
	setFilter(v, "f_D", s.Distance, isNumeric)
	setFilter(v, "f_SB2", s.SalaryBand, inSet(validSalaryBand))
	v.Set("start", strconv.Itoa(start))
	return searchURL + "?" + v.Encode()
}

type Config struct {
	Searches []Search
}

type Scraper struct {
	sources.PaginatedBase
	searches []Search
}

var _ sources.Source = (*Scraper)(nil)
var _ sources.DetailFetcher = (*Scraper)(nil)
var _ sources.SnapshotSource = (*Scraper)(nil)

func New(cfg Config) *Scraper {
	return &Scraper{
		PaginatedBase: sources.NewBase(sources.Config{
			Name:     "linkedin",
			UseProxy: true,
		}),
		searches: cfg.Searches,
	}
}

// ponytail: not using PaginatedBase.IteratePages here — it needs a total result
// count up front to compute page count, but seeMoreJobPostings never returns one.
func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	log := slog.With(slog.String("source", "linkedin"))

	for _, search := range s.searches {
		for start := 0; start < maxStart; {
			body, err := s.Get(ctx, search.pageURL(start))
			if err != nil {
				log.Error("page fetch failed", slog.Int("start", start), slog.Any("err", err))
				break
			}

			jobs, err := ParseURLs(bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("linkedin: parse page (start=%d): %w", start, err)
			}
			if len(jobs) == 0 {
				break
			}

			stop, err := fn(ctx, jobs)
			if err != nil || stop {
				return err
			}
			start += len(jobs)

			wait := minWait + time.Duration(rand.Int64N(int64(maxWait-minWait)))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return nil
}

func (s *Scraper) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	if len(s.searches) != 1 {
		return nil, "", fmt.Errorf("linkedin page fetch requires one search")
	}
	start := 0
	if cursor != "" {
		var err error
		start, err = strconv.Atoi(cursor)
		if err != nil || start <= 0 || start >= maxStart {
			return nil, "", fmt.Errorf("invalid linkedin cursor %q", cursor)
		}
	}
	body, err := s.Get(ctx, s.searches[0].pageURL(start))
	if err != nil {
		return nil, "", err
	}
	jobs, err := ParseURLs(bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	if len(jobs) == 0 || start+len(jobs) >= maxStart {
		return jobs, "", nil
	}
	return jobs, strconv.Itoa(start + len(jobs)), nil
}

func (s *Scraper) GetDetails(ctx context.Context, url string) (dto.Job, error) {
	body, err := s.Get(ctx, url)
	if err != nil {
		return dto.Job{}, err
	}
	job, err := ParseJobDetail(bytes.NewReader(body), url)
	if err != nil {
		return dto.Job{}, err
	}
	// LinkedIn's detail page exposes no machine-readable post date (see
	// ParseJobDetail), so UpdatedAt is stamped here rather than in the pure parser —
	// keeps the parser deterministic for snapshot testing.
	if job.UpdatedAt.IsZero() {
		sources.WarnDefaulted("linkedin", "UpdatedAt", url)
		job.UpdatedAt = time.Now().UTC()
	}
	return job, nil
}

func (s *Scraper) ParseURLs(r io.Reader) ([]dto.Job, error) {
	return ParseURLs(r)
}

func (s *Scraper) ParseJobDetail(r io.Reader, url string) (dto.Job, error) {
	return ParseJobDetail(r, url)
}

func ParseURLs(r io.Reader) ([]dto.Job, error) {
	doc, err := sources.ParseHTML(r)
	if err != nil {
		return nil, err
	}

	var jobs []dto.Job
	doc.Find(selCard).Each(func(_ int, card *goquery.Selection) {
		id := cardJobID(card)
		if id == "" {
			return
		}

		title := strings.TrimSpace(card.Find(selCardTitle).First().Text())
		if title == "" {
			title = strings.TrimSpace(card.Find(selCardTitleAlt).First().Text())
		}

		company := strings.TrimSpace(card.Find(selCardCompany).First().Text())
		location := strings.TrimSpace(card.Find(selCardLocation).First().Text())

		jobs = append(jobs, dto.Job{
			Title:       title,
			Location:    location,
			URL:         baseURL + "/jobs/view/" + id,
			CompanySlug: sources.Slugify(company),
		})
	})
	return jobs, nil
}

// cardJobID prefers the structured data-entity-urn attribute (urn:li:jobPosting:{id})
// over parsing the trailing digits off the full-link href — the urn doesn't depend
// on the title-derived slug staying stable.
func cardJobID(card *goquery.Selection) string {
	if urn, ok := card.Attr("data-entity-urn"); ok {
		if i := strings.LastIndex(urn, ":"); i != -1 {
			return urn[i+1:]
		}
	}
	href, ok := card.Find(selCardLink).First().Attr("href")
	if !ok {
		return ""
	}
	href = strings.SplitN(href, "?", 2)[0]
	if i := strings.LastIndex(href, "-"); i != -1 {
		return href[i+1:]
	}
	return ""
}

func ParseJobDetail(r io.Reader, jobURL string) (dto.Job, error) {
	doc, err := sources.ParseHTML(r)
	if err != nil {
		return dto.Job{}, err
	}

	title := strings.TrimSpace(doc.Find(selDetailTitle).First().Text())
	if title == "" {
		return dto.Job{}, fmt.Errorf("title not found (selector: %q)", selDetailTitle)
	}

	company := strings.TrimSpace(doc.Find(selDetailCompany).First().Text())

	// The applicant-count span further down the page reuses topcard__flavor--bullet,
	// so location must come from the first flavor row specifically, not a global match.
	location := strings.TrimSpace(doc.Find(selDetailFlavorRow).First().Find(selDetailFlavorBullet).First().Text())

	descNode := doc.Find(selDescription).First()
	descHTML, _ := descNode.Html()
	descText := descNode.Text()
	if strings.TrimSpace(descHTML) == "" {
		sources.WarnDefaulted("linkedin", "Description", jobURL)
	}
	description := strings.TrimSpace(withCriteria(descHTML, [][2]string{
		{"Seniority level", criteriaText(doc, "Seniority level")},
		{"Employment type", criteriaText(doc, "Employment type")},
		{"Job function", criteriaText(doc, "Job function")},
		{"Industries", criteriaText(doc, "Industries")},
	}))

	salaryRaw := strings.TrimSpace(doc.Find(selSalary).First().Text())
	if salaryRaw == "" {
		salaryRaw = sources.ParseSalaryRaw(descText)
	}

	workArrangement := sources.DetectWorkArrangement(title + " " + location + " " + descText)

	// ponytail: applyURL is best-effort — LinkedIn only sometimes renders the ATS
	// destination for guest requests. When it doesn't, url falls back to the
	// LinkedIn job page itself, which is always deterministic since it's built from
	// data already captured during discovery.
	resolvedURL := jobURL
	if direct := applyURL(doc); direct != "" {
		resolvedURL = direct
	}

	return dto.Job{
		Title:           title,
		Location:        location,
		URL:             resolvedURL,
		CompanySlug:     sources.Slugify(company),
		Source:          "linkedin",
		Description:     description,
		SalaryRaw:       salaryRaw,
		WorkArrangement: workArrangement,
	}, nil
}

// criteriaText returns "" when header isn't present — not every listing renders
// every criterion.
func criteriaText(doc *goquery.Document, header string) string {
	var text string
	doc.Find(selCriteriaItem).EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if !strings.Contains(item.Find(selCriteriaHeader).First().Text(), header) {
			return true
		}
		text = strings.TrimSpace(item.Find(selCriteriaText).First().Text())
		return false
	})
	return text
}

// withCriteria folds seniority/employment-type/job-function/industries into the
// description text since dto.Job has no dedicated columns for them and
// score/claude.go reads Description directly for LLM scoring.
func withCriteria(descriptionHTML string, criteria [][2]string) string {
	var lines []string
	for _, c := range criteria {
		if c[1] != "" {
			lines = append(lines, c[0]+": "+c[1])
		}
	}
	if len(lines) == 0 {
		return descriptionHTML
	}
	return strings.Join(lines, "\n") + "\n\n" + descriptionHTML
}

// applyURL is empty in the common case — LinkedIn only renders code#applyUrl for a
// subset of listings on guest (unauthenticated) requests.
func applyURL(doc *goquery.Document) string {
	code := doc.Find(selApplyURLCode).First()
	if code.Length() == 0 {
		return ""
	}
	html, err := code.Html()
	if err != nil {
		return ""
	}
	m := applyURLRe.FindStringSubmatch(html)
	if m == nil {
		return ""
	}
	decoded, err := url.QueryUnescape(m[1])
	if err != nil {
		return ""
	}
	return decoded
}
