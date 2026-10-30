package cvtailortest

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

type outreachTemplate struct {
	dto.OutreachTemplate
	userID string
}

func (f *FakeStore) ownedTemplates(userID string, kind dto.OutreachKind) []*outreachTemplate {
	var out []*outreachTemplate
	for _, t := range f.templates {
		if t.userID == userID && (kind == "" || t.Kind == kind) {
			out = append(out, t)
		}
	}
	return out
}

func (f *FakeStore) SeedOutreachTemplates(_ context.Context, _ pgx.Tx, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range store.DefaultOutreachTemplates() {
		id := f.nextID()
		f.templates = append(f.templates, &outreachTemplate{userID: userID, OutreachTemplate: dto.OutreachTemplate{ID: id, Kind: d.Kind, Name: d.Name, Body: d.Body, IsDefault: true}})
	}
	return nil
}

func (f *FakeStore) ListOutreachTemplates(_ context.Context, userID string) ([]dto.OutreachTemplate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.OutreachTemplate{}
	for _, t := range f.ownedTemplates(userID, "") {
		out = append(out, t.OutreachTemplate)
	}
	return out, nil
}

func (f *FakeStore) findTemplate(userID, id string) (*outreachTemplate, error) {
	for _, t := range f.ownedTemplates(userID, "") {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, store.ErrTemplateNotFound
}

func (f *FakeStore) templateNameTaken(userID string, kind dto.OutreachKind, name, exceptID string) bool {
	for _, t := range f.ownedTemplates(userID, kind) {
		if t.Name == name && t.ID != exceptID {
			return true
		}
	}
	return false
}

func (f *FakeStore) CreateOutreachTemplate(_ context.Context, userID string, in dto.OutreachTemplateInput) (dto.OutreachTemplate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.templateNameTaken(userID, in.Kind, in.Name, "") {
		return dto.OutreachTemplate{}, store.ErrTemplateNameTaken
	}
	t := &outreachTemplate{userID: userID, OutreachTemplate: dto.OutreachTemplate{
		ID: f.nextID(), Kind: in.Kind, Name: in.Name, Body: in.Body, IsDefault: len(f.ownedTemplates(userID, in.Kind)) == 0,
	}}
	f.templates = append(f.templates, t)
	return t.OutreachTemplate, nil
}

func (f *FakeStore) UpdateOutreachTemplate(_ context.Context, userID string, in dto.OutreachTemplateInput) (dto.OutreachTemplate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, err := f.findTemplate(userID, in.ID)
	if err != nil {
		return dto.OutreachTemplate{}, err
	}
	if f.templateNameTaken(userID, t.Kind, in.Name, t.ID) {
		return dto.OutreachTemplate{}, store.ErrTemplateNameTaken
	}
	t.Name, t.Body = in.Name, in.Body
	return t.OutreachTemplate, nil
}

func (f *FakeStore) MakeOutreachTemplateDefault(_ context.Context, userID, id string) (dto.OutreachTemplate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, err := f.findTemplate(userID, id)
	if err != nil {
		return dto.OutreachTemplate{}, err
	}
	for _, o := range f.ownedTemplates(userID, t.Kind) {
		o.IsDefault = o == t
	}
	return t.OutreachTemplate, nil
}

func (f *FakeStore) DeleteOutreachTemplate(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, err := f.findTemplate(userID, id)
	if err != nil {
		return err
	}
	if t.IsDefault && len(f.ownedTemplates(userID, t.Kind)) > 1 {
		return store.ErrTemplateDefaultSet
	}
	for i, o := range f.templates {
		if o == t {
			f.templates = append(f.templates[:i], f.templates[i+1:]...)
			break
		}
	}
	return nil
}
