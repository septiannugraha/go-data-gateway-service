// This file configures edge runtime for specific routes only
import { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  // Let the request continue normally
  return;
}

export const config = {
  matcher: [
    // Apply to API routes only, not to docs pages
    '/api/(.*)',
  ],
};
