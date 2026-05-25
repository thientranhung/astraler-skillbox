import React from "react";
import { FolderPlus } from "lucide-react";
import { useProjectsList } from "../features/projects/use-projects-list.js";
import { useAddProject } from "../features/projects/use-add-project.js";
import { ProjectRow } from "../features/projects/project-row.js";
import { ErrorDisplay } from "../components/error-display.js";
import { EmptyState } from "../components/empty-state.js";
import { methods } from "../lib/core-client/methods.js";

export function ProjectsScreen(): React.JSX.Element {
  const { data, isPending, isError, error } = useProjectsList();
  const addProject = useAddProject();

  async function handleAddProject(): Promise<void> {
    const result = await methods.openProjectFolder();
    if (result.path != null) {
      addProject.mutate(result.path);
    }
  }

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold text-zinc-900">Projects</h2>
          {data != null && (
            <p className="mt-0.5 text-xs text-zinc-400">{data.projects.length} project{data.projects.length !== 1 ? "s" : ""}</p>
          )}
        </div>
        <button
          onClick={() => void handleAddProject()}
          disabled={addProject.isPending}
          className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
        >
          <FolderPlus size={13} />
          Add Project
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {isPending && (
          <div className="flex h-40 items-center justify-center">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-700" />
          </div>
        )}

        {isError && (
          <div className="p-4">
            <ErrorDisplay error={error} />
          </div>
        )}

        {!isPending && !isError && data?.projects.length === 0 && (
          <EmptyState
            message="No projects yet"
            description="Add a project folder to start managing skills across your projects."
          />
        )}

        {!isPending && !isError && data != null && data.projects.length > 0 && (
          <table className="w-full text-left">
            <thead className="border-b border-zinc-200 bg-zinc-50">
              <tr>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Project</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Status</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Providers</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Skills</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Warnings</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Last Scanned</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500" />
              </tr>
            </thead>
            <tbody>
              {data.projects.map((project) => (
                <ProjectRow key={project.id} project={project} />
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
