import React from "react";
import { useSearch } from "@tanstack/react-router";
import { AlertCircle } from "lucide-react";

export function StartupErrorScreen(): React.JSX.Element {
  const { message } = useSearch({ from: "/startup-error" });

  return (
    <div className="flex h-screen flex-col items-center justify-center bg-zinc-50 p-8">
      <div className="w-full max-w-lg rounded-xl border border-red-200 bg-white p-8 shadow-sm">
        <div className="mb-4 flex items-center gap-3">
          <AlertCircle className="shrink-0 text-red-500" size={24} />
          <h1 className="text-base font-semibold text-zinc-900">Skillbox failed to start</h1>
        </div>
        <p className="mb-4 text-sm text-zinc-600">
          The Skillbox core process could not start. This usually means the database
          is corrupt, incomplete, or from an incompatible version.
        </p>
        <pre className="mb-6 overflow-auto rounded-lg bg-zinc-100 p-3 font-mono text-xs text-zinc-700 whitespace-pre-wrap break-words">
          {message}
        </pre>
        <p className="mb-6 text-xs text-zinc-500">
          If you see a &ldquo;dirty database&rdquo; error, the database may have been interrupted
          during a migration. Do not attempt to use the app until this is resolved.
          Contact support or check the Skillbox documentation for recovery steps.
        </p>
        <button
          onClick={() => window.close()}
          className="cursor-pointer rounded-lg bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700"
        >
          Quit Skillbox
        </button>
      </div>
    </div>
  );
}
