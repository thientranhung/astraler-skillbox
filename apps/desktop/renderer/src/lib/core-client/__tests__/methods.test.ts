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
