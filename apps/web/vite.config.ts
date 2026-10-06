import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";

const contentSecurityPolicy = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self'",
  "img-src 'self' data:",
  "font-src 'self'",
  "connect-src 'self'",
  "manifest-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "upgrade-insecure-requests",
].join("; ");

function contentSecurityPolicyMeta(): Plugin {
  return {
    name: "content-security-policy-meta",
    apply: "build",
    transformIndexHtml: () => [
      { tag: "meta", attrs: { "http-equiv": "Content-Security-Policy", content: contentSecurityPolicy }, injectTo: "head-prepend" },
    ],
  };
}

export default defineConfig({
  plugins: [react(), contentSecurityPolicyMeta()],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        rewrite: (p) => p.replace(/^\/api/, ""),
      },
      "^/\\.well-known/oauth-": "http://localhost:8080",
      "^/dados/": {
        target: "http://localhost:4443",
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/dados\//, "/diario-dumps/latest/"),
      },
    },
  },
});
