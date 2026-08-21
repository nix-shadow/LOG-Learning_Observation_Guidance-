"use client";
import { createContext, useContext, useEffect, useState } from "react";
import { NextIntlClientProvider } from "next-intl";
import en from "@/messages/en.json";
import np from "@/messages/np.json";

// WP-1.3 (bilingual shell): lightweight client-side locale switching with
// next-intl. No URL-prefix routing — the app is a client-heavy SPA shell for
// low-connectivity school LANs, so the locale lives in the user's browser
// (stored preference, falling back to the browser language).

export type Locale = "en" | "np";

const messages: Record<Locale, Record<string, unknown>> = { en, np };
const STORAGE_KEY = "log-locale";

function storedLocale(): Locale {
  if (typeof window === "undefined") return "en";
  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored === "en" || stored === "np") return stored;
  return navigator.language?.toLowerCase().startsWith("ne") ? "np" : "en";
}

interface LocaleCtxValue {
  locale: Locale;
  setLocale: (l: Locale) => void;
}

const LocaleCtx = createContext<LocaleCtxValue>({ locale: "en", setLocale: () => {} });

export function useLocaleCtx() {
  return useContext(LocaleCtx);
}

export default function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");

  useEffect(() => {
    const initial = storedLocale();
    setLocaleState(initial);
    document.documentElement.lang = initial === "np" ? "ne" : "en";
  }, []);

  const setLocale = (l: Locale) => {
    setLocaleState(l);
    window.localStorage.setItem(STORAGE_KEY, l);
    document.documentElement.lang = l === "np" ? "ne" : "en";
  };

  return (
    <LocaleCtx.Provider value={{ locale, setLocale }}>
      <NextIntlClientProvider locale={locale} messages={messages[locale]}>
        {children}
      </NextIntlClientProvider>
    </LocaleCtx.Provider>
  );
}