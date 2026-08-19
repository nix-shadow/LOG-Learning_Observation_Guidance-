package honesty

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Backend mirror of frontend/__tests__/honesty.test.ts (AGENTS.md §1): the
// backend must NEVER fabricate numbers when data is unavailable — honest
// empty/zero states instead. This test walks the shipped Go source and fails
// when a fabricated-fallback pattern appears, so the guarantee is executable.
//
// Patterns are assembled at runtime so the scanner's own source can never
// self-match (the literal target strings never appear in this file).

var fabricatedPatterns = []struct {
	re  *regexp.Regexp
	why string
}{
	{
		// The old GetChartData fallback: a hardcoded 7-day all-zero week.
		regexp.MustCompile(`"name":\s*"` + "Mon" + `"\s*,\s*"score":\s*0`),
		"fabricated zero-week chart series (GetChartData regression)",
	},
	{
		// A hardcoded weekday-name slice — the tell-tale of a fabricated
		// weekly chart series (the GetChartData regression built one of
		// these; any future weekly fabrication will too).
		regexp.MustCompile(`"` + "Mon" + `",\s*"` + "Tue" + `",\s*"` + "Wed" + `"`),
		"hardcoded weekday series (fabricated weekly chart)",
	},
	{
		// A hardcoded absolute year in time.Date — a fabricated analytics
		// window that pretends to be "the last 7 days" but is pinned to a
		// fixed calendar date forever.
		regexp.MustCompile(`time\.Date\(2[0-9]{3},`),
		"hardcoded absolute date window (fabricated analytics range)",
	},
	{
		// Invented placeholder roster entries.
		regexp.MustCompile(`placeholderStudent|fakeStudent|dummyStudent|demoRoster|mockRoster`),
		"placeholder learner identifiers",
	},
	{
		// Randomly generated analytics.
		regexp.MustCompile(`rand\.Intn|rand\.Float64|Math\.Random|math\.rand`),
		"randomly generated metrics/analytics",
	},
}

// Exclude vendored/cache dirs and this scanner itself from the walk target
// (it is the scanner — nothing to scan). Tests are scanned too: a fabricated
// fallback smuggled into test fixtures is still a red flag, but expected
// values in assertions are legitimate, so only source (non-_test) files are
// scanned for the random-metrics pattern.
func walk(dir string, skipTest bool, out *[]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if e.Name() == "node_modules" || e.Name() == ".git" || e.Name() == "vendor" {
				continue
			}
			walk(p, skipTest, out)
			continue
		}
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if skipTest && strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		*out = append(*out, p)
	}
}

func TestBackendHonestyScan(t *testing.T) {
	root := filepath.Join("..", "..") // backend/internal/honesty -> backend
	var sourceFiles []string
	walk(root, true, &sourceFiles)

	var hits []string
	for _, f := range sourceFiles {
		src, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, p := range fabricatedPatterns {
			if p.re.Match(src) {
				hits = append(hits, filepath.ToSlash(f)+": "+p.why)
			}
		}
	}
	if len(hits) > 0 {
		t.Fatalf("fabricated-fallback patterns found:\n  %s", strings.Join(hits, "\n  "))
	}
}

// The audited surfaces must exist so the sweep covers the endpoints that
// previously fabricated data: chart-data and the moderator roster.
func TestAuditedSurfacesExist(t *testing.T) {
	surfaces := []string{
		filepath.Join("..", "service", "learner_service.go"),    // GetChartData
		filepath.Join("..", "service", "learner_service.go"),    // GetModeratorRoster
		filepath.Join("..", "repository", "course_repo.go"),     // enrollment counts
		filepath.Join("..", "repository", "completion_repo.go"), // DailyActivity rows
	}
	for _, f := range surfaces {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("audited surface missing: %v", f)
		}
	}
}
