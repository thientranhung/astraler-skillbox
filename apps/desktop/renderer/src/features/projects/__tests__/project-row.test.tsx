// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import React from "react";

vi.mock("@tanstack/react-router", () => ({
  useNavigate: vi.fn(),
}));
vi.mock("../use-scan-project.js", () => ({
  useScanProject: vi.fn(),
}));
vi.mock("../use-open-project-folder.js", () => ({
  useOpenProjectFolder: vi.fn(),
}));
vi.mock("../use-open-project-terminal.js", () => ({
  useOpenProjectTerminal: vi.fn(),
}));
vi.mock("../use-remove-project.js", () => ({
  useRemoveProject: vi.fn(),
}));

import { ProjectRow } from "../project-row.js";
import { useNavigate } from "@tanstack/react-router";
import { useScanProject } from "../use-scan-project.js";
import { useOpenProjectFolder } from "../use-open-project-folder.js";
import { useOpenProjectTerminal } from "../use-open-project-terminal.js";
import { useRemoveProject } from "../use-remove-project.js";
import type { ProjectListItem } from "@contracts/index.js";

const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;
const mockUseScanProject = useScanProject as ReturnType<typeof vi.fn>;
const mockUseOpenProjectFolder = useOpenProjectFolder as ReturnType<typeof vi.fn>;
const mockUseOpenProjectTerminal = useOpenProjectTerminal as ReturnType<typeof vi.fn>;
const mockUseRemoveProject = useRemoveProject as ReturnType<typeof vi.fn>;

const project: ProjectListItem = {
  id: 1,
  name: "demo",
  path: "/repo/demo",
  status: "active",
  providers: [],
  skillCount: 2,
  warningCount: 0,
  lastScannedAt: null,
  pluginEnabledCount: 0,
  pluginTotalCount: 0,
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseNavigate.mockReturnValue(vi.fn());
  mockUseScanProject.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
  mockUseOpenProjectFolder.mockReturnValue({ mutate: vi.fn(), isPending: false });
  mockUseOpenProjectTerminal.mockReturnValue({ mutate: vi.fn(), isPending: false });
  mockUseRemoveProject.mockReturnValue({ mutate: vi.fn(), isPending: false });
});

afterEach(() => cleanup());

const filledProject: ProjectListItem = {
  ...project,
  lastScannedAt: "2025-01-01T00:00:00.000Z",
  providers: [
    { key: "generic_agents", displayName: "Shared Agent Skills", providerStatus: "supported", detectionStatus: "detected", entryCount: 2 },
  ],
};

describe("ProjectRow plugin stats", () => {
  it("renders enabled/total when plugins present", () => {
    render(
      <table>
        <tbody>
          <ProjectRow project={{ ...filledProject, pluginEnabledCount: 2, pluginTotalCount: 5 }} />
        </tbody>
      </table>,
    );
    expect(screen.getByText("2/5")).toBeTruthy();
  });

  it("renders an em dash when no plugins", () => {
    render(
      <table>
        <tbody>
          <ProjectRow project={{ ...filledProject, pluginEnabledCount: 0, pluginTotalCount: 0 }} />
        </tbody>
      </table>,
    );
    expect(screen.getByText("—")).toBeTruthy();
  });
});

describe("ProjectRow", () => {
  it("renders the project path without a separate path sub-row", () => {
    render(
      <table>
        <tbody>
          <ProjectRow project={project} />
        </tbody>
      </table>,
    );

    expect(screen.queryByText("project:")).toBeNull();
    expect(screen.getAllByText("/repo/demo")).toHaveLength(1);
  });

  it("renders skill counts per provider instead of one aggregate count", () => {
    render(
      <table>
        <tbody>
          <ProjectRow
            project={{
              ...project,
              skillCount: 5,
              providers: [
                {
                  key: "generic_agents",
                  displayName: "Shared Agent Skills",
                  providerStatus: "supported",
                  detectionStatus: "detected",
                  entryCount: 2,
                },
                {
                  key: "claude",
                  displayName: "Claude",
                  providerStatus: "experimental",
                  detectionStatus: "detected",
                  entryCount: 3,
                },
              ],
            }}
          />
        </tbody>
      </table>,
    );

    expect(screen.getByTitle("Shared Agent Skills: 2 skills")).toBeTruthy();
    expect(screen.getByTitle("Claude: 3 skills")).toBeTruthy();
    expect(screen.queryByText("5")).toBeNull();
  });
});
