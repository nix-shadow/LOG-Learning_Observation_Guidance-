// WP-1.4: SM-2-style spaced repetition scheduler (local-first).
// Pure functions — no I/O — so every rule is unit-testable.
// Quality q is mapped from real quiz accuracy (0..1) to the SM-2 0..5 scale,
// so scheduling is always derived from genuine attempt data, never invented.

export interface ReviewState {
  activityId: string;
  ease: number;
  intervalDays: number;
  repetition: number;
  lastReviewedAt: string;
  dueDate: string;
}

export const DEFAULT_EASE = 2.5;
export const MIN_EASE = 1.3;

export function qualityFromAccuracy(accuracy: number): number {
  const clamped = Math.max(0, Math.min(1, accuracy));
  return Math.max(0, Math.min(5, Math.round(clamped * 5)));
}

function addDays(date: Date, days: number): string {
  const d = new Date(date);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

export function scheduleReview(
  prev: ReviewState | null,
  accuracy: number,
  completedAt: string
): ReviewState {
  const activityId = prev?.activityId ?? '';
  const q = qualityFromAccuracy(accuracy);
  let ease = prev?.ease ?? DEFAULT_EASE;
  let repetition = prev?.repetition ?? 0;
  let intervalDays = prev?.intervalDays ?? 1;

  if (q < 3) {
    // Failed review: reset the repetition count, fall back to a 1-day interval.
    // The ease factor survives — SM-2 treats lapses as scheduling noise, not
    // proof the item is "hard forever". No negative framing, ever.
    repetition = 0;
    intervalDays = 1;
  } else {
    repetition += 1;
    if (repetition === 1) {
      intervalDays = 1;
    } else if (repetition === 2) {
      intervalDays = 6;
    } else {
      intervalDays = Math.max(1, Math.round(intervalDays * ease));
    }
    ease = ease + (0.1 - (5 - q) * (0.08 + (5 - q) * 0.02));
    ease = Math.max(MIN_EASE, ease);
  }

  return {
    activityId,
    ease,
    intervalDays,
    repetition,
    lastReviewedAt: completedAt,
    dueDate: addDays(new Date(completedAt), intervalDays),
  };
}

export function isDue(state: ReviewState, now: Date): boolean {
  return state.dueDate <= now.toISOString().slice(0, 10);
}