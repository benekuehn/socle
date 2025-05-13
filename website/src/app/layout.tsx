import type { Metadata } from "next";
import { GeistSans } from 'geist/font/sans';
import { GeistMono } from 'geist/font/mono';
import "./globals.css";
import Header from "@/components/Header";
import { SoftwareApplication, WithContext } from 'schema-dts'

export const metadata: Metadata = {
  metadataBase: new URL('https://socle.dev'),

  title: {
    template: '%s | Socle',
    default: 'Socle: Effortless Stacked Branches for Git & GitHub',
  },
  description: 'Socle enables developers to master stacked branches (stacked diffs) in Git. Experience focused PRs, better reviews, and continuous productive flow on GitHub. Considered, enabling, open-source.',
  keywords: ['stacked branches', 'stacked diffs', 'stacked changes', 'git', 'github', 'pull requests', 'code review', 'developer workflow', 'git workflow', 'socle', 'open source', 'cli', 'git tool', 'graphite', 'ghstack'],
  icons: [
    {
      rel: 'icon',
      url: '/favicon-dark.png',
      media: '(prefers-color-scheme: light)',
      sizes: '512x512',
      type: 'image/png',
    },
    {
      rel: 'icon',
      url: '/favicon-light.png',
      media: '(prefers-color-scheme: dark)',
      sizes: '512x512',
      type: 'image/png',
    },
  ],
  openGraph: {
    url: 'https://socle.dev', // Canonical URL for the page
    siteName: 'Socle',
    images: [
      {
        url: '/og-image-socle.png', // Path to your OG image (e.g., 1200x630px)
        width: 1200,
        height: 630,
        alt: 'Socle Logo and tagline: Effortless Stacked Branches for Git & GitHub',
      },
    ],
    locale: 'en_US', // Specify language/region
    type: 'website', // Use 'article' for blog posts/docs pages
  },
}

const jsonLd: WithContext<SoftwareApplication> = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'Socle',
  applicationCategory: 'DeveloperApplication',
  operatingSystem: 'macOS',
  description: metadata.description!, // Reuse description
  url: 'https://socle.dev',
  maintainer: '@benekuehn',
  downloadUrl: 'https://github.com/benekuehn/socle/releases'

}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <head>
        {/* Add JSON-LD to your page */}
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
      </head>
      <body className={`font-sans antialiased`}>
        <Header />
        <main className="pt-16">
          {children}
        </main>
      </body>
    </html>
  );
}
