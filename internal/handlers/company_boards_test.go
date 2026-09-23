package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type boardVerifierFunc func(source, token string) error

func (f boardVerifierFunc) Verify(_ context.Context, source, token string) error {
	return f(source, token)
}

func TestCompanyBoardConfirmation(t *testing.T) {
	companies := providers.NewMockCompanyProvider()
	company, err := companies.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	verifyErr := errors.New("board unavailable")
	h := NewCompaniesHandler(companies, providers.NewMockSourceTargetProvider(), boardVerifierFunc(func(source, token string) error {
		return verifyErr
	}))
	post := func(body string) dto.CompanyBoard {
		t.Helper()
		r := withRouteID(withSession(httptest.NewRequest(http.MethodPost, "/companies/"+company.ID+"/boards", bytes.NewBufferString(body)), "user-1"), company.ID)
		w := httptest.NewRecorder()
		h.AddBoard(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("post: %d %s", w.Code, w.Body.String())
		}
		var board dto.CompanyBoard
		if err := json.Unmarshal(w.Body.Bytes(), &board); err != nil {
			t.Fatal(err)
		}
		return board
	}
	board := post(`{"url":"https://boards.greenhouse.io/acme"}`)
	if board.Status != dto.BoardCandidate {
		t.Fatalf("unconfirmed board must stay candidate: %+v", board)
	}
	board = post(`{"url":"https://boards.greenhouse.io/acme","confirm":true}`)
	if board.Status != dto.BoardCandidate {
		t.Fatalf("failed fetch must stay candidate: %+v", board)
	}
	verifyErr = nil
	board = post(`{"url":"https://boards.greenhouse.io/acme","confirm":true}`)
	if board.Status != dto.BoardVerified || board.VerificationMethod != "user_confirmed" {
		t.Fatalf("successful retry must verify: %+v", board)
	}
	boards, err := companies.ListCompanyBoards(t.Context(), company.ID)
	if err != nil || len(boards) != 1 {
		t.Fatalf("want one board: %+v %v", boards, err)
	}
	r := withRouteID(withSession(httptest.NewRequest(http.MethodGet, "/companies/"+company.ID+"/boards", nil), "user-1"), company.ID)
	w := httptest.NewRecorder()
	h.ListBoards(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	var listed []dto.CompanyBoard
	if err := json.Unmarshal(w.Body.Bytes(), &listed); err != nil || len(listed) != 1 || listed[0].Status != dto.BoardVerified {
		t.Fatalf("listed: %+v %v", listed, err)
	}
}

func TestCompanyCreateOnlyAddsCandidateBoard(t *testing.T) {
	companies := providers.NewMockCompanyProvider()
	targets := providers.NewMockSourceTargetProvider()
	h := NewCompaniesHandler(companies, targets, nil)
	r := withSession(httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(`{"url":"https://boards.greenhouse.io/acme"}`)), "user-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var company dto.Company
	if err := json.Unmarshal(w.Body.Bytes(), &company); err != nil {
		t.Fatal(err)
	}
	boards, err := companies.ListCompanyBoards(t.Context(), company.ID)
	if err != nil || len(boards) != 1 || boards[0].Status != dto.BoardCandidate {
		t.Fatalf("want candidate: %+v %v", boards, err)
	}
	listed, err := targets.ListSourceTargetsByUser(t.Context(), "user-1")
	if err != nil || len(listed) != 0 {
		t.Fatalf("candidate must not schedule legacy target: %+v %v", listed, err)
	}
}
