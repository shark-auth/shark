package storage_test

import (
	"context"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

func TestResolveBranding_GlobalOnly(t *testing.T) {
	s := testutil.NewTestDB(t)
	ctx := context.Background()

	got, err := s.ResolveBranding(ctx, "")
	if err != nil {
		t.Fatalf("ResolveBranding: %v", err)
	}
	if got.PrimaryColor != "#7c3aed" {
		t.Errorf("expected default primary, got %q", got.PrimaryColor)
	}
}

func TestResolveBranding_AppOverride(t *testing.T) {
	s := testutil.NewTestDB(t)
	ctx := context.Background()

	// Seed an application row with a branding_override JSON blob.
	app := newTestApp(t, "Branding Test", false)
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx,
		`UPDATE applications SET branding_override = ? WHERE id = ?`,
		`{"primary_color":"#ff0000"}`, app.ID); err != nil {
		t.Fatalf("seed branding_override: %v", err)
	}

	got, err := s.ResolveBranding(ctx, app.ID)
	if err != nil {
		t.Fatalf("ResolveBranding: %v", err)
	}
	if got.PrimaryColor != "#ff0000" {
		t.Errorf("expected override primary #ff0000, got %q", got.PrimaryColor)
	}
	if got.SecondaryColor != "#1a1a1a" {
		t.Errorf("expected fallback secondary, got %q", got.SecondaryColor)
	}
}
