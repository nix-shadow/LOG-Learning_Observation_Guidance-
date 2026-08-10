import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Navigation from "@/components/Navigation";
import { Toaster } from "react-hot-toast";
import PageTransition from "@/components/PageTransition";
import { AuthProvider } from "@/context/AuthContext";
import OfflineBanner from "@/components/OfflineBanner";
import InstallPrompt from "@/components/InstallPrompt";

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
    <html lang="en">
      <body className={`${inter.variable} min-h-screen flex flex-col bg-brand-white text-brand-text font-sans antialiased`}>
        <AuthProvider>
          <OfflineBanner />
          <Navigation />
          <main className="flex-1 flex flex-col w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 relative">
            <PageTransition>
              {children}
            </PageTransition>
          </main>
          <Toaster
            position="bottom-center"
            toastOptions={{
              style: {
                background: '#333',
                color: '#fff',
                borderRadius: '999px',
              },
              success: {
                style: {
                  background: '#00B4D8',
                },
              },
            }}
          />
          <InstallPrompt />
        </AuthProvider>
      </body>
    </html>
  );
}
