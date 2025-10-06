import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

// No basePath needed - NGINX ingress will strip /api prefix
const normalizedBasePath = '';
const assetPrefix = undefined;

/** @type {import('next').NextConfig} */
const config = {
  // Server-first approach - removed static export
  reactStrictMode: true,
  basePath: normalizedBasePath,
  assetPrefix,
  // Removed redirects - basePath handles routing automatically
  // Disable image optimization to prevent 500 errors in containerized environment
  images: {
    unoptimized: true
  },
};

export default withMDX(config);
