package db_test

import (
	"context"
	"testing"
	"time"
)

func truncateCompanies(t *testing.T) {
	t.Helper()
	if _, err := testDB.Pool().Exec(context.Background(), "TRUNCATE companies, source_targets, users CASCADE"); err != nil {
		t.Fatalf("truncate companies: %v", err)
	}
}

func TestUpsertCompany(t *testing.T) {
	ctx := context.Background()

	t.Run("insert", func(t *testing.T) {
		truncateCompanies(t)

		c, err := testDB.UpsertCompany(ctx, "acme", "Acme", "greenhouse", "acme")
		if err != nil {
			t.Fatalf("UpsertCompany: %v", err)
		}
		if c.Slug != "acme" || c.Name != "Acme" || c.ATSSource != "greenhouse" || c.ATSToken != "acme" {
			t.Errorf("unexpected company: %+v", c)
		}
	})

	t.Run("conflict fills in missing ats fields", func(t *testing.T) {
		truncateCompanies(t)

		first, err := testDB.UpsertCompany(ctx, "acme", "Acme", "", "")
		if err != nil {
			t.Fatalf("first UpsertCompany: %v", err)
		}
		if first.ATSSource != "" {
			t.Fatalf("expected empty ats source on discovery-first insert, got %q", first.ATSSource)
		}

		second, err := testDB.UpsertCompany(ctx, "acme", "Acme Corp", "greenhouse", "acme")
		if err != nil {
			t.Fatalf("second UpsertCompany: %v", err)
		}
		if second.ID != first.ID {
			t.Errorf("expected same row, got different IDs")
		}
		if second.ATSSource != "greenhouse" || second.ATSToken != "acme" {
			t.Errorf("expected ats fields filled in, got source=%q token=%q", second.ATSSource, second.ATSToken)
		}
	})

	t.Run("conflict never overwrites an existing ats board", func(t *testing.T) {
		truncateCompanies(t)

		if _, err := testDB.UpsertCompany(ctx, "acme", "Acme", "greenhouse", "acme"); err != nil {
			t.Fatalf("first UpsertCompany: %v", err)
		}

		got, err := testDB.UpsertCompany(ctx, "acme", "Acme", "lever", "acme-other")
		if err != nil {
			t.Fatalf("second UpsertCompany: %v", err)
		}
		if got.ATSSource != "greenhouse" || got.ATSToken != "acme" {
			t.Errorf("expected original ats board preserved, got source=%q token=%q", got.ATSSource, got.ATSToken)
		}
	})
}

func TestListCompaniesForUser(t *testing.T) {
	ctx := context.Background()
	truncateCompanies(t)

	user, err := testDB.CreateUser(ctx, "alice", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	tracked, err := testDB.UpsertCompany(ctx, "acme", "Acme", "greenhouse", "acme")
	if err != nil {
		t.Fatalf("UpsertCompany tracked: %v", err)
	}
	if _, err := testDB.UpsertCompany(ctx, "widgetco", "Widgetco", "ashby", "widgetco"); err != nil {
		t.Fatalf("UpsertCompany untracked: %v", err)
	}

	// Pre-existing target created before this feature (no company_id) must still
	// join to the company via (source, value) and show as tracked.
	if _, err := testDB.CreateSourceTarget(ctx, user.ID, "greenhouse", "acme", true, nil); err != nil {
		t.Fatalf("CreateSourceTarget: %v", err)
	}

	companies, err := testDB.ListCompaniesForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListCompaniesForUser: %v", err)
	}
	if len(companies) != 2 {
		t.Fatalf("want 2 companies, got %d", len(companies))
	}

	for _, c := range companies {
		if c.ID == tracked.ID {
			if !c.Tracked {
				t.Errorf("expected acme to show as tracked via (source, value) join")
			}
			continue
		}
		if c.Tracked {
			t.Errorf("expected widgetco to show as untracked")
		}
	}
}

