package suitability

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	amountPattern       = regexp.MustCompile(`(?i)\d[\d,]*(?:\.\d+)?\s*k?`)
	currencyCodePattern = regexp.MustCompile(`\b(GBP|USD|EUR|CAD|AUD|NZD|CHF)\b`)
	dayRatePattern      = regexp.MustCompile(`(?i)per\s*day|/\s*day|a\s*day|daily|day\s*rate`)
	currencySymbols     = map[string]string{"£": "GBP", "$": "USD", "€": "EUR"}
)

func parseSalary(raw string) (amount int, currency string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || dayRatePattern.MatchString(raw) {
		return 0, "", false
	}

	currency = detectCurrency(raw)
	if currency == "" {
		return 0, "", false
	}

	for _, m := range amountPattern.FindAllString(raw, -1) {
		if v, ok := parseAmount(m); ok && v > amount {
			amount = v
		}
	}
	if amount == 0 {
		return 0, "", false
	}
	return amount, currency, true
}

func detectCurrency(raw string) string {
	for symbol, code := range currencySymbols {
		if strings.Contains(raw, symbol) {
			return code
		}
	}
	return currencyCodePattern.FindString(strings.ToUpper(raw))
}

func parseAmount(m string) (int, bool) {
	m = strings.TrimSpace(m)
	thousands := strings.HasSuffix(strings.ToLower(m), "k")
	if thousands {
		m = strings.TrimSpace(m[:len(m)-1])
	}
	m = strings.ReplaceAll(m, ",", "")
	v, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, false
	}
	if thousands {
		v *= 1000
	}
	return int(v), true
}

func salaryPick(floor dto.Money, salaryRaw string) evaluatedPick {
	answer := dto.Answer{PNotStated: 1}
	if amount, currency, ok := parseSalary(salaryRaw); ok && currency == floor.Currency {
		if amount < floor.Amount {
			answer = dto.Answer{PYes: 1}
		} else {
			answer = dto.Answer{PNo: 1}
		}
	}
	return evaluatedPick{key: "salary", label: "Salary", stance: "avoid", known: true, answer: answer}
}
