import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Navigation from "@/components/Navigation";
import { Toaster } from "react-hot-toast";
import { LazyMotion, domAnimation, MotionConfig } from "framer-motion";
import PageTransition from "@/components/PageTransition";
import { AuthProvider } from "@/context/AuthContext";
import InstallPrompt from "@/components/InstallPrompt";
import ThreeBackground from "@/components/ThreeBackground";
import SyncIsland from "@/components/SyncIsland";
import dynamic from "next/dynamic";
import { ThemeProvider } from "@/components/ThemeProvider";

// WP-0.3 bundle research round: CommandPalette (cmdk + lucide) is now loaded
// on demand — the palette only renders its trigger until the first ⌘K or click.
const CommandPalette = dynamic(() => import("@/components/CommandPalette"), {
  ssr: false,
  loading: () => null,
});

const inter = Inter({ subsets: ["latin"], variable: '--font-inter' });

export const metadata: Metadata = {
  title: "LOG | Learning Observation Guidance",
  description: "A smart learning companion for low-connectivity regions.",
  manifest: "/manifest.json",
  icons: {
    apple: [{ url: "/icons/icon-192.png", sizes: "192x192", type: "image/png" }],
  },
};

export const viewport = {
  themeColor: "#00B4D8",
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${inter.variable} min-h-screen flex flex-col bg-white dark:bg-brand-darker text-gray-900 dark:text-brand-text font-sans antialiased overflow-x-hidden transition-colors duration-300`}>
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false}>
          {/* WP-0.3 a11y research round: skip link surfaces on keyboard focus
              (WCAG 2.4.1) so users who cannot use a mouse bypass the nav. */}
          <a
            href="#main-content"
            className="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-[100] focus:bg-brand-neon focus:text-black focus:px-4 focus:py-2 focus:rounded-xl focus:font-semibold focus:shadow-glow"
          >
            Skip to content
          </a>
          {/* WP-0.3 bundle research round: LazyMotion + domAnimation keeps
              the framer-motion animation features out of the initial bundle —
              `m.*` components across the app resolve to them on demand. */}
          <LazyMotion features={domAnimation} strict={false}>
          <MotionConfig reducedMotion="user">
            <ThreeBackground />
            <AuthProvider>
            <SyncIsland />
            <CommandPalette />
            <Navigation />
            <main id="main-content" className="flex-1 flex flex-col w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 relative z-10" tabIndex={-1}>
              <PageTransition>
                {children}
              </PageTransition>
            </main>
            <Toaster
              position="bottom-center"
              // WP-0.3 a11y research round: pause-on-hover is built into
              // react-hot-toast's Toaster (container onMouseEnter/Leave),
              // verified in the installed 2.6.0 source — no prop needed.
              toastOptions={{
                style: {
                  background: 'var(--toast-bg, rgba(0,0,0,0.8))',
                  color: 'var(--toast-text, #fff)',
                  borderRadius: '16px',
                  border: '1px solid var(--toast-border, rgba(255,255,255,0.1))',
                  backdropFilter: 'blur(16px)',
                },
                success: {
                  style: {
                    background: 'var(--toast-success-bg, rgba(0,180,216,0.2))',
                    border: '1px solid var(--toast-success-border, rgba(0,180,216,0.5))',
                  },
                },
              }}
            />
            <InstallPrompt />
            </AuthProvider>
          </MotionConfig>
          </LazyMotion>
        </ThemeProvider>
      </body>
    </html>
  );
}
