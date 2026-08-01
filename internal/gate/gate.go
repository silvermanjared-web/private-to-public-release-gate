package gate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Policy struct {
	OverlayDir          string        `json:"overlay_dir"`
	ExcludePaths        []string      `json:"exclude_paths"`
	AllowedOverlayPaths []string      `json:"allowed_overlay_paths"`
	Privacy             PrivacyPolicy `json:"privacy"`
}

type PrivacyPolicy struct {
	AllowedHomeUsers []string `json:"allowed_home_users"`
	AllowedEmails    []string `json:"allowed_emails"`
	ForbiddenTerms   []string `json:"forbidden_terms"`
}

type Finding struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

type Result struct {
	Status              string    `json:"status"`
	PrivacyFindingCount int       `json:"privacy_finding_count"`
	DriftFindingCount   int       `json:"drift_finding_count"`
	Findings            []Finding `json:"findings"`
}

type entry struct {
	kind   string
	mode   fs.FileMode
	digest [sha256.Size]byte
	target string
}

var emailPattern = regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)
var homePattern = regexp.MustCompile(`/Users/([A-Za-z0-9._-]+)`)
var hostnamePattern = regexp.MustCompile(`(?i)\b[A-Za-z0-9._-]*(?:macbook|imac|mac-mini)[A-Za-z0-9._-]*\.local\b`)
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`gh[opsu]_[A-Za-z0-9]{20,}`),
}

func LoadPolicy(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return Policy{}, err
	}
	if policy.OverlayDir == "" || filepath.IsAbs(policy.OverlayDir) || strings.Contains(filepath.ToSlash(policy.OverlayDir), "../") {
		return Policy{}, fmt.Errorf("overlay_dir must be a safe relative path")
	}
	for _, path := range append(append([]string{}, policy.ExcludePaths...), policy.AllowedOverlayPaths...) {
		if path == "" || filepath.IsAbs(path) || strings.HasPrefix(filepath.ToSlash(filepath.Clean(path)), "../") {
			return Policy{}, fmt.Errorf("policy path must stay inside the repository: %q", path)
		}
	}
	return policy, nil
}

func Run(source, distribution string, policy Policy) (Result, error) {
	for label, root := range map[string]string{"source": source, "distribution": distribution} {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			return Result{}, fmt.Errorf("%s is not a directory: %s", label, root)
		}
	}
	temp, err := os.MkdirTemp("", "private-public-release-gate-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(temp)
	generated := filepath.Join(temp, "generated")
	if err := os.MkdirAll(generated, 0o755); err != nil {
		return Result{}, err
	}
	if err := copySource(source, generated, policy); err != nil {
		return Result{}, err
	}
	if err := applyOverlay(source, generated, policy); err != nil {
		return Result{}, err
	}
	privacy, err := scanPrivacy(generated, policy.Privacy)
	if err != nil {
		return Result{}, err
	}
	drift, err := compare(generated, distribution)
	if err != nil {
		return Result{}, err
	}
	findings := append(privacy, drift...)
	sortFindings(findings)
	status := "pass"
	if len(findings) != 0 {
		status = "fail"
	}
	return Result{Status: status, PrivacyFindingCount: len(privacy), DriftFindingCount: len(drift), Findings: findings}, nil
}

func copySource(source, generated string, policy Policy) error {
	return filepath.WalkDir(source, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") || rel == policy.OverlayDir || strings.HasPrefix(rel, policy.OverlayDir+"/") || excluded(rel, policy.ExcludePaths) {
			if item.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return copyEntry(path, filepath.Join(generated, filepath.FromSlash(rel)), item)
	})
}

func excluded(rel string, paths []string) bool {
	for _, path := range paths {
		path = filepath.ToSlash(filepath.Clean(path))
		if rel == path || strings.HasPrefix(rel, path+"/") {
			return true
		}
	}
	return false
}

