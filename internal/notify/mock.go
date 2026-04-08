package notify

import "context"

// MockNotifier lives outside _test.go so it can be imported by tests in other packages.
type MockNotifier struct {
	Sent []SentEmail

	SendErr error
}

type SentEmail struct {
	To      string
	Subject string
	HTML    string
}

func (m *MockNotifier) Send(_ context.Context, to, subject, htmlBody string) error {
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Sent = append(m.Sent, SentEmail{To: to, Subject: subject, HTML: htmlBody})
	return nil
}
