package domain

import (
	"math"
	"time"
)

// AttemptStats is the real, client-reported fact set of one completion:
// how long the learner spent and how many knowledge checks they got right.
// The client grades its own quiz (offline-first, no proctoring), so this is
// the only honest signal the platform can derive metrics from.
type AttemptStats struct {
	ElapsedSeconds int `json:"elapsed_seconds"`
	CorrectCount   int `json:"correct_count"`
	TotalCount     int `json:"total_count"`
	// CompletedAtUnixMs is the client's wall-clock completion time (epoch
	// ms). Offline completions land at flush time otherwise, which backdates
	// streaks and daily charts to the wrong day — the learner did the work
	// days ago on a mountain road. Optional; server clamps to a sane window.
	CompletedAtUnixMs int64 `json:"completed_at_unix_ms"`
	// TimezoneIANA is the learner's IANA timezone (e.g. "Asia/Kathmandu").
	// "Today" is computed in THIS zone, not server UTC, so a completion at
	// 11pm Nepali time counts on the right calendar day.
	TimezoneIANA string `json:"timezone_iana"`
}

// HasQuiz reports whether the attempt carried knowledge-check data at all.
func (a AttemptStats) HasQuiz() bool {
	return a.TotalCount > 0
}

// Clamp bounds every field to sane ranges so a hostile or buggy client can
// never poison analytics: elapsed time is capped at 4 hours, quiz counts at
// 100k questions (Accuracy already clamps the fraction).
func (a AttemptStats) Clamp() AttemptStats {
	if a.ElapsedSeconds < 0 {
		a.ElapsedSeconds = 0
	}
	if a.ElapsedSeconds > 4*3600 {
		a.ElapsedSeconds = 4 * 3600
	}
	if a.CorrectCount < 0 {
		a.CorrectCount = 0
	}
	if a.TotalCount < 0 {
		a.TotalCount = 0
	}
	if a.CorrectCount > 100000 {
		a.CorrectCount = 100000
	}
	if a.TotalCount > 100000 {
		a.TotalCount = 100000
	}
	return a
}

// Accuracy returns the fraction of correct answers (0..1), clamped so a
// malformed payload (more correct than total) can never exceed 100%.
func (a AttemptStats) Accuracy() float64 {
	if !a.HasQuiz() {
		return 0
	}
	if a.CorrectCount >= a.TotalCount {
		return 1
	}
	if a.CorrectCount <= 0 {
		return 0
	}
	return float64(a.CorrectCount) / float64(a.TotalCount)
}

// Score maps accuracy to the 0-100 scale used by LearnerActivity.Score,
// rounded to one decimal place.
func (a AttemptStats) Score() float64 {
	return math.Round(a.Accuracy()*1000) / 10
}

// CompletedAt resolves the attempt's completion instant. Research round
// (WP-0.2): without the client timestamp, an offline completion would be
// dated at flush time — a completion made on Monday that syncs on Friday
// would land on Friday's streak and chart row. The client clock is trusted
// only inside a sane window: up to 14 days in the past (a long offline gap)
// and 24 hours in the future (clock skew); anything outside collapses to
// the nearest bound, so a hostile clock can neither backdate forever nor
// fabricate tomorrow.
func (a AttemptStats) CompletedAt(now time.Time) time.Time {
	if a.CompletedAtUnixMs <= 0 {
		return now
	}
	t := time.UnixMilli(a.CompletedAtUnixMs)
	lower := now.Add(-14 * 24 * time.Hour)
	upper := now.Add(24 * time.Hour)
	switch {
	case t.Before(lower):
		return lower
	case t.After(upper):
		return upper
	default:
		return t
	}
}

// Location returns the learner's IANA timezone, or UTC when absent/invalid.
// The calendar date for streak and daily-chart rows is derived in this zone.
func (a AttemptStats) Location() *time.Location {
	if a.TimezoneIANA == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(a.TimezoneIANA)
	if err != nil {
		return time.UTC
	}
	return loc
}

// ObservationText and GuidanceText derive supportive copy from real accuracy
// (AGENTS.md: positive phrasing only — "could use more practice", never
// "failed"). The no-quiz branch preserves the legacy encouragement for
// clients that complete without knowledge checks.
func (a AttemptStats) ObservationText(activityTitle string) string {
	title := activityTitle
	if title == "" {
		title = "Module Completed"
	}
	switch {
	case !a.HasQuiz():
		return "Demonstrated excellent focus and successfully completed " + title + "."
	case a.Accuracy() >= 0.8:
		return "Strong understanding of " + title + " — you answered most knowledge checks correctly on the first try."
	case a.Accuracy() >= 0.5:
		return "Good effort on " + title + " — you are building a solid grasp of the core ideas."
	default:
		return "You showed great persistence working through " + title + " — every attempt strengthens your understanding."
	}
}

func (a AttemptStats) GuidanceText() string {
	switch {
	case !a.HasQuiz():
		return "Great momentum! Continue to the next practice module to reinforce your logic skills."
	case a.Accuracy() >= 0.8:
		return "Keep up the great work! Move on to the next module to build on this strength."
	case a.Accuracy() >= 0.5:
		return "This area could use more practice. Revisit the module, then try the knowledge checks once more."
	default:
		return "Let's strengthen the foundations together: review the module content step by step, then try the knowledge checks again. You've got this."
	}
}
