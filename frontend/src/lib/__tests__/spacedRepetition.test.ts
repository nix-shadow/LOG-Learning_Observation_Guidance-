import { scheduleReview, qualityFromAccuracy, isDue, DEFAULT_EASE, MIN_EASE, ReviewState } from '../spacedRepetition';

const base: ReviewState = {
  activityId: 'act-1',
  ease: DEFAULT_EASE,
  intervalDays: 1,
  repetition: 0,
  lastReviewedAt: '2026-08-01T00:00:00.000Z',
  dueDate: '2026-08-02',
};

describe('qualityFromAccuracy', () => {
  it('maps honest accuracy to the SM-2 0..5 scale', () => {
    expect(qualityFromAccuracy(1)).toBe(5);
    expect(qualityFromAccuracy(0.8)).toBe(4);
    expect(qualityFromAccuracy(0.5)).toBe(3);
    expect(qualityFromAccuracy(0.4)).toBe(2);
    expect(qualityFromAccuracy(0)).toBe(0);
  });

  it('clamps out-of-range input instead of inventing values', () => {
    expect(qualityFromAccuracy(-1)).toBe(0);
    expect(qualityFromAccuracy(2)).toBe(5);
  });
});

describe('scheduleReview (SM-2 rules)', () => {
  it('first review schedules a 1-day interval at default ease', () => {
    const r = scheduleReview(null, 1, '2026-08-01T00:00:00.000Z');
    expect(r.intervalDays).toBe(1);
    expect(r.repetition).toBe(1);
    expect(r.ease).toBe(DEFAULT_EASE + 0.1); // perfect quality nudges ease up
    expect(r.dueDate).toBe('2026-08-02');
    expect(r.activityId).toBe('');
  });

  it('second consecutive success moves to a 6-day interval', () => {
    const afterFirst: ReviewState = { ...base, repetition: 1, intervalDays: 1, ease: 2.5, dueDate: '2026-08-02' };
    const r = scheduleReview(afterFirst, 0.8, '2026-08-03T00:00:00.000Z');
    expect(r.repetition).toBe(2);
    expect(r.intervalDays).toBe(6);
    expect(r.dueDate).toBe('2026-08-09');
  });

  it('third success grows the interval by the ease factor', () => {
    const afterTwo: ReviewState = { ...base, repetition: 2, intervalDays: 6, dueDate: '2026-08-09' };
    const r = scheduleReview(afterTwo, 1, '2026-08-09T00:00:00.000Z');
    expect(r.repetition).toBe(3);
    expect(r.intervalDays).toBe(Math.round(6 * DEFAULT_EASE)); // 15
    expect(r.dueDate).toBe('2026-08-24');
  });

  it('a weak review (q<3) resets repetition to a 1-day interval but keeps ease', () => {
    const strong: ReviewState = { ...base, ease: 3.0, repetition: 4, intervalDays: 30 };
    const r = scheduleReview(strong, 0.3, '2026-08-10T00:00:00.000Z');
    expect(r.repetition).toBe(0);
    expect(r.intervalDays).toBe(1);
    expect(r.ease).toBe(3.0); // lapse is scheduling noise, not a "hard item" verdict
    expect(r.dueDate).toBe('2026-08-11');
  });

  it('ease factor is clamped to a supportive minimum, never below 1.3', () => {
    const r = scheduleReview({ ...base, ease: 1.0, repetition: 3, intervalDays: 10 }, 0.6, '2026-08-11T00:00:00.000Z');
    expect(r.ease).toBe(MIN_EASE);
  });

  it('no quiz (total_count 0) counts as a perfect review — it was completed', () => {
    // Concept-only lessons have no quiz; the completion is real and positive.
    const afterFirst: ReviewState = { ...base, repetition: 1, intervalDays: 1, ease: 2.6, dueDate: '2026-08-02' };
    const r = scheduleReview(afterFirst, 1, '2026-08-05T00:00:00.000Z');
    expect(r.repetition).toBe(2);
    expect(r.intervalDays).toBe(6);
  });
});

describe('isDue', () => {
  it('due on the due date, not before', () => {
    const s: ReviewState = { ...base, dueDate: '2026-08-02' };
    expect(isDue(s, new Date('2026-08-01T23:59:00.000Z'))).toBe(false);
    expect(isDue(s, new Date('2026-08-02T00:00:00.000Z'))).toBe(true);
    expect(isDue(s, new Date('2026-08-05T00:00:00.000Z'))).toBe(true);
  });
});