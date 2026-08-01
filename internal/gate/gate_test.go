package gate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunPassesReviewedExportAndFailsPrivacyBeforeRelease(t *testing.T) {
	source := t.TempDir()
	distribution := t.TempDir()
	write(t, source, "shared.txt", "safe\n", 0o644)
	write(t, source, "private/ignored.txt", "internal\n", 0o644)
	write(t, source, "_public-overlay/README.md", "public\n", 0o644)
	write(t, distribution, "shared.txt", "safe\n", 0o644)
	write(t, distribution, "README.md", "public\n", 0o644)
	policy := Policy{
		OverlayDir:          "_public-overlay",
		ExcludePaths:        []string{".git", "private"},
		AllowedOverlayPaths: []string{"README.md"},
		Privacy: PrivacyPolicy{
			AllowedHomeUsers: []string{"local-operator"},
			AllowedEmails:    []string{"work@example.com"},
			ForbiddenTerms:   []string{"internal-company.example"},
		},
	}
	result, err := Run(source, distribution, policy)
	if err != nil || result.Status != "pass" {
		t.Fatalf("reviewed export rejected: result=%+v err=%v", result, err)
	}
	write(t, source, "person@private-domain.test", "safe\n", 0o644)
	result, err = Run(source, distribution, policy)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || result.PrivacyFindingCount == 0 {
		t.Fatalf("private filename passed: %+v", result)
	}
	for _, finding := range result.Findings {
		if finding.Detail == "person@private-domain.test" {
			t.Fatal("privacy finding echoed matched value")
		}
	}
}

func TestRunRejectsUnreviewedOverlayAndDetectsExecutableDrift(t *testing.T) {
	source := t.TempDir()
	distribution := t.TempDir()
	write(t, source, "script.sh", "#!/bin/sh\n", 0o755)
	write(t, source, "_public-overlay/README.md", "public\n", 0o644)
	write(t, source, "_public-overlay/unreviewed.txt", "unexpected\n", 0o644)
	write(t, distribution, "script.sh", "#!/bin/sh\n", 0o644)
	write(t, distribution, "README.md", "public\n", 0o644)
	policy := Policy{OverlayDir: "_public-overlay", AllowedOverlayPaths: []string{"README.md"}}
	if _, err := Run(source, distribution, policy); err == nil {
		t.Fatal("unreviewed overlay path accepted")
	}
	if err := os.Remove(filepath.Join(source, "_public-overlay", "unreviewed.txt")); err != nil {
		t.Fatal(err)
	}
	result, err := Run(source, distribution, policy)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, finding := range result.Findings {
		if finding.Type == "drift_executable_bit" {
			found = true
		}
	}
	if !found {
		t.Fatalf("executable-bit drift not detected: %+v", result)
	}
}

func TestRunRejectsMissingAllowlistedOverlayPath(t *testing.T) {
	source := t.TempDir()
	distribution := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "_public-overlay"), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := Policy{OverlayDir: "_public-overlay", AllowedOverlayPaths: []string{"README.md"}}
	if _, err := Run(source, distribution, policy); err == nil {
		t.Fatal("missing allowlisted overlay path accepted")
	}
}

func write(t *testing.T, root, rel, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
