import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  // Server-first approach - removed static export
  reactStrictMode: true,
  // Re-enable image optimization for SSR
};

export default withMDX(config);
