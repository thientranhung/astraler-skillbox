import React from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import { LayoutDashboard, Library, Globe, FolderGit2, Puzzle, Settings, Info } from "lucide-react";

export const NAV_ITEMS = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/skills", label: "Host Skills", icon: Library },
  { to: "/global", label: "Global Skills", icon: Globe },
  { to: "/plugins", label: "Global Plugins", icon: Puzzle },
  { to: "/projects", label: "Projects", icon: FolderGit2 },
  { to: "/settings", label: "Settings", icon: Settings },
  { to: "/about", label: "About", icon: Info },
] as const;

type NavTo = typeof NAV_ITEMS[number]["to"];

const SECTIONS: Array<{ label?: string; routes: NavTo[] }> = [
  { routes: ["/dashboard"] },
  { label: "Skills", routes: ["/skills", "/global"] },
  { label: "Plugins", routes: ["/plugins"] },
  { routes: ["/projects", "/settings", "/about"] },
];

export function Sidebar(): React.JSX.Element {
  const location = useRouterState({ select: (s) => s.location });
  const navByRoute = Object.fromEntries(NAV_ITEMS.map((item) => [item.to, item]));

  return (
    <nav className="flex w-44 flex-col border-r border-zinc-200 bg-zinc-50">
      <div className="px-3 py-3 text-xs font-semibold uppercase tracking-wider text-zinc-400">
        Skillbox
      </div>
      {SECTIONS.map((section, idx) => (
        <React.Fragment key={idx}>
          {section.label != null && (
            <div className="mt-1 px-3 pb-0.5 pt-2 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              {section.label}
            </div>
          )}
          {idx > 0 && section.label == null && (
            <div className="mx-3 my-1 border-t border-zinc-200" />
          )}
          {section.routes.map((to) => {
            const item = navByRoute[to];
            const Icon = item.icon;
            const active = location.pathname.startsWith(to);
            return (
              <Link
                key={to}
                to={to}
                className={`flex items-center gap-2 px-3 py-2 text-sm ${
                  active
                    ? "bg-zinc-200 font-medium text-zinc-900"
                    : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900"
                }`}
              >
                <Icon size={15} />
                {item.label}
              </Link>
            );
          })}
        </React.Fragment>
      ))}
    </nav>
  );
}
