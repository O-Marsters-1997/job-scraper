package dto

// Answer is Jev's reply to one choice question about one job.
type Answer struct {
	PYes       float64
	PNo        float64
	PNotStated float64
	Confidence float64
}
