import type { Metadata } from "next";
import { Overpass, Overpass_Mono, Raleway } from "next/font/google";
import { Header } from "@/components/Header";
import { ToastProvider } from "@/components/ui";
import "./globals.css";

// Raleway is the app font. Overpass is kept only for the printed booklet.
const raleway = Raleway({ variable: "--font-raleway", subsets: ["latin"] });
const overpass = Overpass({ variable: "--font-overpass", subsets: ["latin"] });
const overpassMono = Overpass_Mono({ variable: "--font-overpass-mono", subsets: ["latin"] });

export const metadata: Metadata = {
  title: "King's High School Athletics",
  description: "Organise your school athletics day: entries, schedule and printable booklet.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${raleway.variable} ${overpass.variable} ${overpassMono.variable} h-full`}>
      <body className="min-h-full bg-white text-zinc-700">
        <ToastProvider>
          <Header />
          <main className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 print:max-w-none print:p-0">{children}</main>
        </ToastProvider>
      </body>
    </html>
  );
}
