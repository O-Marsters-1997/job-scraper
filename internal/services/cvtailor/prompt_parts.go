package cvtailor

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/promptparts"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

const maxPartRunes = 20000

func (m *Module) resolvedParts(ctx context.Context, userID string, g promptparts.Generation) ([]promptparts.Resolved, error) {
	overrides, err := m.store.ListPartOverrides(ctx, userID, g.Name)
	if err != nil {
		return nil, err
	}
	return promptparts.Resolve(g, overrides), nil
}

func findPart(generation, path string) (promptparts.Generation, promptparts.Part, error) {
	g := profiledraft.Generation
	if string(g.Name) == generation {
		for _, p := range g.Parts {
			if p.Path == path {
				return g, p, nil
			}
		}
	}
	return promptparts.Generation{}, promptparts.Part{}, apperr.NotFound("prompt file not found")
}

func (m *Module) SavePromptPart(ctx context.Context, userID string, in dto.PromptPartInput) (dto.PromptPart, error) {
	g, part, err := findPart(in.Generation, in.Path)
	if err != nil {
		return dto.PromptPart{}, err
	}
	if n := utf8.RuneCountInString(in.Text); n == 0 || n > maxPartRunes {
		return dto.PromptPart{}, apperr.Invalid(fmt.Sprintf("a prompt file must be 1 to %d characters", maxPartRunes))
	}
	o := promptparts.Override{Path: part.Path, Text: in.Text, BaseDefault: part.Default}
	if err := m.store.SavePartOverride(ctx, userID, g.Name, o); err != nil {
		return dto.PromptPart{}, err
	}
	return dto.PromptPart{Path: o.Path, Text: o.Text, Source: string(promptparts.SourceYours)}, nil
}

func (m *Module) ResetPromptPart(ctx context.Context, userID string, ref dto.PromptPartRef) (struct{}, error) {
	g, part, err := findPart(ref.Generation, ref.Path)
	if err != nil {
		return struct{}{}, err
	}
	return struct{}{}, m.store.DeletePartOverride(ctx, userID, g.Name, part.Path)
}
