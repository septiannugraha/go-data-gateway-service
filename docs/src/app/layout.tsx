import '@/app/global.css';
import { RootProvider } from 'fumadocs-ui/provider';
import { Inter } from 'next/font/google';
import type { ReactNode } from 'react';
import type { Metadata } from 'next';

const inter = Inter({
  subsets: ['latin'],
});

export const metadata: Metadata = {
  title: {
    default: 'INAPROC API Gateway',
    template: '%s | INAPROC API Gateway'
  },
  description: 'Dokumentasi lengkap untuk INAPROC API Gateway',
  icons: {
    icon: '/img/apigw_favicon.png',
    shortcut: '/img/apigw_favicon.png',
    apple: '/img/apigw_favicon.png',
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={inter.className} suppressHydrationWarning>
      <body className="flex flex-col min-h-screen">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
