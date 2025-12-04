import type { Metadata } from "next";
import { Outfit, Source_Sans_3, JetBrains_Mono } from "next/font/google";
import { ConditionalClerkProvider } from "@/providers/conditional-clerk-provider";
import { QueryProvider } from "@/providers/query-provider";
import "./globals.css";

// Display font - distinctive geometric sans for headings
const outfit = Outfit({
  variable: "--font-display",
  subsets: ["latin"],
  display: "swap",
  weight: ["400", "500", "600", "700"],
});

// Body font - highly legible for data-dense interfaces
const sourceSans = Source_Sans_3({
  variable: "--font-body",
  subsets: ["latin"],
  display: "swap",
  weight: ["400", "500", "600"],
});

// Mono font - excellent for technical data
const jetbrainsMono = JetBrains_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
  display: "swap",
  weight: ["400", "500", "600"],
});

export const metadata: Metadata = {
  title: {
    default: "QuantumLayer Resilience Fabric | Control Tower",
    template: "%s | QL-RF",
  },
  description:
    "One Control Tower for All Your Clouds. Real-time visibility into golden images, patch drift, compliance, and DR readiness across AWS, Azure, GCP, and on-premises infrastructure.",
  keywords: [
    "multi-cloud management",
    "patch drift detection",
    "golden image management",
    "infrastructure compliance",
    "disaster recovery",
    "DevOps automation",
    "AI-powered infrastructure",
  ],
  authors: [{ name: "QuantumLayer" }],
  openGraph: {
    type: "website",
    locale: "en_US",
    url: "https://quantumlayer.io",
    siteName: "QuantumLayer Resilience Fabric",
    title: "QuantumLayer Resilience Fabric | Control Tower",
    description:
      "One Control Tower for All Your Clouds. Real-time visibility into golden images, patch drift, and DR readiness.",
    images: [
      {
        url: "/og-image.png",
        width: 1200,
        height: 630,
        alt: "QuantumLayer Resilience Fabric",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "QuantumLayer Resilience Fabric",
    description: "One Control Tower for All Your Clouds",
    images: ["/og-image.png"],
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <ConditionalClerkProvider>
      <html lang="en" suppressHydrationWarning>
        <body
          className={`${outfit.variable} ${sourceSans.variable} ${jetbrainsMono.variable} font-sans antialiased`}
          style={{ fontFamily: 'var(--font-body), system-ui, sans-serif' }}
        >
          <QueryProvider>{children}</QueryProvider>
        </body>
      </html>
    </ConditionalClerkProvider>
  );
}
