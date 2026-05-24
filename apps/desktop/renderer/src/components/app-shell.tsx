import React from "react";
import { Outlet } from "@tanstack/react-router";
import { Sidebar } from "./sidebar.js";

export function AppShell(): React.JSX.Element {
  return (
    <div className="flex h-screen overflow-hidden bg-white">
      <Sidebar />
      <main className="flex flex-1 flex-col overflow-y-auto">
        <Outlet />
      </main>
    </div>
  );
}