func TestListDueSourceTargets(t *testing.T) {
	ctx := context.Background()
	truncateCompanies(t)

	user, err := testDB.CreateUser(ctx, "bob", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	fresh, err := testDB.CreateSourceTarget(ctx, user.ID, "greenhouse", "fresh", true, nil)
	if err != nil {
		t.Fatalf("CreateSourceTarget fresh: %v", err)
	}
	neverChecked, err := testDB.CreateSourceTarget(ctx, user.ID, "ashby", "never", true, nil)
	if err != nil {
		t.Fatalf("CreateSourceTarget never-checked: %v", err)
	}
	stale, err := testDB.CreateSourceTarget(ctx, user.ID, "lever", "stale", true, nil)
	if err != nil {
		t.Fatalf("CreateSourceTarget stale: %v", err)
	}
	if _, err := testDB.CreateSourceTarget(ctx, user.ID, "workable", "disabled", false, nil); err != nil {
		t.Fatalf("CreateSourceTarget disabled: %v", err)
	}

	// fresh: checked 1 minute ago, interval 360 (default) — not due.
	if _, err := testDB.Pool().Exec(ctx,
		"UPDATE source_targets SET last_checked_at = NOW() - interval '1 minute' WHERE id = $1", fresh.ID); err != nil {
		t.Fatalf("set fresh last_checked_at: %v", err)
	}
	// stale: checked 7 hours ago, interval 360 (6h) — due.
	if _, err := testDB.Pool().Exec(ctx,
		"UPDATE source_targets SET last_checked_at = NOW() - interval '7 hours' WHERE id = $1", stale.ID); err != nil {
		t.Fatalf("set stale last_checked_at: %v", err)
	}

	due, err := testDB.ListDueSourceTargets(ctx)
	if err != nil {
		t.Fatalf("ListDueSourceTargets: %v", err)
	}

	dueIDs := make(map[string]bool, len(due))
	for _, target := range due {
		dueIDs[target.ID] = true
	}
	if dueIDs[fresh.ID] {
		t.Errorf("fresh target should not be due")
	}
	if !dueIDs[neverChecked.ID] {
		t.Errorf("never-checked target should be due")
	}
	if !dueIDs[stale.ID] {
		t.Errorf("stale target should be due")
	}
	for id := range dueIDs {
		if id != neverChecked.ID && id != stale.ID {
			t.Errorf("unexpected target in due set: %s", id)
		}
	}
}

func TestTouchSourceTargetsChecked(t *testing.T) {
	ctx := context.Background()
	truncateCompanies(t)

	alice, err := testDB.CreateUser(ctx, "alice2", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	bob, err := testDB.CreateUser(ctx, "bob2", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}

	if _, err := testDB.CreateSourceTarget(ctx, alice.ID, "greenhouse", "acme", true, nil); err != nil {
		t.Fatalf("CreateSourceTarget alice: %v", err)
	}
	if _, err := testDB.CreateSourceTarget(ctx, bob.ID, "greenhouse", "acme", true, nil); err != nil {
		t.Fatalf("CreateSourceTarget bob: %v", err)
	}

	if err := testDB.TouchSourceTargetsChecked(ctx, "greenhouse", "acme"); err != nil {
		t.Fatalf("TouchSourceTargetsChecked: %v", err)
	}

	for _, uid := range []string{alice.ID, bob.ID} {
		targets, err := testDB.ListSourceTargetsByUser(ctx, uid)
		if err != nil {
			t.Fatalf("ListSourceTargetsByUser(%s): %v", uid, err)
		}
		if len(targets) != 1 || targets[0].LastCheckedAt == nil {
			t.Errorf("expected user %s's target to have last_checked_at set", uid)
			continue
		}
		if time.Since(*targets[0].LastCheckedAt) > time.Minute {
			t.Errorf("last_checked_at not recent for user %s", uid)
		}
	}
}

func TestListUserIDsForTarget(t *testing.T) {
	ctx := context.Background()
	truncateCompanies(t)

	alice, err := testDB.CreateUser(ctx, "alice3", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	bob, err := testDB.CreateUser(ctx, "bob3", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}

	if _, err := testDB.CreateSourceTarget(ctx, alice.ID, "greenhouse", "acme", true, nil); err != nil {
		t.Fatalf("CreateSourceTarget alice ats: %v", err)
	}
	if _, err := testDB.CreateSourceTarget(ctx, bob.ID, "wis", "engineer", true, nil); err != nil {
		t.Fatalf("CreateSourceTarget bob wis: %v", err)
	}

	t.Run("exact match", func(t *testing.T) {
		ids, err := testDB.ListUserIDsForTarget(ctx, "greenhouse", "acme")
		if err != nil {
			t.Fatalf("ListUserIDsForTarget: %v", err)
		}
		if len(ids) != 1 || ids[0] != alice.ID {
			t.Errorf("want [alice], got %v", ids)
		}
	})

	t.Run("any value", func(t *testing.T) {
		ids, err := testDB.ListUserIDsForTarget(ctx, "wis", "")
		if err != nil {
			t.Fatalf("ListUserIDsForTarget: %v", err)
		}
		if len(ids) != 1 || ids[0] != bob.ID {
			t.Errorf("want [bob], got %v", ids)
		}
	})
}

func TestUpsertSourceTargetForCompany(t *testing.T) {
	ctx := context.Background()
	truncateCompanies(t)

	user, err := testDB.CreateUser(ctx, "carol", "hash", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	company, err := testDB.UpsertCompany(ctx, "acme", "Acme", "greenhouse", "acme")
	if err != nil {
		t.Fatalf("UpsertCompany: %v", err)
	}

	created, err := testDB.UpsertSourceTargetForCompany(ctx, user.ID, "greenhouse", "acme", company.ID, true)
	if err != nil {
		t.Fatalf("UpsertSourceTargetForCompany create: %v", err)
	}
	if !created.Enabled || created.CompanyID != company.ID {
		t.Fatalf("unexpected created target: %+v", created)
	}

	disabled, err := testDB.UpsertSourceTargetForCompany(ctx, user.ID, "greenhouse", "acme", company.ID, false)
	if err != nil {
		t.Fatalf("UpsertSourceTargetForCompany disable: %v", err)
	}
	if disabled.ID != created.ID {
		t.Errorf("expected same target row on toggle, got different ID")
	}
	if disabled.Enabled {
		t.Errorf("expected target to be disabled")
	}
}