func applyOverlay(source, generated string, policy Policy) error {
	overlay := filepath.Join(source, filepath.FromSlash(policy.OverlayDir))
	expected := make(map[string]bool, len(policy.AllowedOverlayPaths))
	for _, path := range policy.AllowedOverlayPaths {
		expected[filepath.ToSlash(filepath.Clean(path))] = true
	}
	if err := filepath.WalkDir(overlay, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == overlay || item.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(overlay, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !expected[rel] {
			return fmt.Errorf("unreviewed overlay path: %s", rel)
		}
		delete(expected, rel)
		return copyEntry(path, filepath.Join(generated, filepath.FromSlash(rel)), item)
	}); err != nil {
		return err
	}
	if len(expected) != 0 {
		missing := make([]string, 0, len(expected))
		for path := range expected {
			missing = append(missing, path)
		}
		sort.Strings(missing)
		return fmt.Errorf("allowlisted overlay paths missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

func copyEntry(source, destination string, item fs.DirEntry) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if item.IsDir() {
		return os.MkdirAll(destination, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return err
		}
		_ = os.Remove(destination)
		return os.Symlink(target, destination)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, info.Mode().Perm())
}

func scanPrivacy(root string, policy PrivacyPolicy) ([]Finding, error) {
	allowedUsers := stringSet(policy.AllowedHomeUsers)
	allowedEmails := stringSet(policy.AllowedEmails)
	var findings []Finding
	err := filepath.WalkDir(root, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || item.IsDir() || item.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		combined := rel + "\n" + string(data)
		rules := map[string]bool{}
		for _, match := range emailPattern.FindAllString(combined, -1) {
			if !allowedEmails[strings.ToLower(match)] {
				rules["non-allowlisted-email"] = true
			}
		}
		for _, match := range homePattern.FindAllStringSubmatch(combined, -1) {
			if len(match) == 2 && !allowedUsers[strings.ToLower(match[1])] {
				rules["non-allowlisted-home-path"] = true
			}
		}
		if hostnamePattern.FindStringIndex(combined) != nil {
			rules["local-hostname"] = true
		}
		for _, term := range policy.ForbiddenTerms {
			if term != "" && strings.Contains(strings.ToLower(combined), strings.ToLower(term)) {
				rules["forbidden-private-term"] = true
			}
		}
		for _, pattern := range secretPatterns {
			if pattern.Find(data) != nil {
				rules["secret-like-token"] = true
			}
		}
		for rule := range rules {
			findings = append(findings, Finding{Path: rel, Type: "privacy_" + rule, Detail: "path and rule only; matched value suppressed"})
		}
		return nil
	})
	sortFindings(findings)
	return findings, err
}

func compare(generated, distribution string) ([]Finding, error) {
	left, err := inventory(generated)
	if err != nil {
		return nil, err
	}
	right, err := inventory(distribution)
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for path := range left {
		paths[path] = true
	}
	for path := range right {
		paths[path] = true
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	var findings []Finding
	for _, path := range ordered {
		a, aOK := left[path]
		b, bOK := right[path]
		switch {
		case !aOK:
			findings = append(findings, Finding{Path: path, Type: "drift_distribution_only", Detail: "path exists only in distribution"})
		case !bOK:
			findings = append(findings, Finding{Path: path, Type: "drift_generated_only", Detail: "path exists only in generated export"})
		case a.kind != b.kind:
			findings = append(findings, Finding{Path: path, Type: "drift_type", Detail: "entry type differs"})
		case a.kind == "file" && (a.mode.Perm()&0o111 != 0) != (b.mode.Perm()&0o111 != 0):
			findings = append(findings, Finding{Path: path, Type: "drift_executable_bit", Detail: "Git executable bit differs"})
		case a.kind == "file" && a.digest != b.digest:
			findings = append(findings, Finding{Path: path, Type: "drift_content", Detail: "content differs"})
		case a.kind == "symlink" && a.target != b.target:
			findings = append(findings, Finding{Path: path, Type: "drift_symlink", Detail: "symlink target differs"})
		}
	}
	return findings, nil
}

func inventory(root string) (map[string]entry, error) {
	entries := map[string]entry{}
	err := filepath.WalkDir(root, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			if item.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		current := entry{mode: info.Mode()}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			current.kind = "symlink"
			current.target, err = os.Readlink(path)
		case info.IsDir():
			current.kind = "directory"
		case info.Mode().IsRegular():
			current.kind = "file"
			var data []byte
			data, err = os.ReadFile(path)
			current.digest = sha256.Sum256(data)
		default:
			return fmt.Errorf("unsupported file type: %s", rel)
		}
		if err != nil {
			return err
		}
		entries[rel] = current
		return nil
	})
	return entries, err
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[strings.ToLower(value)] = true
	}
	return set
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path == findings[j].Path {
			return findings[i].Type < findings[j].Type
		}
		return findings[i].Path < findings[j].Path
	})
}
