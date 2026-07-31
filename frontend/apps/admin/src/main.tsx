import {
  createHashHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import {
  TanStackQueryContext,
  TanStackQueryProvider,
} from "@workspace/ui/integrations/tanstack-query";
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { HelmetProvider } from "react-helmet-async";
// Import the generated route tree
import { routeTree } from "./routeTree.gen";
// Styles
import "@workspace/ui/globals.css";
import "./styles/admin.css";
import { DirectionProvider } from "@workspace/ui/integrations/direction";
import { LanguageProvider } from "@workspace/ui/integrations/language";
import { ThemeProvider } from "@workspace/ui/integrations/theme";
import { configureAuthCookieName } from "@workspace/ui/lib/auth-token";
import { initializeI18n } from "@workspace/ui/lib/i18n";
import { AdminRouteError } from "./components/admin-route-error";
import { AdminRoutePending } from "./components/admin-route-pending";
import { fallbackLng, supportedLngs } from "./config/index.ts";
// Report web vitals
import reportWebVitals from "./reportWebVitals.ts";
// Common utilities
import { Logout } from "./utils/common.ts";

document.documentElement.dataset.app = "admin";

configureAuthCookieName("alpha_admin_authorization");

initializeI18n({
  supportedLngs,
  fallbackLng,
  ns: [
    "ads",
    "announcement",
    "auth-control",
    "auth",
    "components",
    "coupon",
    "dashboard",
    "document",
    "log",
    "membership",
    "menu",
    "nodes",
    "order",
    "payment",
    "product",
    "servers",
    "subscribe",
    "system",
    "ticket",
    "tool",
    "translation",
    "user",
    "xray-templates",
  ],
});

window.logout = Logout;

// Create a new router instance
const TanStackQueryProviderContext = TanStackQueryContext();
TanStackQueryProviderContext.queryClient.setDefaultOptions({
  queries: {
    gcTime: 5 * 60_000,
    retry: 1,
    staleTime: 15_000,
  },
});
const hashHistory = createHashHistory();
const router = createRouter({
  routeTree,
  history: hashHistory,
  context: {
    ...TanStackQueryProviderContext,
  },
  defaultErrorComponent: AdminRouteError,
  defaultPendingComponent: AdminRoutePending,
  defaultPendingMinMs: 300,
  defaultPendingMs: 150,
  defaultPreload: "intent",
  scrollRestoration: true,
  defaultStructuralSharing: true,
  defaultPreloadStaleTime: 0,
});

// Register the router instance for type safety
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

// Render the app
const rootElement = document.getElementById("app");
if (rootElement && !rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <StrictMode>
      <TanStackQueryProvider {...TanStackQueryProviderContext}>
        <HelmetProvider>
          <LanguageProvider supportedLanguages={supportedLngs}>
            <ThemeProvider defaultTheme="light">
              <DirectionProvider>
                <RouterProvider router={router} />
              </DirectionProvider>
            </ThemeProvider>
          </LanguageProvider>
        </HelmetProvider>
      </TanStackQueryProvider>
    </StrictMode>
  );
}

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
