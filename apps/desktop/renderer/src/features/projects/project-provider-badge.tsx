import React from "react";
import type { ProjectListProviderSummary } from "@contracts/index.js";

const DETECTION_CLASS: Record<ProjectListProviderSummary["detectionStatus"], string> = {
  detected: "bg-blue-100 text-blue-800",
  missing: "bg-zinc-100 text-zinc-500",
  invalid_structure: "bg-yellow-100 text-yellow-700",
};

interface ProjectProviderBadgeProps {
  provider: ProjectListProviderSummary;
}

export function ProjectProviderBadge({ provider }: ProjectProviderBadgeProps): React.JSX.Element {
  const cls = DETECTION_CLASS[provider.detectionStatus] ?? DETECTION_CLASS.missing;
  return (
    <span className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${cls}`}>
      {provider.displayName}
    </span>
  );
}
