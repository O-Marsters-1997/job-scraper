package dto

import "net/http"

type CachedResponse struct {
	URL    string
	Status int
	Header http.Header
	Body   []byte
}
