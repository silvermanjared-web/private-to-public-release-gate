package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"private-to-public-release-gate/internal/gate"
)

func main() {
	source := flag.String("source", "", "private canonical source checkout")
	distribution := flag.String("distribution", "", "reviewed public distribution checkout")
	policyPath := flag.String("policy", "publication-policy.json", "publication policy path")
	jsonOutput := flag.Bool("json", false, "emit JSON")
	flag.Parse()
	if *source == "" || *distribution == "" {
		fatal(fmt.Errorf("-source and -distribution are required"), *jsonOutput)
	}
	policy, err := gate.LoadPolicy(*policyPath)
	if err != nil {
		fatal(err, *jsonOutput)
	}
	result, err := gate.Run(*source, *distribution, policy)
	if err != nil {
		fatal(err, *jsonOutput)
	}
	if *jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fatal(err, true)
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("release gate: %s privacy_findings=%d drift_findings=%d\n", result.Status, result.PrivacyFindingCount, result.DriftFindingCount)
		for _, finding := range result.Findings {
			fmt.Printf("finding: %s type=%s detail=%s\n", finding.Path, finding.Type, finding.Detail)
		}
	}
	if result.Status != "pass" {
		os.Exit(1)
	}
}

func fatal(err error, jsonOutput bool) {
	if jsonOutput {
		data, _ := json.MarshalIndent(map[string]string{"status": "error", "error": err.Error()}, "", "  ")
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Fprintln(os.Stderr, "release gate error:", err)
	}
	os.Exit(2)
}
