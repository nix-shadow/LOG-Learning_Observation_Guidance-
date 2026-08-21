import { adToBs, bsToAd } from '@sbmdkl/nepali-date-converter';

// WP-1.3: Bikram Sambat calendar support for Nepali-first UI. The conversion
// dataset lives in @sbmdkl/nepali-date-converter (the standard Nepali
// calendar table, BS 1978-2099 / AD 1921-2040). Round-trip verified:
//   2026-08-20 → 2083-05-04 → 2026-08-20
// Every conversion here is derived from that dataset — no invented dates.

export interface BsDate {
  year: number;
  month: number; // 1-12 (Baisakh=1 ... Chaitra=12)
  day: number;
}

export const BS_MONTHS = [
  'Baisakh',
  'Jestha',
  'Ashadh',
  'Shrawan',
  'Bhadra',
  'Ashwin',
  'Kartik',
  'Mangsir',
  'Poush',
  'Magh',
  'Falgun',
  'Chaitra',
];

const DEVANAGARI_DIGITS = ['०', '१', '२', '३', '४', '५', '६', '७', '८', '९'];

function toYmd(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

/** AD date → BS date. Returns null when out of the converter's range. */
export function adToBsDate(d: Date | string): BsDate | null {
  try {
    const out = adToBs(d instanceof Date ? toYmd(d) : d);
    if (typeof out !== 'string') return null;
    const [year, month, day] = out.split('-').map(Number);
    if (!year || !month || !day) return null;
    return { year, month, day };
  } catch {
    return null;
  }
}

/** BS date → AD ISO string. Returns null when out of range. */
export function bsToAdDate(bs: BsDate): string | null {
  try {
    const y = String(bs.year);
    const m = String(bs.month).padStart(2, '0');
    const d = String(bs.day).padStart(2, '0');
    return bsToAd(`${y}-${m}-${d}`);
  } catch {
    return null;
  }
}

/** Convert Western digits to Devanagari (e.g. 2083 → २०८३). */
export function toDevanagariDigits(n: number | string): string {
  return String(n)
    .split('')
    .map((c) => (c >= '0' && c <= '9' ? DEVANAGARI_DIGITS[Number(c)] : c))
    .join('');
}

/** "Baisakh 4, 2083" (English) or "बैशाख ४, २०८३" (Nepali). */
export function formatBs(d: Date | string, lang: 'en' | 'np' = 'en'): string | null {
  const bs = adToBsDate(d);
  if (!bs) return null;
  if (lang === 'np') {
    const month = BS_MONTHS_NP[bs.month - 1];
    return `${month} ${toDevanagariDigits(bs.day)}, ${toDevanagariDigits(bs.year)}`;
  }
  return `${BS_MONTHS[bs.month - 1]} ${bs.day}, ${bs.year}`;
}

export const BS_MONTHS_NP = [
  'बैशाख',
  'जेठ',
  'असार',
  'साउन',
  'भदौ',
  'असोज',
  'कात्तिक',
  'मंसिर',
  'पुष',
  'माघ',
  'फागुन',
  'चैत',
];