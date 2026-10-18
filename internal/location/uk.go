package location

import "regexp"

var (
	ukName = regexp.MustCompile(`(?i)\b(?:united kingdom|uk|u\.k\.|great britain|england|scotland|northern ireland|wales)\b`)
	ukCity = regexp.MustCompile(`(?i)\b(?:london|manchester|edinburgh|glasgow|bristol|birmingham|leeds|cambridge|oxford|belfast|cardiff|reading|brighton|liverpool|newcastle|sheffield|nottingham)\b`)

	newSouthWales = regexp.MustCompile(`(?i)\bnew south wales\b`)
	usStateSuffix = regexp.MustCompile(`,\s*[A-Z]{2}\b`)
	foreignName   = regexp.MustCompile(`(?i)\b(?:usa|united states|canada|ontario|australia|nsw|massachusetts|ohio|kentucky|pennsylvania|connecticut|new hampshire)\b`)
)

// IsUK reports whether a free-text job location is in the UK, rejecting
// same-named places elsewhere (Cambridge MA, London ON, New South Wales).
func IsUK(location string) bool {
	if ukName.MatchString(newSouthWales.ReplaceAllString(location, "")) {
		return true
	}
	return ukCity.MatchString(location) && !usStateSuffix.MatchString(location) && !foreignName.MatchString(location)
}
