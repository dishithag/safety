import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { minioCatalogPlugin } from "./server/minioCatalog";

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

export default defineConfig(({ mode }) => {
  const environment = { ...process.env, ...loadEnv(mode, process.cwd(), "") };

  return {
    plugins: [minioCatalogPlugin(environment), react()],
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
  };
});
