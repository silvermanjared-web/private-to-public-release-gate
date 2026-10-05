package gate

import (
	"strings"
	"testing"
)

func TestPrivacyFindingDoesNotEchoForbiddenValue(t *testing.T) {
	source := t.TempDir()
	distribution := t.TempDir()
	privateValue := "private-host.example"
	write(t, source, "notes.txt", "connect to "+privateValue+"\n", 0o644)
	write(t, distribution, "notes.txt", "connect to safe-host.example\n", 0o644)

	policy := Policy{
		Privacy: PrivacyPolicy{
			ForbiddenTerms: []string{privateValue},
		},
	}
	result, err := Run(source, distribution, policy)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fail" || result.PrivacyFindingCount == 0 {
		t.Fatalf("expected privacy failure: %+v", result)
	}
	for _, finding := range result.Findings {
		if strings.Contains(finding.Detail, privateValue) {
			t.Fatal("privacy finding exposed the matched private value")
		}
	}
}
