import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock invoke before importing methods
vi.mock("../client.js", () => ({
  invoke: vi.fn(),
  AppClientError: class extends Error {
    constructor(public code: string, public userMessage: string, public technicalMessage: string) {
      super(userMessage);
    }
  },
}));

import { methods } from "../methods.js";
import { invoke } from "../client.js";

const mockInvoke = invoke as ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.clearAllMocks();
  mockInvoke.mockResolvedValue({});
});

describe("methods.getSettings", () => {
  it("calls settings.get with empty params", async () => {
    await methods.getSettings();
    expect(mockInvoke).toHaveBeenCalledWith("settings.get", {});
  });
});

describe("methods.chooseHost", () => {
  it("calls host.choose with path", async () => {
    await methods.chooseHost({ path: "/tmp/host" });
    expect(mockInvoke).toHaveBeenCalledWith("host.choose", { path: "/tmp/host" });
  });
});

describe("methods.scanHost", () => {
  it("calls host.scan with hostId", async () => {
    await methods.scanHost({ hostId: 7 });
    expect(mockInvoke).toHaveBeenCalledWith("host.scan", { hostId: 7 });
  });
});

describe("methods.listSkills", () => {
  it("calls skill.list with hostId", async () => {
    await methods.listSkills({ hostId: 3 });
    expect(mockInvoke).toHaveBeenCalledWith("skill.list", { hostId: 3 });
  });
});

describe("methods.cancelOperation", () => {
  it("calls operation.cancel with operationId", async () => {
    await methods.cancelOperation({ operationId: 99 });
    expect(mockInvoke).toHaveBeenCalledWith("operation.cancel", { operationId: 99 });
  });
});

describe("methods.openHostFolder", () => {
  it("calls dialog.openHostFolder with empty params", async () => {
    await methods.openHostFolder();
    expect(mockInvoke).toHaveBeenCalledWith("dialog.openHostFolder", {});
  });
});

describe("methods.openProjectFolder", () => {
  it("calls dialog.openProjectFolder with empty params", async () => {
    await methods.openProjectFolder();
    expect(mockInvoke).toHaveBeenCalledWith("dialog.openProjectFolder", {});
  });
});

describe("methods.addProject", () => {
  it("calls project.add with path", async () => {
    await methods.addProject({ path: "/home/user/myproject" });
    expect(mockInvoke).toHaveBeenCalledWith("project.add", { path: "/home/user/myproject" });
  });
});

describe("methods.listProjects", () => {
  it("calls project.list with empty params", async () => {
    await methods.listProjects();
    expect(mockInvoke).toHaveBeenCalledWith("project.list", {});
  });
});

describe("methods.getProject", () => {
  it("calls project.get with projectId", async () => {
    await methods.getProject({ projectId: 5 });
    expect(mockInvoke).toHaveBeenCalledWith("project.get", { projectId: 5 });
  });
});

describe("methods.scanProject", () => {
  it("calls project.scan with projectId", async () => {
    await methods.scanProject({ projectId: 3 });
    expect(mockInvoke).toHaveBeenCalledWith("project.scan", { projectId: 3 });
  });
});
