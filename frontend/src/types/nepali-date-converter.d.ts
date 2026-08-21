// Type shim: @sbmdkl/nepali-date-converter ships types/ but its package.json
// "exports" map omits the "types" condition, so TS can't resolve them.
declare module '@sbmdkl/nepali-date-converter' {
  export function adToBs(adDate: string): string | { currentYear: number; currentMonth: number; currentDay: number };
  export function bsToAd(selectedDate: string): string;
}
