package generatedapp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	frontendInvokeActionPattern  = regexp.MustCompile(`action:\s*['"]([^'"]+)['"]`)
	backendDispatchActionPattern = regexp.MustCompile(`case\s+"([^"]+)"`)
)

func normalizeManifestActions(appDir string, item manifest) (manifest, bool, error) {
	normalized := normalizeSortedUniqueStrings(item.Actions)
	if len(normalized) > 0 {
		if !equalStrings(normalized, item.Actions) {
			item.Actions = normalized
			return item, true, nil
		}
		return item, false, nil
	}
	inferred, err := inferManifestActions(appDir)
	if err != nil {
		return item, false, err
	}
	if len(inferred) == 0 {
		return item, false, nil
	}
	item.Actions = inferred
	return item, true, nil
}

func inferManifestActions(appDir string) ([]string, error) {
	frontendActions, err := extractActionStrings(filepath.Join(appDir, "frontend", "api.js"), frontendInvokeActionPattern)
	if err != nil {
		return nil, err
	}
	backendActions, err := extractActionStrings(filepath.Join(appDir, "backend", "main.go"), backendDispatchActionPattern)
	if err != nil {
		return nil, err
	}

	candidates := intersectStrings(frontendActions, backendActions)
	if len(candidates) == 0 {
		candidates = unionStrings(frontendActions, backendActions)
	}

	actions := make([]string, 0, len(candidates))
	for _, action := range candidates {
		if ActionRequiresAuthorization(action) {
			actions = append(actions, action)
		}
	}
	return normalizeSortedUniqueStrings(actions), nil
}

func extractActionStrings(path string, pattern *regexp.Regexp) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read action source %q: %w", path, err)
	}
	matches := pattern.FindAllStringSubmatch(string(raw), -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		values = append(values, match[1])
	}
	return normalizeSortedUniqueStrings(values), nil
}

// ActionRequiresAuthorization identifies actions that must be declared in the
// manifest and granted by the generated-function menu before they can receive
// write or outbound platform capabilities.
func ActionRequiresAuthorization(action string) bool {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		return false
	}
	readonlySuffixes := []string{
		"_list", "_detail", "_get", "_query", "_search", "_page", "_options",
		"_stats", "_count", "_summary", "_info", "_tree", "_preview",
	}
	for _, suffix := range readonlySuffixes {
		if strings.HasSuffix(action, suffix) {
			return false
		}
	}
	readonlyKeywords := []string{
		"list", "detail", "get", "query", "search", "page", "options",
		"stats", "count", "summary", "info", "tree", "preview",
	}
	for _, keyword := range readonlyKeywords {
		if action == keyword {
			return false
		}
	}
	return true
}

func normalizeSortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func intersectStrings(left []string, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		rightSet[item] = struct{}{}
	}
	out := make([]string, 0, len(left))
	for _, item := range left {
		if _, ok := rightSet[item]; ok {
			out = append(out, item)
		}
	}
	return normalizeSortedUniqueStrings(out)
}

func unionStrings(groups ...[]string) []string {
	var merged []string
	for _, group := range groups {
		merged = append(merged, group...)
	}
	return normalizeSortedUniqueStrings(merged)
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
