import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig, loadEnv, type Plugin } from "vite";

// Plugin to generate version.lock file after build
function versionLockPlugin(): Plugin {
  return {
    name: "version-lock",
    apply: "build",
    closeBundle() {
      const distDir = fileURLToPath(new URL("./dist", import.meta.url));
      const rootPkgPath = fileURLToPath(
        new URL("../../package.json", import.meta.url)
      );
      const rootPkg = JSON.parse(readFileSync(rootPkgPath, "utf-8"));
      const version = rootPkg.version || "0.0.0";

      mkdirSync(distDir, { recursive: true });
      writeFileSync(`${distDir}/version.lock`, version);
    },
  };
}

function runtimeConfigPlugin(config: {
  apiBaseURL: string;
  apiPrefix: string;
  authCookieName: string;
}): Plugin {
  return {
    name: "runtime-config",
    apply: "build",
    closeBundle() {
      const distDir = fileURLToPath(new URL("./dist", import.meta.url));
      const runtimeConfig = `window.__APP_CONFIG__ = ${JSON.stringify(
        {
          API_BASE_URL: config.apiBaseURL,
          API_PREFIX: config.apiPrefix,
          AUTH_COOKIE_NAME: config.authCookieName,
        },
        null,
        2
      )};\n`;

      mkdirSync(distDir, { recursive: true });
      writeFileSync(`${distDir}/runtime-config.js`, runtimeConfig);
    },
  };
}

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const devProxyTarget =
    process.env.API_PROXY_TARGET || env.API_PROXY_TARGET || env.API_BASE_URL;
  const devProxyOrigin = process.env.API_PROXY_ORIGIN || env.API_PROXY_ORIGIN;
  const devtoolsPort = Number(env.TANSTACK_DEVTOOLS_PORT || 42_071);
  const apiBaseURL =
    process.env.VITE_API_BASE_URL ?? env.VITE_API_BASE_URL ?? "";
  const configuredApiPrefix =
    process.env.VITE_API_PREFIX ?? env.VITE_API_PREFIX;
  const apiPrefix =
    configuredApiPrefix === undefined
      ? apiBaseURL
        ? ""
        : "/api"
      : configuredApiPrefix;
  const authCookieName =
    process.env.VITE_AUTH_COOKIE_NAME ??
    env.VITE_AUTH_COOKIE_NAME ??
    "alpha_admin_authorization";

  return {
    base: "./",
    plugins: [
      devtools({ eventBusConfig: { port: devtoolsPort } }),
      tanstackRouter({
        target: "react",
        autoCodeSplitting: true,
      }),
      viteReact(),
      tailwindcss(),
      versionLockPlugin(),
      runtimeConfigPlugin({ apiBaseURL, apiPrefix, authCookieName }),
    ],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
    server: {
      proxy: {
        "/api": {
          target: devProxyTarget || "http://127.0.0.1:27001",
          changeOrigin: true,
          secure: false,
          configure: (proxy) => {
            if (!devProxyOrigin) return;
            proxy.on("proxyReq", (proxyRequest) => {
              if (devProxyOrigin === "omit") {
                proxyRequest.removeHeader("Origin");
                return;
              }
              proxyRequest.setHeader("Origin", devProxyOrigin);
            });
          },
          rewrite: (path) => path.replace(/^\/api/, ""),
        },
      },
    },
    define: {
      __TANSTACK_DEVTOOLS_PORT__: JSON.stringify(devtoolsPort),
    },
    build: {
      assetsDir: "static",
      // Monaco is route-level lazy loaded and kept intact to preserve its
      // initialization graph. Release packaging enforces a separate budget.
      chunkSizeWarningLimit: 4096,
    },
  };
});
