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

describe("ProjectRow", () => {
  it("renders a full project path sub-row for scanning", () => {
    render(
      <table>
        <tbody>
          <ProjectRow project={project} />
        </tbody>
      </table>,
    );

    expect(screen.getByText("project:")).toBeTruthy();
    expect(screen.getAllByText("/repo/demo").length).toBeGreaterThan(1);
  });
});
