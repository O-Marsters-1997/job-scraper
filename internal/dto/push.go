package dto

type PushKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// PushSubscriptionInput is a browser PushSubscription as its toJSON emits it.
type PushSubscriptionInput struct {
	Endpoint string   `json:"endpoint"`
	Keys     PushKeys `json:"keys"`
}

type PushEndpointInput struct {
	Endpoint string `json:"endpoint"`
}

type PushMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag"`
}

type VAPIDKey struct {
	Key string `json:"key"`
}
