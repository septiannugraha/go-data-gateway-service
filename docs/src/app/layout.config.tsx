import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { Book, ComponentIcon, Pencil, PlusIcon, Server } from 'lucide-react';

/**
 * Shared layout configurations
 *
 * you can customise layouts individually from:
 * Home Layout: app/(home)/layout.tsx
 * Docs Layout: app/docs/layout.tsx
 */
export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <>
        <svg
          width="28"
          height="28"
          xmlns="http://www.w3.org/2000/svg"
          aria-label="INAPROC Logo"
          className="text-fd-primary"
        >
          <defs>
            <linearGradient id="logo-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="currentColor" />
              <stop offset="100%" stopColor="currentColor" stopOpacity="0.8" />
            </linearGradient>
          </defs>
          <rect x="2" y="2" width="24" height="24" rx="6" fill="url(#logo-gradient)" />
          <path
            d="M8 8h8v2H8V8zm0 4h8v2H8v-2zm0 4h6v2H8v-2z"
            fill="white"
          />
        </svg>
        <span className="font-bold text-fd-primary">INAPROC</span>
        <span className="text-fd-muted-foreground text-sm font-medium">API Gateway</span>
      </>
    ),
  },
  // Navigation links are now configured in individual layouts
  // see https://fumadocs.dev/docs/ui/navigation/links
};
