package cvtailortest

import (
	"context"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type bankSkill struct {
	dto.BankSkill
	userID string
}

func (f *FakeStore) ownedSkills(userID string) []*bankSkill {
	var out []*bankSkill
	for _, s := range f.skills {
		if s.userID == userID {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b *bankSkill) int { return a.SortOrder - b.SortOrder })
	return out
}

func (f *FakeStore) nameTaken(userID, name, exceptID string) bool {
	for _, s := range f.ownedSkills(userID) {
		if s.ID != exceptID && strings.EqualFold(s.Name, name) {
			return true
		}
	}
	return false
}

func (f *FakeStore) ListBankSkills(_ context.Context, userID string) ([]dto.BankSkill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.BankSkill{}
	for _, s := range f.ownedSkills(userID) {
		out = append(out, s.BankSkill)
	}
	return out, nil
}

func (f *FakeStore) CreateBankSkill(_ context.Context, userID string, in dto.BankSkillInput) (dto.BankSkill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.nameTaken(userID, in.Name, "") {
		return dto.BankSkill{}, errBankSkillExists
	}
	order := 1
	if owned := f.ownedSkills(userID); len(owned) > 0 {
		order = owned[len(owned)-1].SortOrder + 1
	}
	s := &bankSkill{
		BankSkill: dto.BankSkill{ID: f.nextID(), Name: in.Name, Category: in.Category, SortOrder: order},
		userID:    userID,
	}
	f.skills[s.ID] = s
	return s.BankSkill, nil
}

func (f *FakeStore) UpdateBankSkill(_ context.Context, userID string, in dto.BankSkillInput) (dto.BankSkill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.skills[in.ID]
	if !ok || s.userID != userID {
		return dto.BankSkill{}, errBankSkillNotFound
	}
	if f.nameTaken(userID, in.Name, s.ID) {
		return dto.BankSkill{}, errBankSkillExists
	}
	s.Name, s.Category = in.Name, in.Category
	return s.BankSkill, nil
}

func (f *FakeStore) DeleteBankSkill(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.skills[id]
	if !ok || s.userID != userID {
		return errBankSkillNotFound
	}
	delete(f.skills, id)
	return nil
}

func (f *FakeStore) ReorderBankSkills(_ context.Context, userID string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	owned := f.ownedSkills(userID)
	if !sameSet(ids, len(owned), func(id string) bool { s, ok := f.skills[id]; return ok && s.userID == userID }) {
		return errIncompleteOrder
	}
	for i, id := range ids {
		f.skills[id].SortOrder = i + 1
	}
	return nil
}
