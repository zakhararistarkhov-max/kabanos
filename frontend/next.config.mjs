/** @type {import('next').NextConfig} */
const nextConfig = {
  // Produces a minimal standalone server bundle for a small, fast Docker image.
  output: "standalone",
  reactStrictMode: true,
  poweredByHeader: false,
  experimental: {
    // Keep server-only secrets (backend URL, cookies) out of the client bundle.
    serverActions: { bodySizeLimit: "1mb" },
  },
};

export default nextConfig;
