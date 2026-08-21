"use client";

// WP-3.4 (RC-12): applies the user's accessibility packs (font scale +
// high contrast) to <html> the moment the app mounts, and keeps them in
// sync with localStorage. Settings live in the Settings page; this provider
// is the single apply-point so the whole app honors the choice offline.
import { useEffect, useState, createContext, useContext } from "react";
import {
  A11yPrefs,
  applyA11yPrefs,
  defaultA11yPrefs,
  loadA11yPrefs,
  saveA11yPrefs,
} from "@/lib/a11y";

const A11yContext = createContext<{
  prefs: A11yPrefs;
  setPrefs: (p: A11yPrefs) => void;
}>({ prefs: defaultA11yPrefs(), setPrefs: () => {} });

export function useA11y() {
  return useContext(A11yContext);
}

export default function A11yProvider({ children }: { children: React.ReactNode }) {
  const [prefs, setPrefsState] = useState<A11yPrefs>(defaultA11yPrefs());

  useEffect(() => {
    const loaded = loadA11yPrefs();
    setPrefsState(loaded);
    applyA11yPrefs(loaded);
  }, []);

  const setPrefs = (p: A11yPrefs) => {
    setPrefsState(p);
    applyA11yPrefs(p);
    saveA11yPrefs(p);
  };

  return <A11yContext.Provider value={{ prefs, setPrefs }}>{children}</A11yContext.Provider>;
}