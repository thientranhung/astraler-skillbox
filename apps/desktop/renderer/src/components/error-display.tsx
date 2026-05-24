import React from "react";
import { AlertCircle } from "lucide-react";
import type { AppClientError } from "../lib/core-client/client.js";

interface ErrorDisplayProps {
  error: AppClientError | Error | unknown;
  className?: string;
}

function getMessage(error: unknown): string {
  if (error != null && typeof error === "object" && "userMessage" in error) {
    return (error as AppClientError).userMessage;
  }
  if (error instanceof Error) return error.message;
  return "An unexpected error occurred.";
}

export function ErrorDisplay({ error, className = "" }: ErrorDisplayProps): React.JSX.Element {
  return (
    <div
      className={`flex items-start gap-2 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800 ${className}`}
    >
      <AlertCircle size={16} className="mt-0.5 shrink-0" />
      <span>{getMessage(error)}</span>
    </div>
  );
}
