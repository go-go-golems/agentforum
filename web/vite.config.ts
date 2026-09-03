import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig } from "vite";

// Adapted from publish-vault's vite.config.ts (design §6.1): the vault-only
// plugins (manus runtime collector, jsx-loc) are dropped; the react +
// tailwind v4 pipeline is identical.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "src"),
    },
  },
  server: {
    // dev proxy: the UI always talks to the local agentforum server
    proxy: {
      "/v1": {
        target: process.env.AGENTFORUM_URL ?? "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
  },
});
