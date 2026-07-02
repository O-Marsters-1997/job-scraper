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

	UpsertErr error
	ListErr   error
}

func NewMockCompanyProvider() *MockCompanyProvider {
	return &MockCompanyProvider{nextID: 1}
}

func (m *MockCompanyProvider) UpsertCompany(_ context.Context, slug, name, atsSource, atsToken string) (dto.Company, error) {
	if m.UpsertErr != nil {
		return dto.Company{}, m.UpsertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.companies {
		if c.Slug == slug {
			if c.ATSSource == "" {
				m.companies[i].ATSSource = atsSource
			}
			if c.ATSToken == "" {
				m.companies[i].ATSToken = atsToken
			}
			return m.companies[i], nil
		}
	}
	c := dto.Company{
		ID:        fmt.Sprintf("company-%d", m.nextID),
		Slug:      slug,
		Name:      name,
		ATSSource: atsSource,
		ATSToken:  atsToken,
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

func (m *MockCompanyProvider) ListCompaniesForUser(_ context.Context, _ string) ([]dto.Company, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]dto.Company(nil), m.companies...), nil
}
