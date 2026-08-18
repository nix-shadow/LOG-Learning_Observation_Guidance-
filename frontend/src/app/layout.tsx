import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Navigation from "@/components/Navigation";
import { Toaster } from "react-hot-toast";
import PageTransition from "@/components/PageTransition";
import { AuthProvider } from "@/context/AuthContext";
import InstallPrompt from "@/components/InstallPrompt";
import ThreeBackground from "@/components/ThreeBackground";
import SyncIsland from "@/components/SyncIsland";
import CommandPalette from "@/components/CommandPalette";
import { GoogleOAuthProvider } from '@react-oauth/google';

import { ThemeProvider } from "@/components/ThemeProvider";

const inter = Inter({ subsets: ["latin"], variable: '--font-inter' });

export const metadata: Metadata = {
  title: "LOG | Learning Observation Guidance",
  description: "A smart learning companion for low-connectivity regions.",
  manifest: "/manifest.json",
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
          <ThreeBackground />
          <GoogleOAuthProvider clientId={process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || 'YOUR_GOOGLE_CLIENT_ID_HERE'}>
            <AuthProvider>
            <SyncIsland />
            <CommandPalette />
            <Navigation />
            <main className="flex-1 flex flex-col w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 relative z-10">
              <PageTransition>
                {children}
              </PageTransition>
            </main>
            <Toaster
              position="bottom-center"
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
          </GoogleOAuthProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
