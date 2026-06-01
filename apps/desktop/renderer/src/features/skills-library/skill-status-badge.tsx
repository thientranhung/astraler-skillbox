import React from "react";
import type { SkillListSkill } from "@contracts/index.js";

type Status = SkillListSkill["status"];

const STATUS_CONFIG: Record<Status, { label: string; className: string }> = {
  available: { label: "Available", className: "bg-green-100 text-green-800" },
  missing: { label: "Missing", className: "bg-red-100 text-red-800" },
  unreadable: { label: "Unreadable", className: "bg-red-100 text-red-800" },
  local_modified: { label: "Modified", className: "bg-yellow-100 text-yellow-800" },
  external_symlink: { label: "External Symlink", className: "bg-amber-100 text-amber-800" },
  unknown: { label: "Unknown", className: "bg-zinc-100 text-zinc-600" },
};

interface SkillStatusBadgeProps {
  status: Status;
}

export function SkillStatusBadge({ status }: SkillStatusBadgeProps): React.JSX.Element {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.unknown;
  return (
    <span
      className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${config.className}`}
    >
      {config.label}
    </span>
  );
}
