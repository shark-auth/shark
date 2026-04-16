package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestAskQuestionsDefault(t *testing.T) {
	stdin := strings.NewReader("\n") // accept default
	var out bytes.Buffer
	a, err := askQuestions(stdin, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL default: got %q", a.BaseURL)
	}
}

func TestAskQuestionsCustom(t *testing.T) {
	stdin := strings.NewReader("https://auth.example.com\n")
	var out bytes.Buffer
	a, err := askQuestions(stdin, &out)
	if err != nil {
		t.Fatal(err)
	}
	if a.BaseURL != "https://auth.example.com" {
		t.Errorf("BaseURL: got %q", a.BaseURL)
	}
}

func TestRenderYAMLContainsRequiredVars(t *testing.T) {
	a := initAnswers{BaseURL: "https://auth.example.com"}
	out := renderYAML(a, "a"+strings.Repeat("b", 63))
	for _, want := range []string{
		"https://auth.example.com",
		`provider: "shark"`,
		`secret: "a` + strings.Repeat("b", 63) + `"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered YAML missing %q", want)
		}
	}
}

func TestRenderYAMLHasTestingNotice(t *testing.T) {
	// The generated file must point operators at how to switch from the
	// shark.email testing tier to a production provider.
	a := initAnswers{BaseURL: "http://localhost:8080"}
	out := renderYAML(a, strings.Repeat("x", 64))
	for _, want := range []string{
		"shark email setup",
		"Settings → Email",
		"sharkauth.com/docs",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered YAML missing switch-provider hint %q", want)
		}
	}
}

func TestRenderYAMLIsShort(t *testing.T) {
	// Phase 2 constraint: minimal config. Generated file stays under 20 lines.
	a := initAnswers{BaseURL: "http://localhost:8080"}
	out := renderYAML(a, strings.Repeat("x", 64))
	lines := strings.Count(out, "\n")
	if lines > 20 {
		t.Errorf("generated YAML is %d lines; phase 2 requires minimal (≤20)", lines)
	}
}

func TestPostInitNoticeMentionsAllThreeSwitchPaths(t *testing.T) {
	var out bytes.Buffer
	printPostInitNotice(&out)
	body := out.String()
	for _, want := range []string{
		"shark.email",
		"testing",
		"sharkauth.yaml",
		"shark email setup",
		"admin dashboard",
		"sharkauth.com/docs",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("post-init notice missing %q", want)
		}
	}
}
