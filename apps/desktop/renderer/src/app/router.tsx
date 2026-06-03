import React from "react";
import {
  createRouter,
  createRoute,
  createRootRoute,
  createMemoryHistory,
  Outlet,
  useNavigate,
} from "@tanstack/react-router";
import { useAppSettings } from "../features/app-settings/use-app-settings.js";
import { AppShell } from "../components/app-shell.js";
import { StartupErrorScreen } from "../screens/startup-error-screen.js";
import { SetupScreen } from "../screens/setup-screen.js";
import { DashboardScreen } from "../screens/dashboard-screen.js";
import { SkillsLibraryScreen } from "../screens/skills-library-screen.js";
import { SettingsScreen } from "../screens/settings-screen.js";
import { ProjectsScreen } from "../screens/projects-screen.js";
import { ProjectDetailScreen } from "../screens/project-detail-screen.js";
import { SkillDetailScreen } from "../screens/skill-detail-screen.js";
import { GlobalSkillsScreen } from "../screens/global-skills-screen.js";
import { PluginsScreen } from "../screens/plugins-screen.js";
import { AboutScreen } from "../screens/about-screen.js";

type StartupCheckState = "pending" | "clear";

// Root — bare layout with no shell.
//
// Gates <Outlet /> (and all core-dependent child routes) until the pre-ready
// startup-error query resolves. Without the gate, IndexRedirector mounts
// immediately, calls useAppSettings(), hits isError (Go is not running), and
// navigates to /setup — racing or overwriting the /startup-error navigation.
//
// Two states:
//   pending — getStartupError IPC in-flight; render a spinner, no child routes
//   clear   — no pre-ready error; render <Outlet /> normally
//
// Separate onEvent subscription handles mid-run fatal crashes (Go dies after
// server.ready when the window is already open and child routes are active).
function RootLayout(): React.JSX.Element {
  const navigate = useNavigate();

  // Start as "pending" only when the handler is available (Electron runtime).
  // In test/storybook environments where window.core.getStartupError is absent,
  // start as "clear" so child routes render immediately.
  const [startupCheck, setStartupCheck] = React.useState<StartupCheckState>(
    () => (window.core?.getStartupError ? "pending" : "clear"),
  );

  React.useEffect(() => {
    if (!window.core?.getStartupError) return;
    void (async () => {
      const msg = await window.core!.getStartupError!();
      // Navigate first, then clear the gate so <Outlet /> renders the already-routed
      // /startup-error screen. /startup-error is a child of RootLayout and needs
      // <Outlet /> to be rendered.
      if (msg) {
        await navigate({ to: "/startup-error", replace: true, search: { message: msg } });
      }
      setStartupCheck("clear");
    })();
  }, [navigate]);

  // Mid-run fatal: Go crashed after server.ready. Main sends this as a core:event.
  React.useEffect(() => {
    if (!window.core) return;
    return window.core.onEvent("startup.error", (params) => {
      void navigate({
        to: "/startup-error",
        replace: true,
        search: { message: (params as { message?: string }).message ?? "Unknown error" },
      });
    });
  }, [navigate]);

  if (startupCheck === "pending") {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-700" />
      </div>
    );
  }

  return <Outlet />;
}

// Index route: read settings and redirect
export function IndexRedirector(): React.JSX.Element {
  const { data, isPending, isError } = useAppSettings();
  const navigate = useNavigate();

  React.useEffect(() => {
    if (isPending) return;
    if (isError || data?.activeHost == null) {
      navigate({ to: "/setup", replace: true });
    } else {
      navigate({ to: "/dashboard", replace: true });
    }
  }, [isPending, isError, data, navigate]);

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="h-6 w-6 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-700" />
    </div>
  );
}

const rootRoute = createRootRoute({ component: RootLayout });

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: IndexRedirector,
});

const startupErrorRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/startup-error",
  validateSearch: (search: Record<string, unknown>) => ({
    message: typeof search["message"] === "string" ? search["message"] : "Unknown startup error",
  }),
  component: StartupErrorScreen,
});

const setupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/setup",
  component: SetupScreen,
});

// Shell layout wraps /skills and /settings
const shellRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "shell",
  component: AppShell,
});

const dashboardRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/dashboard",
  component: DashboardScreen,
});

const skillsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/skills",
  component: SkillsLibraryScreen,
});

const projectsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/projects",
  component: ProjectsScreen,
});

const projectDetailRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/projects/$projectId",
  component: ProjectDetailScreen,
});

const skillDetailRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/skills/$skillId",
  component: SkillDetailScreen,
});

const globalRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/global",
  component: GlobalSkillsScreen,
});

const pluginsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/plugins",
  component: PluginsScreen,
});

const settingsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/settings",
  component: SettingsScreen,
});

const aboutRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/about",
  component: AboutScreen,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  startupErrorRoute,
  setupRoute,
  shellRoute.addChildren([dashboardRoute, skillsRoute, skillDetailRoute, globalRoute, projectsRoute, projectDetailRoute, pluginsRoute, settingsRoute, aboutRoute]),
]);

export const router = createRouter({
  routeTree,
  history: createMemoryHistory({ initialEntries: ["/"] }),
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
