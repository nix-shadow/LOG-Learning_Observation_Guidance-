package domain

import "math"

// AttemptStats is the real, client-reported fact set of one completion:
// how long the learner spent and how many knowledge checks they got right.
// The client grades its own quiz (offline-first, no proctoring), so this is
// the only honest signal the platform can derive metrics from.
type AttemptStats struct {
	ElapsedSeconds int `json:"elapsed_seconds"`
	CorrectCount   int `json:"correct_count"`
	TotalCount     int `json:"total_count"`
}

// HasQuiz reports whether the attempt carried knowledge-check data at all.
func (a AttemptStats) HasQuiz() bool {
	return a.TotalCount > 0
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
