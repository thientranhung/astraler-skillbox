import React from "react";
import type { ProjectListItem } from "@contracts/index.js";

type Status = ProjectListItem["status"];

const STATUS_CONFIG: Record<Status, { label: string; className: string }> = {
  active: { label: "Active", className: "bg-green-100 text-green-800" },
  missing: { label: "Missing", className: "bg-red-100 text-red-800" },
  unreadable: { label: "Unreadable", className: "bg-red-100 text-red-800" },
};

interface ProjectStatusBadgeProps {
  status: Status;
}

export function ProjectStatusBadge({ status }: ProjectStatusBadgeProps): React.JSX.Element {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.active;
  return (
    <span
      className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${config.className}`}
    >
      {config.label}
    </span>
  );
}
