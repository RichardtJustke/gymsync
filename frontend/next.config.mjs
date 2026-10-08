/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Build estático (pasta out/) consumido pelo Capacitor: NEXT_EXPORT=1 npm run build
  ...(process.env.NEXT_EXPORT ? { output: 'export', images: { unoptimized: true }, trailingSlash: true } : {}),
};

export default nextConfig;
