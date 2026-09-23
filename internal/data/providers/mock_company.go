package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type MockCompanyProvider struct {
	mu        sync.Mutex
	companies []dto.Company
	nextID    int
	tracking  map[string]dto.CompanyTracking

	UpsertErr error
	ListErr   error
}

func NewMockCompanyProvider() *MockCompanyProvider {
	return &MockCompanyProvider{nextID: 1, tracking: make(map[string]dto.CompanyTracking)}
}

func (m *MockCompanyProvider) UpsertCompany(_ context.Context, in dto.CompanyUpsert) (dto.Company, error) {
	if m.UpsertErr != nil {
		return dto.Company{}, m.UpsertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.companies {
		if c.Slug == in.Slug {
			if c.ATSSource == "" {
				m.companies[i].ATSSource = in.ATSSource
			}
			if c.ATSToken == "" {
				m.companies[i].ATSToken = in.ATSToken
			}
			if c.Domain == "" {
				m.companies[i].Domain = in.Domain
			}
			if c.LinkedInCompanyID == "" {
				m.companies[i].LinkedInCompanyID = in.LinkedInCompanyID
			}
			return m.companies[i], nil
		}
	}
	c := dto.Company{
		ID:                fmt.Sprintf("company-%d", m.nextID),
		Slug:              in.Slug,
		Name:              in.Name,
		ATSSource:         in.ATSSource,
		ATSToken:          in.ATSToken,
		Domain:            in.Domain,
		LinkedInCompanyID: in.LinkedInCompanyID,
	}
	m.nextID++
	m.companies = append(m.companies, c)
	return c, nil
}

func (m *MockCompanyProvider) GetCompany(_ context.Context, id string) (dto.Company, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.companies {
		if c.ID == id {
			return c, nil
		}
	}
	return dto.Company{}, ErrNotFound
}

func (m *MockCompanyProvider) ListCompaniesForUser(_ context.Context, userID string) ([]dto.Company, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := append([]dto.Company(nil), m.companies...)
	for i := range out {
		if tracked, ok := m.tracking[userID+":"+out[i].ID]; ok {
			out[i].Tracked = tracked.Enabled
			out[i].CheckIntervalMinutes = tracked.CheckIntervalMinutes
		}
	}
	return out, nil
}

func (m *MockCompanyProvider) SetCompanyTracking(_ context.Context, userID, companyID string, enabled bool, interval int) (dto.CompanyTracking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, company := range m.companies {
		if company.ID == companyID {
			if interval == 0 {
				interval = m.tracking[userID+":"+companyID].CheckIntervalMinutes
				if interval == 0 {
					interval = 360
				}
			}
			tracking := dto.CompanyTracking{CompanyID: companyID, UserID: userID, Enabled: enabled, CheckIntervalMinutes: interval}
			m.tracking[userID+":"+companyID] = tracking
			return tracking, nil
		}
	}
	return dto.CompanyTracking{}, ErrNotFound
}
