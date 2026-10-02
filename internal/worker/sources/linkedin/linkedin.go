package linkedin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	baseURL   = "https://www.linkedin.com"
	searchURL = baseURL + "/jobs-guest/jobs/api/seeMoreJobPostings/search"
	detailURL = baseURL + "/jobs-guest/jobs/api/jobPosting/"

	// maxStart bounds pagination — seeMoreJobPostings never returns a total result
	// count, so an empty page is the only end-of-results signal LinkedIn gives us.
	// LinkedIn also tends to start 429ing a given IP after ~page 10; the proxy
	// (Route below) mitigates that, and this cap bounds the damage if it doesn't.
	maxStart = 1000

	selCard         = `div.base-search-card`
	selCardLink     = `a.base-card__full-link`
	selCardTitle    = `h3.base-search-card__title`
	selCardTitleAlt = `span.sr-only`
	selCardCompany  = `h4.base-search-card__subtitle a`
	selCardLocation = `span.job-search-card__location`

	selDetailTitle        = `.top-card-layout__title.topcard__title`
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

func pageURL(keywords string, filters map[string]string, recency string, start int) string {
	v := url.Values{}
	v.Set("keywords", keywords)
	fields, _ := sourcespec.LookupFilterFields("linkedin")
	for _, f := range fields {
		value := filters[f.Name]
		if value == "" {
			continue
		}
		if !sourcespec.ValidFilterValue("linkedin", f.Name, value) {
			slog.Warn("linkedin: dropping invalid filter value", slog.String("param", f.Param), slog.String("value", value))
			continue
		}
		v.Set(f.Param, value)
	}
	if recency != "" {
		v.Set("f_TPR", recency)
	}
	v.Set("start", strconv.Itoa(start))
	return searchURL + "?" + v.Encode()
}

type Scraper struct {
	sources.PaginatedBase
	keywords string
	filters  map[string]string
	recency  string
}

const recencyMargin = time.Hour

// Recency returns the f_TPR value that fetches only what appeared since lastSucceeded,
// padded by an hour and capped at configured (the Target's own r<seconds> filter).
// It returns "" when the Target's stored recency should be used unchanged.
func Recency(configured string, lastSucceeded *time.Time, now time.Time) string {
	if lastSucceeded == nil {
		return ""
	}
	seconds := int64((now.Sub(*lastSucceeded) + recencyMargin).Seconds())
	if seconds < int64(recencyMargin.Seconds()) {
		seconds = int64(recencyMargin.Seconds())
	}
	if n, err := strconv.ParseInt(strings.TrimPrefix(configured, "r"), 10, 64); err == nil && seconds >= n {
		return ""
	}
	return "r" + strconv.FormatInt(seconds, 10)
}

var _ sources.Source = (*Scraper)(nil)
var _ sources.DetailFetcher = (*Scraper)(nil)
var _ sources.SnapshotSource = (*Scraper)(nil)

func New(keywords string, filters map[string]string, recency string) *Scraper {
	return &Scraper{
		PaginatedBase: sources.NewBase(sources.Config{
			Name:  "linkedin",
			Route: sources.RouteTiered,
		}),
		keywords: keywords,
		filters:  filters,
		recency:  recency,
	}
}

func (s *Scraper) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	start := 0
	if cursor != "" {
		var err error
		start, err = strconv.Atoi(cursor)
		if err != nil || start <= 0 || start >= maxStart {
			return nil, "", fmt.Errorf("invalid linkedin cursor %q", cursor)
		}
	}
	body, err := s.Get(ctx, pageURL(s.keywords, s.filters, s.recency, start))
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
	body, err := s.Get(ctx, fragmentURL(url))
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

func fragmentURL(jobURL string) string {
	return detailURL + path.Base(jobURL)
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
			CompanySlug: slug.Make(company),
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
		CompanySlug:     slug.Make(company),
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
