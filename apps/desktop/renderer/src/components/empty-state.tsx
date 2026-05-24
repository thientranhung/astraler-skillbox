import React from "react";

interface EmptyStateProps {
  message: string;
  description?: string;
}

export function EmptyState({ message, description }: EmptyStateProps): React.JSX.Element {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <p className="text-sm font-medium text-zinc-500">{message}</p>
      {description && <p className="mt-1 text-xs text-zinc-400">{description}</p>}
    </div>
  );
}
