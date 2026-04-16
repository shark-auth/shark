package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestAskQuestionsDefaults(t *testing.T) {
	stdin := strings.NewReader("\n\n\n") // accept all defaults
	var out bytes.Buffer
	a, err := askQuestions(stdin, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL default: got %q", a.BaseURL)
	}
	if a.AdminEmail != "" {
		t.Errorf("AdminEmail default: got %q", a.AdminEmail)
	}
	if a.SMTPProvider != "resend" {
		t.Errorf("SMTPProvider default: got %q", a.SMTPProvider)
	}
}

func TestAskQuestionsCustom(t *testing.T) {
	stdin := strings.NewReader("https://auth.example.com\nadmin@example.com\nnone\n")
	var out bytes.Buffer
	a, err := askQuestions(stdin, &out)
	if err != nil {
		t.Fatal(err)
	}
	if a.BaseURL != "https://auth.example.com" {
		t.Errorf("BaseURL: got %q", a.BaseURL)
	}
	if a.AdminEmail != "admin@example.com" {
		t.Errorf("AdminEmail: got %q", a.AdminEmail)
	}
	if a.SMTPProvider != "none" {
		t.Errorf("SMTPProvider: got %q", a.SMTPProvider)
	}
}

func TestAskQuestionsInvalidProvider(t *testing.T) {
	stdin := strings.NewReader("\n\nmailgun\n")
	var out bytes.Buffer
	_, err := askQuestions(stdin, &out)
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestRenderYAMLContainsAnswers(t *testing.T) {
	a := initAnswers{
		BaseURL:      "https://auth.example.com",
		AdminEmail:   "admin@example.com",
		SMTPProvider: "resend",
	}
	out := renderYAML(a, "a"+strings.Repeat("b", 63))
	for _, want := range []string{
		"https://auth.example.com",
		"admin@example.com",
		`provider: "resend"`,
		`secret: "a` + strings.Repeat("b", 63) + `"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered YAML missing %q", want)
		}
	}
}

func TestRenderYAMLNoSMTP(t *testing.T) {
	a := initAnswers{BaseURL: "http://localhost:8080", AdminEmail: "x@y.z", SMTPProvider: "none"}
	out := renderYAML(a, strings.Repeat("x", 64))
	if strings.Contains(out, `provider: "resend"`) {
		t.Error("none provider should not emit resend email block")
	}
	if !strings.Contains(out, "shark serve --dev") {
		t.Error("none provider should hint at `shark serve --dev`")
	}
}

func TestRenderYAMLIsShort(t *testing.T) {
	// Phase 2 constraint (ATTACK.md): config-yaml settable with only 3 minimum
	// vars. Generated file stays under 20 lines so new users aren't intimidated.
	a := initAnswers{BaseURL: "http://localhost:8080", AdminEmail: "x@y.z", SMTPProvider: "resend"}
	out := renderYAML(a, strings.Repeat("x", 64))
	lines := strings.Count(out, "\n")
	if lines > 20 {
		t.Errorf("generated YAML is %d lines; phase 2 requires minimal (≤20) config", lines)
	}
}
