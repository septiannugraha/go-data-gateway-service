import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  output: 'export', // Enable static export for Cloudflare Pages
  reactStrictMode: true,
  images: {
    unoptimized: true, // Disable image optimization for static builds
  },
  trailingSlash: true, // Add trailing slashes for better static hosting compatibility
};

export default withMDX(config);
