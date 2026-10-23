/** @type {import('next').NextConfig} */
const basePath = process.env.NEXT_PUBLIC_BASE_PATH || "";

const nextConfig = {
  reactStrictMode: true,
  // Build estático (pasta out/) consumido pelo Capacitor e pelo GitHub Pages:
  //   NEXT_EXPORT=1 npm run build
  // Para publicar em um subpath (Pages de projeto), defina NEXT_PUBLIC_BASE_PATH=/gymsync
  ...(process.env.NEXT_EXPORT
    ? {
        output: "export",
        images: { unoptimized: true },
        trailingSlash: true,
        ...(basePath ? { basePath } : {}),
      }
    : {}),
};

export default nextConfig;
