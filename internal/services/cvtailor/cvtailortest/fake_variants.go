package cvtailortest

import (
	"context"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var errVariantNotFound = apperr.NotFound("variant not found")

type variant struct {
	dto.AchievementVariant
	userID string
}

func (f *FakeStore) variantsOf(achievementID string) []dto.AchievementVariant {
	out := []dto.AchievementVariant{}
	for _, v := range f.variants {
		if v.AchievementID == achievementID {
			out = append(out, v.AchievementVariant)
		}
	}
	return out
}

func (f *FakeStore) CreateAchievementVariant(_ context.Context, userID string, in dto.AchievementVariantInput) (dto.AchievementVariant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if a, ok := f.achievements[in.AchievementID]; !ok || a.userID != userID {
		return dto.AchievementVariant{}, errAchievementNotFound
	}
	v := &variant{
		AchievementVariant: dto.AchievementVariant{ID: f.nextID(), AchievementID: in.AchievementID, Text: in.Text, Label: in.Label},
		userID:             userID,
	}
	f.variants = append(f.variants, v)
	return v.AchievementVariant, nil
}

func (f *FakeStore) UpdateAchievementVariant(_ context.Context, userID string, in dto.AchievementVariantInput) (dto.AchievementVariant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range f.variants {
		if v.ID == in.ID && v.userID == userID {
			v.Text, v.Label = in.Text, in.Label
			return v.AchievementVariant, nil
		}
	}
	return dto.AchievementVariant{}, errVariantNotFound
}

func (f *FakeStore) DeleteAchievementVariant(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := len(f.variants)
	f.variants = slices.DeleteFunc(f.variants, func(v *variant) bool { return v.ID == id && v.userID == userID })
	if len(f.variants) == n {
		return errVariantNotFound
	}
	return nil
}
