package providers

import (
	"context"
	"time"
)

// MockNotificationProvider lives outside _test.go so it can be imported by tests in other packages.
type MockNotificationProvider struct {
	Digests []DigestRecord

	RecordDigestErr      error
	GetLastDigestSentErr error
	LastDigestTime       time.Time
}

type DigestRecord struct {
	SentAt   time.Time
	JobCount int
}

func NewMockNotificationProvider() *MockNotificationProvider {
	return &MockNotificationProvider{}
}

func (m *MockNotificationProvider) RecordDigest(_ context.Context, sentAt time.Time, jobCount int) error {
	if m.RecordDigestErr != nil {
		return m.RecordDigestErr
	}
	m.Digests = append(m.Digests, DigestRecord{SentAt: sentAt, JobCount: jobCount})
	m.LastDigestTime = sentAt
	return nil
}

func (m *MockNotificationProvider) GetLastDigestSentAt(_ context.Context) (time.Time, error) {
	if m.GetLastDigestSentErr != nil {
		return time.Time{}, m.GetLastDigestSentErr
	}
	return m.LastDigestTime, nil
}
