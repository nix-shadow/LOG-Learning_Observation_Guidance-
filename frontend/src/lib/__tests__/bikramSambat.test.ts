import { adToBsDate, bsToAdDate, formatBs, toDevanagariDigits } from '../bikramSambat';

// WP-1.3: Bikram Sambat conversion unit tests. Anchors are round-trip
// verified against the converter dataset:
//   2026-08-20 → 2083-05-04 → 2026-08-20 (Shrawan 4, 2083).

describe('bikramSambat', () => {
  it('converts a known AD date to BS (2026-08-20 → 2083-05-04)', () => {
    expect(adToBsDate('2026-08-20')).toEqual({ year: 2083, month: 5, day: 4 });
  });

  it('converts a Date object the same as its ISO string', () => {
    const d = new Date(2026, 7, 20); // Aug 20 local
    const iso = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    expect(adToBsDate(d)).toEqual(adToBsDate(iso));
  });

  it('round-trips BS → AD → BS', () => {
    expect(bsToAdDate({ year: 2083, month: 5, day: 4 })).toBe('2026-08-20');
    expect(adToBsDate('2026-08-20')).toEqual({ year: 2083, month: 5, day: 4 });
  });

  it('returns null for out-of-range dates instead of inventing values', () => {
    expect(adToBsDate('1900-01-01')).toBeNull();
    expect(adToBsDate('not-a-date')).toBeNull();
    expect(bsToAdDate({ year: 2099, month: 12, day: 40 })).toBeNull();
  });

  it('formats BS dates in English and Nepali', () => {
    expect(formatBs('2026-08-20', 'en')).toBe('Bhadra 4, 2083');
    expect(formatBs('2026-08-20', 'np')).toBe('भदौ ४, २०८३');
  });

  it('converts Western digits to Devanagari', () => {
    expect(toDevanagariDigits(2083)).toBe('२०८३');
    expect(toDevanagariDigits('4')).toBe('४');
  });

  it('returns null formatting garbage input', () => {
    expect(formatBs('junk')).toBeNull();
  });
});