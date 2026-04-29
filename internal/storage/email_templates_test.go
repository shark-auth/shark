package storage_test

import (
	"context"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

func TestSeedEmailTemplates_Idempotent(t *testing.T) {
	s := testutil.NewTestDB(t)
	ctx := context.Background()

	if err := s.SeedEmailTemplates(ctx); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := s.SeedEmailTemplates(ctx); err != nil {
		t.Fatalf("second seed (should be idempotent): %v", err)
	}

	list, err := s.ListEmailTemplates(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 5 {
		t.Fatalf("expected 5 seed templates, got %d", len(list))
	}

	wantIDs := map[string]bool{
		"magic_link": true, "password_reset": true, "verify_email": true,
		"organization_invitation": true, "welcome": true,
	}
	for _, tpl := range list {
		delete(wantIDs, tpl.ID)
	}
	if len(wantIDs) > 0 {
		t.Errorf("missing templates: %v", wantIDs)
	}
}

func TestUpdateEmailTemplate_PreservesUnchanged(t *testing.T) {
	s := testutil.NewTestDB(t)
	ctx := context.Background()
	if err := s.SeedEmailTemplates(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := s.UpdateEmailTemplate(ctx, "magic_link", map[string]any{
		"subject": "Custom subject",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := s.GetEmailTemplate(ctx, "magic_link")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Subject != "Custom subject" {
		t.Errorf("subject not persisted: %q", got.Subject)
	}
	if got.HeaderText == "" {
		t.Errorf("header_text was cleared â€” should only update 'subject'")
	}
	if got.CTAText == "" {
		t.Errorf("cta_text was cleared â€” should only update 'subject'")
	}
	if len(got.BodyParagraphs) == 0 {
		t.Errorf("body_paragraphs was cleared â€” should only update 'subject'")
	}
}
