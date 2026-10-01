package wttj

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	jobPrefix     = "https://app.welcometothejungle.com/jobs/"
	companyPrefix = "https://app.welcometothejungle.com/companies/"
)

func parseSitemap(r io.Reader) (jobIDs, companyNames []string, err error) {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return jobIDs, companyNames, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("parse sitemap: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "loc" {
			continue
		}
		var loc string
		if err := dec.DecodeElement(&loc, &start); err != nil {
			return nil, nil, fmt.Errorf("parse sitemap: %w", err)
		}
		if id, ok := strings.CutPrefix(loc, jobPrefix); ok {
			jobIDs = append(jobIDs, id)
		} else if name, ok := strings.CutPrefix(loc, companyPrefix); ok {
			companyNames = append(companyNames, name)
		}
	}
}
