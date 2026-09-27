package docref

import (
	"fmt"
	"net/url"
	"strings"
)

func ParseDocID(urlOrID string) (string, error) {
	if strings.Contains(urlOrID, "/document/d/") {
		u, err := url.Parse(urlOrID)
		if err != nil {
			return "", fmt.Errorf("invalid Google Docs URL or ID: %s", urlOrID)
		}
		segments := strings.Split(u.Path, "/")
		for i, seg := range segments {
			if seg == "d" && i+1 < len(segments) && segments[i+1] != "" {
				return segments[i+1], nil
			}
		}
		return "", fmt.Errorf("invalid Google Docs URL or ID: %s", urlOrID)
	}
	if !strings.Contains(urlOrID, "/") && len(urlOrID) >= 10 {
		return urlOrID, nil
	}
	return "", fmt.Errorf("invalid Google Docs URL or ID: %s", urlOrID)
}
