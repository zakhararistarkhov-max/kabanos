import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "Kabanos — здоровье под контролем",
  description: "Вода, вес, калории, тренировки и таблетки — в одном месте.",
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: dark)", color: "#0b1120" },
    { media: "(prefers-color-scheme: light)", color: "#f8fafc" },
  ],
  width: "device-width",
  initialScale: 1,
};

// Runs before paint to set the theme from the saved choice or the OS preference,
// avoiding a flash of the wrong theme.
const themeInit = `(function(){try{var t=localStorage.getItem('kabanos:theme');if(t!=='light'&&t!=='dark'){t=window.matchMedia('(prefers-color-scheme: light)').matches?'light':'dark';}document.documentElement.dataset.theme=t;}catch(e){document.documentElement.dataset.theme='dark';}})();`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" suppressHydrationWarning>
      <body>
        <script dangerouslySetInnerHTML={{ __html: themeInit }} />
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
