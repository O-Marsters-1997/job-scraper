// Package cvtailortest is a map-backed fake of cvtailor.Store, proven
// against the real store by RunStoreContract (ADR 0012).
package cvtailortest

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
)

var (
	errPositionNotFound    = apperr.NotFound("position not found")
	errAchievementNotFound = apperr.NotFound("achievement not found")
	errIncompleteOrder     = apperr.Invalid("ids must list every item exactly once")
)

type position struct {
	dto.Position
	userID string
	order  int
}

type achievement struct {
	dto.Achievement
	userID string
	order  int
}

type FakeStore struct {
	mu           sync.Mutex
	positions    map[string]*position
	achievements map[string]*achievement
	seq          int
}

func NewFakeStore() *FakeStore {
	return &FakeStore{positions: map[string]*position{}, achievements: map[string]*achievement{}}
}

func (f *FakeStore) nextID() string {
	f.seq++
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", f.seq)
}

func (f *FakeStore) ownedPositions(userID string) []*position {
	var out []*position
	for _, p := range f.positions {
		if p.userID == userID {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b *position) int { return a.order - b.order })
	return out
}

func (f *FakeStore) ownedAchievements(userID, positionID string) []*achievement {
	var out []*achievement
	for _, a := range f.achievements {
		if a.userID == userID && a.PositionID == positionID {
			out = append(out, a)
		}
	}
	slices.SortFunc(out, func(a, b *achievement) int { return a.order - b.order })
	return out
}

func (f *FakeStore) view(p *position) dto.Position {
	out := p.Position
	out.Achievements = []dto.Achievement{}
	for _, a := range f.ownedAchievements(p.userID, p.ID) {
		out.Achievements = append(out.Achievements, a.Achievement)
	}
	return out
}

func (f *FakeStore) ListPositions(_ context.Context, userID string) ([]dto.Position, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.Position{}
	for _, p := range f.ownedPositions(userID) {
		out = append(out, f.view(p))
	}
	return out, nil
}

func (f *FakeStore) CreatePosition(_ context.Context, userID string, in dto.PositionInput) (dto.Position, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	order := 0
	if owned := f.ownedPositions(userID); len(owned) > 0 {
		order = owned[0].order - 1
	}
	p := &position{
		Position: dto.Position{ID: f.nextID(), Employer: in.Employer, Title: in.Title, StartDate: in.StartDate, EndDate: in.EndDate},
		userID:   userID,
		order:    order,
	}
	f.positions[p.ID] = p
	return f.view(p), nil
}

func (f *FakeStore) UpdatePosition(_ context.Context, userID string, in dto.PositionInput) (dto.Position, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.positions[in.ID]
	if !ok || p.userID != userID {
		return dto.Position{}, errPositionNotFound
	}
	p.Employer, p.Title, p.StartDate, p.EndDate = in.Employer, in.Title, in.StartDate, in.EndDate
	return f.view(p), nil
}

func (f *FakeStore) DeletePosition(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.positions[id]
	if !ok || p.userID != userID {
		return errPositionNotFound
	}
	for aid, a := range f.achievements {
		if a.PositionID == id {
			delete(f.achievements, aid)
		}
	}
	delete(f.positions, id)
	return nil
}

func (f *FakeStore) ReorderPositions(_ context.Context, userID string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	owned := f.ownedPositions(userID)
	if !sameSet(ids, len(owned), func(id string) bool { p, ok := f.positions[id]; return ok && p.userID == userID }) {
		return errIncompleteOrder
	}
	for i, id := range ids {
		f.positions[id].order = i + 1
	}
	return nil
}

func (f *FakeStore) CreateAchievement(_ context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.positions[in.PositionID]
	if !ok || p.userID != userID {
		return dto.Achievement{}, errPositionNotFound
	}
	order := 1
	if owned := f.ownedAchievements(userID, p.ID); len(owned) > 0 {
		order = owned[len(owned)-1].order + 1
	}
	a := &achievement{
		Achievement: dto.Achievement{ID: f.nextID(), PositionID: p.ID, Text: in.Text},
		userID:      userID,
		order:       order,
	}
	f.achievements[a.ID] = a
	return a.Achievement, nil
}

func (f *FakeStore) UpdateAchievement(_ context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.achievements[in.ID]
	if !ok || a.userID != userID {
		return dto.Achievement{}, errAchievementNotFound
	}
	a.Text = in.Text
	return a.Achievement, nil
}

func (f *FakeStore) DeleteAchievement(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.achievements[id]
	if !ok || a.userID != userID {
		return errAchievementNotFound
	}
	delete(f.achievements, id)
	return nil
}

func (f *FakeStore) ReorderAchievements(_ context.Context, userID, positionID string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.positions[positionID]; !ok || p.userID != userID {
		return errPositionNotFound
	}
	owned := f.ownedAchievements(userID, positionID)
	if !sameSet(ids, len(owned), func(id string) bool {
		a, ok := f.achievements[id]
		return ok && a.userID == userID && a.PositionID == positionID
	}) {
		return errIncompleteOrder
	}
	for i, id := range ids {
		f.achievements[id].order = i + 1
	}
	return nil
}

func (f *FakeStore) ImportPositions(ctx context.Context, userID string, in []dto.ImportPosition) ([]dto.Position, error) {
	out := make([]dto.Position, len(in))
	for i := len(in) - 1; i >= 0; i-- {
		p := in[i]
		created, err := f.CreatePosition(ctx, userID, dto.PositionInput{
			Employer: p.Employer, Title: p.Title, StartDate: p.StartDate, EndDate: p.EndDate,
		})
		if err != nil {
			return nil, err
		}
		out[i] = created
		for _, text := range p.Achievements {
			a, err := f.CreateAchievement(ctx, userID, dto.AchievementInput{PositionID: created.ID, Text: text})
			if err != nil {
				return nil, err
			}
			out[i].Achievements = append(out[i].Achievements, a)
		}
	}
	return out, nil
}

func sameSet(ids []string, want int, member func(string) bool) bool {
	if len(ids) != want {
		return false
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, dup := seen[id]; dup || !member(id) {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

var _ cvtailor.Store = (*FakeStore)(nil)
