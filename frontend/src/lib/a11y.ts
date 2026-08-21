// WP-3.4 (RC-12): accessibility packs — user-controlled font scale and
// high-contrast theme, persisted locally and applied to <html> via data
// attributes. Honest by design: no server round trip, works fully offline,
// and nothing is applied until the user chooses it.

export type FontScale = "normal" | "large" | "xlarge";
export type A11yPrefs = {
  fontScale: FontScale;
  highContrast: boolean;
};

const STORAGE_KEY = "log:a11y";

export const FONT_SCALES: Record<FontScale, number> = {
  normal: 1,
  large: 1.18,
  xlarge: 1.35,
};

export const FONT_SCALE_LABELS: FontScale[] = ["normal", "large", "xlarge"];

export function defaultA11yPrefs(): A11yPrefs {
  return { fontScale: "normal", highContrast: false };
}

export function loadA11yPrefs(): A11yPrefs {
  if (typeof window === "undefined") return defaultA11yPrefs();
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultA11yPrefs();
    const parsed = JSON.parse(raw) as Partial<A11yPrefs>;
    return {
      fontScale:
        parsed.fontScale && parsed.fontScale in FONT_SCALES
          ? (parsed.fontScale as FontScale)
          : "normal",
      highContrast: parsed.highContrast === true,
    };
  } catch {
    return defaultA11yPrefs();
  }
}

export function saveA11yPrefs(prefs: A11yPrefs): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // Storage unavailable (private mode) — the session still applies the
    // prefs; they just won't survive a reload. No fabricated state.
  }
}

export function applyA11yPrefs(prefs: A11yPrefs): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  if (prefs.fontScale === "normal") {
    root.removeAttribute("data-font-scale");
  } else {
    root.setAttribute("data-font-scale", prefs.fontScale);
  }
  if (prefs.highContrast) {
    root.setAttribute("data-contrast", "high");
  } else {
    root.removeAttribute("data-contrast");
  }
}

// Apply persisted prefs before first paint so there is no flash of the
// unstyled (wrong-size) interface. Called from the provider on mount.
export function initA11yPrefs(): void {
  applyA11yPrefs(loadA11yPrefs());
}