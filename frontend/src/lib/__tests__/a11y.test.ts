import {
  defaultA11yPrefs,
  loadA11yPrefs,
  saveA11yPrefs,
  applyA11yPrefs,
  FONT_SCALES,
} from '@/lib/a11y';

describe('a11y prefs (WP-3.4 RC-12)', () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.removeAttribute('data-font-scale');
    document.documentElement.removeAttribute('data-contrast');
  });

  it('defaults are honest: normal size, no high contrast, nothing applied', () => {
    const prefs = defaultA11yPrefs();
    expect(prefs.fontScale).toBe('normal');
    expect(prefs.highContrast).toBe(false);
    expect(document.documentElement.hasAttribute('data-font-scale')).toBe(false);
    expect(document.documentElement.hasAttribute('data-contrast')).toBe(false);
  });

  it('round-trips through localStorage without inventing values', () => {
    saveA11yPrefs({ fontScale: 'xlarge', highContrast: true });
    const loaded = loadA11yPrefs();
    expect(loaded).toEqual({ fontScale: 'xlarge', highContrast: true });
  });

  it('rejects unknown stored values and falls back to defaults', () => {
    window.localStorage.setItem('log:a11y', JSON.stringify({ fontScale: 'huge', highContrast: 'yes' }));
    expect(loadA11yPrefs()).toEqual(defaultA11yPrefs());
  });

  it('applies font scale + high contrast to <html> for the whole app', () => {
    applyA11yPrefs({ fontScale: 'large', highContrast: true });
    expect(document.documentElement.getAttribute('data-font-scale')).toBe('large');
    expect(document.documentElement.getAttribute('data-contrast')).toBe('high');
  });

  it('maps every scale to a real multiplier (no fabricated sizes)', () => {
    expect(FONT_SCALES.normal).toBe(1);
    expect(FONT_SCALES.large).toBeGreaterThan(1);
    expect(FONT_SCALES.xlarge).toBeGreaterThan(FONT_SCALES.large);
  });
});