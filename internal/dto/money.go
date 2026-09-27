package dto

// Money is an amount in a currency, e.g. a salary floor.
type Money struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}
