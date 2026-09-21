import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Em dev, /api é redirecionado para a API local (mesmo contrato do nginx em produção).
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        rewrite: (p) => p.replace(/^\/api/, ""),
      },
    },
  },
});
