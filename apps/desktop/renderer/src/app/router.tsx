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
import { SetupScreen } from "../screens/setup-screen.js";
import { SkillsLibraryScreen } from "../screens/skills-library-screen.js";
import { SettingsScreen } from "../screens/settings-screen.js";
import { ProjectsScreen } from "../screens/projects-screen.js";
import { ProjectDetailScreen } from "../screens/project-detail-screen.js";

// Root — bare layout with no shell
function RootLayout(): React.JSX.Element {
  return <Outlet />;
}

// Index route: read settings and redirect
function IndexRedirector(): React.JSX.Element {
  const { data, isPending, isError } = useAppSettings();
  const navigate = useNavigate();

  React.useEffect(() => {
    if (isPending) return;
    if (isError || data?.activeHost == null) {
      navigate({ to: "/setup", replace: true });
    } else {
      navigate({ to: "/skills", replace: true });
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

const settingsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: "/settings",
  component: SettingsScreen,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  setupRoute,
  shellRoute.addChildren([skillsRoute, projectsRoute, projectDetailRoute, settingsRoute]),
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
