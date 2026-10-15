/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  async redirects() {
    return [
      ...["/notes", "/files"].map(source => ({ source, destination: "/account", permanent: false })),
      ...["/examples/notes-to-markdown", "/examples/private-image-upload"].flatMap(source => [
        { source, destination: "/examples", permanent: false },
        { source: `/zh${source}`, destination: "/zh/examples", permanent: false }
      ])
    ];
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" }
        ]
      },
      ...["account", "dashboard", "billing", "referrals", "credits", "files", "notes", "admin", "login", "invite", "oauth", "api"].map(segment => ({
        source: `/${segment}/:path*`,
        headers: [{ key: "X-Robots-Tag", value: "noindex, nofollow" }]
      }))
    ];
  }
};

export default nextConfig;
