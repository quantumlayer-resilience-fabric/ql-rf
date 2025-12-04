import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Enable standalone output for Docker deployment
  output: "standalone",

  // Environment variables available at runtime
  // Default to localhost for local development
  // Override with NEXT_PUBLIC_API_URL and NEXT_PUBLIC_ORCHESTRATOR_URL for production
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080",
    NEXT_PUBLIC_ORCHESTRATOR_URL: process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || "http://localhost:8083",
  },
};

export default nextConfig;
