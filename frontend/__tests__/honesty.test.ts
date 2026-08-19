import fs from 'fs';
import path from 'path';

/**
 * WP-0.2 C5/C1 (AGENTS.md §1): the frontend must NEVER render invented
 * numbers or placeholder learners when data is unavailable — honest 0/empty
 * states instead. This test scans the shipped source (pages + components,
 * tests excluded) for fabricated-fallback patterns and fails when one is
 * found, so the "no placeholder stubs" guarantee is executable, not a rule
 * on paper.
 */
const ROOT = path.join(__dirname, '..', 'src');

const BANNED_IDENTIFIERS = [
  'placeholderStudent',
  'fakeStudent',
  'dummyStudent',
  'sampleRoster',
  'mockRoster',
  'fakeData',
  'dummyData',
  'sampleData',
  'placeholderData',
  'hardcodedStudents',
  'demoRoster',
];

// Patterns that fabricate data at render time instead of showing 0/empty.
const BANNED_PATTERNS: Array<[RegExp, string]> = [
  [/const\s+\w+\s*=\s*\[\s*\{\s*name:\s*['"][A-Z][a-z]+ ['"][A-Z]/, 'hardcoded named object (likely an invented learner)'],
  [/Math\.random\(\).*score|score.*Math\.random\(\)/, 'randomly generated scores'],
  [/\[\s*\{\s*(name|label|day):\s*['"][^'"]+['"],\s*(value|score|duration|count):\s*\d+\s*\}/, 'hardcoded data point array'],
];

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === '__tests__' || entry.name === 'node_modules') continue;
      walk(p, out);
    } else if (entry.name.endsWith('.tsx') || entry.name.endsWith('.ts')) {
      out.push(p);
    }
  }
  return out;
}

describe('honest-zero sweep (WP-0.2)', () => {
  const files = walk(ROOT);

  it('scans the shipped source for fabricated-fallback identifiers', () => {
    const hits: string[] = [];
    for (const file of files) {
      const src = fs.readFileSync(file, 'utf8');
      for (const id of BANNED_IDENTIFIERS) {
        if (src.includes(id)) hits.push(`${path.relative(ROOT, file)}: "${id}"`);
      }
    }
    expect(hits).toEqual([]);
  });

  it('scans for hardcoded invented data (named objects, random scores, data arrays)', () => {
    const hits: string[] = [];
    for (const file of files) {
      const src = fs.readFileSync(file, 'utf8');
      for (const [re, why] of BANNED_PATTERNS) {
        if (re.test(src)) hits.push(`${path.relative(ROOT, file)}: ${why}`);
      }
    }
    expect(hits).toEqual([]);
  });

  it('covers the audited surfaces: goal ring, moderator roster, chart-data consumers', () => {
    const surfaces = [
      'src/app/dashboard/page.tsx',   // goal ring
      'src/components/moderator/RosterOverview.tsx', // roster
      'src/app/observation/page.tsx', // chart-data charts
    ].map((p) => path.join(__dirname, '..', p));
    for (const f of surfaces) {
      expect(fs.existsSync(f)).toBe(true);
    }
  });
});