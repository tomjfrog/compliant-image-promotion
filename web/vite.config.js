import { defineConfig } from "vite";

// During local development, proxy /api calls to the Go back-end on :8080.
// In production the Go server serves the built assets, so no proxy is needed.
export default defineConfig({
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
