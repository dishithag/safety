import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const apiProxy = {
  "/zero-trust-analytics": {
    target: "http://localhost:8080",
    changeOrigin: true,
  },
  "/healthz": {
    target: "http://localhost:8080",
    changeOrigin: true,
  },
};

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    proxy: apiProxy,
  },
  preview: {
    host: "0.0.0.0",
    port: 4173,
    strictPort: true,
    proxy: apiProxy,
  },
});

