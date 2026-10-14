// Package salary parses free-text salary strings into numeric bounds.
package salary

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	kSuffix = regexp.MustCompile(`(?i)\b(\d+(?:\.\d+)?)k\b`)
	number  = regexp.MustCompile(`\d+(?:\.\d+)?`)
	symbols = strings.NewReplacer("£", "", "$", "", "€", "", ",", "")
)

// Parse returns the first two numbers in raw as min and max, expanding "k"
// suffixes; a single number is both bounds. ok is false when raw has no number.
func Parse(raw string) (lo, hi float64, ok bool) {
	normalised := kSuffix.ReplaceAllStringFunc(symbols.Replace(raw), func(m string) string {
		n, err := strconv.ParseFloat(m[:len(m)-1], 64)
		if err != nil {
			return m
		}
		return strconv.FormatFloat(n*1000, 'f', -1, 64)
	})
	nums := number.FindAllString(normalised, 2)
	if len(nums) == 0 {
		return 0, 0, false
	}
	lo, err := strconv.ParseFloat(nums[0], 64)
	if err != nil || math.IsInf(lo, 0) {
		return 0, 0, false
	}
	hi = lo
	if len(nums) == 2 {
		if hi, err = strconv.ParseFloat(nums[1], 64); err != nil || math.IsInf(hi, 0) {
			return 0, 0, false
		}
	}
	return lo, hi, true
}
