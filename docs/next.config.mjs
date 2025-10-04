import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

const defaultBasePath = '/api';
const normalizedBasePath = (() => {
  const value = process.env.NEXT_PUBLIC_BASE_PATH ?? defaultBasePath;
  if (value === '' || value === '/') {
    return '';
  }

  return value.startsWith('/') ? value.replace(/\/$/, '') : `/${value.replace(/\/$/, '')}`;
})();

const assetPrefix = process.env.NEXT_PUBLIC_ASSET_PREFIX
  ?? (normalizedBasePath ? normalizedBasePath : undefined);

/** @type {import('next').NextConfig} */
const config = {
  // Server-first approach - removed static export
  reactStrictMode: true,
  basePath: normalizedBasePath,
  assetPrefix,
  async redirects() {
    if (!normalizedBasePath) {
      return [];
    }

    return [
      {
        source: '/',
        destination: normalizedBasePath,
        permanent: false,
      },
    ];
  },
  // Re-enable image optimization for SSR
};

export default withMDX(config);
