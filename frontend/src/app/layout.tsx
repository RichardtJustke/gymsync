import type { Metadata, Viewport } from "next";
import { Manrope } from "next/font/google";
import "./globals.css";
import Navbar from "@/components/Navbar";
import FirstAccessGate from "@/components/FirstAccessGate";
import PushRegistrar from "@/components/PushRegistrar";

const manrope = Manrope({
  subsets: ["latin"],
  variable: "--font-sans",
  display: "swap",
});

import { ToastProvider } from "@/context/ToastContext";
import { ThemeProvider } from "@/context/ThemeContext";

export const viewport: Viewport = {
  themeColor: "#0a0b0d",
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  viewportFit: "cover",
};

export const metadata: Metadata = {
  title: "Unika GymSync — Seu Treino Inteligente",
  description: "Treinos, fichas, evolução corporal e check-in na palma da sua mão.",
  manifest: "/manifest.json",
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: "GymSync",
  },
  icons: {
    icon: "/logo-192.png",
    apple: "/logo-192.png",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="pt-BR" className={manrope.variable}>
      <head>
        <link rel="manifest" href="/manifest.json" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
        <script
          dangerouslySetInnerHTML={{
            __html: `
              if ('serviceWorker' in navigator && window.location.protocol === 'http:' || window.location.protocol === 'https:') {
                window.addEventListener('load', function() {
                  navigator.serviceWorker.register('/sw.js').catch(function() {});
                });
              }
            `,
          }}
        />
      </head>
      <body className="min-h-screen bg-ink-950 text-ink-100 flex flex-col font-sans transition-colors duration-200 select-none touch-manipulation">
        <ThemeProvider>
          <ToastProvider>
            <Navbar />
            <FirstAccessGate />
            <PushRegistrar />
            <main className="flex-1 flex flex-col">{children}</main>
          </ToastProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
