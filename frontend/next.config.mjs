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
  // Conservative security headers for every response served by the frontend.
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
          { key: "Strict-Transport-Security", value: "max-age=31536000; includeSubDomains" },
        ],
      },
    ];
  },
};

export default nextConfig;
