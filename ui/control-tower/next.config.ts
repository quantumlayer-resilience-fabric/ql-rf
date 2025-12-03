import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Enable standalone output for Docker deployment
  output: "standalone",

  // Environment variables available at runtime
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080",
    NEXT_PUBLIC_ORCHESTRATOR_URL: process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || "http://localhost:8083",
  },
};

export default nextConfig;
