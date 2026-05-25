import React from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import { Library, FolderGit2, Settings } from "lucide-react";

const NAV_ITEMS = [
  { to: "/skills", label: "Skills", icon: Library },
  { to: "/projects", label: "Projects", icon: FolderGit2 },
  { to: "/settings", label: "Settings", icon: Settings },
] as const;

export function Sidebar(): React.JSX.Element {
  const location = useRouterState({ select: (s) => s.location });

  return (
    <nav className="flex w-44 flex-col border-r border-zinc-200 bg-zinc-50">
      <div className="px-3 py-3 text-xs font-semibold uppercase tracking-wider text-zinc-400">
        Skillbox
      </div>
      {NAV_ITEMS.map(({ to, label, icon: Icon }) => {
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
            {label}
          </Link>
        );
      })}
    </nav>
  );
}
