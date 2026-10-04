package discover_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

func TestResolveDomain(t *testing.T) {
	tests := []struct {
		name      string
		pages     map[string]string
		want      discover.Board
		wantFound bool
		wantGets  int
	}{
		{
			name: "direct link",
			pages: map[string]string{
				"https://acme.com": `<a href="https://boards.greenhouse.io/acme">Jobs</a>`,
			},
			want:      discover.Board{Source: "greenhouse", Token: "acme"},
			wantFound: true,
			wantGets:  1,
		},
		{
			name: "iframe embed",
			pages: map[string]string{
				"https://acme.com": `<iframe src="https://jobs.lever.co/acme"></iframe>`,
			},
			want:      discover.Board{Source: "lever", Token: "acme"},
			wantFound: true,
			wantGets:  1,
		},
		{
			name: "sniff only",
			pages: map[string]string{
				"https://acme.com": `<script src="https://boards.greenhouse.io/embed/job_board?for=acme"></script>`,
			},
			want:      discover.Board{Source: "greenhouse", Token: "acme"},
			wantFound: true,
			wantGets:  1,
		},
		{
			name: "careers hop",
			pages: map[string]string{
				"https://acme.com":         `<a href="/careers">Careers</a>`,
				"https://acme.com/careers": `<a href="https://jobs.ashbyhq.com/acme">Apply</a>`,
			},
			want:      discover.Board{Source: "ashby", Token: "acme"},
			wantFound: true,
			wantGets:  2,
		},
		{
			name: "nothing found",
			pages: map[string]string{
				"https://acme.com":         `<a href="/careers">Careers</a>`,
				"https://acme.com/careers": `<p>Email us</p>`,
			},
			wantGets: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gets := 0
			get := func(_ context.Context, url string) ([]byte, error) {
				gets++
				page, ok := tt.pages[url]
				if !ok {
					return nil, fmt.Errorf("unexpected GET %s", url)
				}
				return []byte(page), nil
			}

			got, found, err := discover.ResolveDomain(t.Context(), get, "acme.com")
			if err != nil {
				t.Fatal(err)
			}

			if found != tt.wantFound {
				t.Errorf("ResolveDomain found = %v, want %v", found, tt.wantFound)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ResolveDomain board (-want +got):\n%s", diff)
			}
			if gets != tt.wantGets {
				t.Errorf("ResolveDomain made %d GETs, want %d", gets, tt.wantGets)
			}
		})
	}
}
